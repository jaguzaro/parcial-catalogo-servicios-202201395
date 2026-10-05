// Comando catalogo: un solo binario con los subcomandos servir, migrar, importar,
// sembrar-demo y salud.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"catalogo/internal/auth"
	"catalogo/internal/catalogo"
	"catalogo/internal/config"
	"catalogo/internal/db"
	"catalogo/internal/demo"
	apihttp "catalogo/internal/http"
	"catalogo/internal/importador"
	"catalogo/internal/organizacion"
	"catalogo/internal/usuarios"
	"catalogo/internal/web"
)

const (
	direccion = ":8080"
	urlSalud  = "http://127.0.0.1:8080/healthz"
)

func main() {
	if len(os.Args) != 2 {
		uso()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "servir":
		err = servir()
	case "migrar":
		err = migrar()
	case "salud":
		err = salud()
	case "sembrar-demo":
		err = sembrarDemo()
	case "importar":
		err = importar()
	default:
		uso()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if errors.Is(err, importador.ErrEnCurso) {
			os.Exit(2) // docs/diseno/arquitectura.md: 2 = IMPORTACION_EN_CURSO
		}
		os.Exit(1)
	}
}

func uso() {
	fmt.Fprintln(os.Stderr, "uso: catalogo <servir|migrar|importar|sembrar-demo|salud>")
}

func cargarConfig() (*config.Config, error) {
	return config.Cargar(os.LookupEnv)
}

func migrar() error {
	cfg, err := cargarConfig()
	if err != nil {
		return err
	}
	return aplicarMigraciones(context.Background(), cfg.DatabaseURL)
}

func aplicarMigraciones(ctx context.Context, url string) error {
	antes, despues, err := db.Migrar(ctx, url)
	if err != nil {
		return err
	}
	if antes == despues {
		fmt.Printf("migraciones: la base ya estaba en la version %d, nada que aplicar\n", despues)
	} else {
		fmt.Printf("migraciones: version %d -> %d\n", antes, despues)
	}
	return nil
}

func servir() error {
	cfg, err := cargarConfig()
	if err != nil {
		return err
	}
	ctx, parar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer parar()

	if err := aplicarMigraciones(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := db.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if cfg.AutoInicio {
		autoInicio(ctx, cfg, pool.Pool)
	} else {
		slog.Info("arranque automatico desactivado (AUTO_INICIO=false): no se importa ni se siembra")
	}

	servicios := apihttp.Servicios{
		Salud:         pool,
		Sesiones:      auth.Nuevo(pool.Pool, cfg.SessionTTL, cfg.CookieSecure),
		Organizacion:  organizacion.Nuevo(pool.Pool),
		Usuarios:      usuarios.Nuevo(pool.Pool),
		Catalogo:      catalogo.Nuevo(pool.Pool),
		Importaciones: importador.NuevoLector(pool.Pool),
	}
	// Los tiempos cubren la peticion completa. Lo mas lento es el login: argon2id con 64 MiB
	// tarda del orden de 100 ms, y unos segundos si hay varios a la vez. Los cuerpos admitidos
	// son de hasta 1 MiB, asi que estos margenes sobran. ReadHeaderTimeout: un cliente
	// legitimo envia las cabeceras de inmediato. ReadTimeout: cabeceras y cuerpo.
	// WriteTimeout: incluye el hash de argon2id bajo carga. IdleTimeout: conexiones
	// keep-alive sin uso.
	srv := &http.Server{
		Addr:              direccion,
		Handler:           apihttp.Nuevo(servicios, web.Archivos()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	slog.Info("escuchando", "direccion", direccion)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	slog.Info("apagando")
	cierre, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := srv.Shutdown(cierre); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// autoInicio importa el Excel y despues siembra la demostracion, en ese orden: el sembrado
// asigna responsables sobre servicios ya importados. Ambas operaciones son idempotentes.
// Nunca devuelve error: si algo falla se registra y el servidor arranca igual.
func autoInicio(ctx context.Context, cfg *config.Config, pool demo.Conexion) {
	resumen, err := importador.Importar(ctx, pool, cfg.ExcelPath)
	switch {
	case err != nil:
		slog.Error("arranque automatico: la importacion fallo; se sigue sin importar", "error", err)
	case resumen.Creados == 0 && resumen.Actualizados == 0:
		slog.Info("arranque automatico: importacion sin cambios, el catalogo ya estaba cargado",
			"importacion", resumen.ImportacionID, "omitidos", resumen.Omitidos)
	default:
		slog.Info("arranque automatico: importacion aplicada", "importacion", resumen.ImportacionID,
			"creados", resumen.Creados, "actualizados", resumen.Actualizados, "omitidos", resumen.Omitidos)
	}

	cuentas, err := demo.CargarCuentas(os.LookupEnv)
	if err != nil {
		slog.Error("arranque automatico: no se puede sembrar la demostracion", "error", err)
		return
	}
	var salida strings.Builder
	err = demo.Sembrar(ctx, pool, cuentas, &salida)
	for _, linea := range strings.Split(salida.String(), "\n") {
		if linea = strings.TrimSpace(linea); linea != "" {
			slog.Info("arranque automatico: sembrar-demo: " + linea)
		}
	}
	if err != nil {
		slog.Error("arranque automatico: el sembrado fallo; se sigue sin sembrar", "error", err)
		return
	}
	slog.Info("arranque automatico: sembrado terminado")
}

// sembrarDemo crea la estructura DEMO y las cuentas de evaluacion. Las migraciones se
// aplican antes, para que funcione tambien sobre una base recien creada.
func sembrarDemo() error {
	cfg, err := cargarConfig()
	if err != nil {
		return err
	}
	cuentas, err := demo.CargarCuentas(os.LookupEnv)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := aplicarMigraciones(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := db.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := demo.Sembrar(ctx, pool.Pool, cuentas, os.Stdout); err != nil {
		return err
	}
	fmt.Println("sembrar-demo: terminado")
	return nil
}

// importar lee el Excel de EXCEL_PATH y lo lleva a la base. Las migraciones se aplican
// antes, igual que en sembrar-demo.
func importar() error {
	cfg, err := cargarConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := aplicarMigraciones(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := db.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	resumen, err := importador.Importar(ctx, pool.Pool, cfg.ExcelPath)
	if err != nil {
		return err
	}
	resumen.Escribir(os.Stdout)
	return nil
}

func salud() error {
	cliente := &http.Client{Timeout: 3 * time.Second}
	resp, err := cliente.Get(urlSalud)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/healthz respondio %d", resp.StatusCode)
	}
	return nil
}
