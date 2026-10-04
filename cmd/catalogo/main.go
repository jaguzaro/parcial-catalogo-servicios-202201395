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
	"syscall"
	"time"

	"catalogo/internal/config"
	"catalogo/internal/db"
	apihttp "catalogo/internal/http"
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
	case "importar", "sembrar-demo":
		fmt.Fprintf(os.Stderr, "el subcomando %q todavia no esta implementado\n", os.Args[1])
		os.Exit(3)
	default:
		uso()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
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

	srv := &http.Server{
		Addr:              direccion,
		Handler:           apihttp.Nuevo(pool, web.Archivos()),
		ReadHeaderTimeout: 5 * time.Second,
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
