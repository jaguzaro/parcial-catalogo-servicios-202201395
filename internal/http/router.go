// Package http contiene el enrutador, los middlewares y el formato de error.
// No tiene reglas de negocio.
package http

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"catalogo/internal/auth"
	"catalogo/internal/catalogo"
	"catalogo/internal/organizacion"
	"catalogo/internal/usuarios"
)

// Salud comprueba que la base responde.
type Salud interface {
	Salud(ctx context.Context) error
}

// Servicios son las dependencias del enrutador.
type Servicios struct {
	Salud        Salud
	Sesiones     *auth.Servicio
	Organizacion *organizacion.Servicio
	Usuarios     *usuarios.Servicio
	Catalogo     *catalogo.Servicio
}

// Ruta es una ruta registrada de la API.
type Ruta struct {
	Metodo string
	Patron string
}

// registro registra rutas en el mux y las anota, para que las pruebas recorran las rutas
// reales y no una copia escrita a mano.
type registro struct {
	mux   *http.ServeMux
	rutas []Ruta
}

func (rg *registro) handle(patron string, h http.Handler) {
	metodo, ruta, ok := strings.Cut(patron, " ")
	if !ok {
		panic("ruta sin metodo: " + patron)
	}
	rg.mux.Handle(patron, h)
	rg.rutas = append(rg.rutas, Ruta{Metodo: metodo, Patron: ruta})
}

// Nuevo arma el enrutador: /healthz, /api/ y la interfaz embebida.
func Nuevo(s Servicios, ui fs.FS) http.Handler {
	api, _ := rutasAPI(s)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz(s.Salud))
	mux.Handle("/api/", exigirJSON(api))
	mux.Handle("/", interfaz(ui))
	return mux
}

// RutasAPI devuelve las rutas de la API tal como las registra Nuevo.
func RutasAPI() []Ruta {
	_, rutas := rutasAPI(Servicios{})
	return rutas
}

// rutasAPI registra todas las rutas de /api/. Lectura pide sesion; escritura pide sesion y
// rol administrador. Ver docs/diseno/reglas.md, "Autorizacion por rol".
func rutasAPI(s Servicios) (*http.ServeMux, []Ruta) {
	lectura := func(h http.HandlerFunc) http.Handler {
		return s.Sesiones.RequerirSesion(EscribirError, h)
	}
	escritura := func(h http.HandlerFunc) http.Handler {
		return s.Sesiones.RequerirSesion(EscribirError, auth.RequerirRol(EscribirError, h, auth.RolAdministrador))
	}

	rg := &registro{mux: http.NewServeMux()}
	rg.handle("POST /api/auth/login", login(s.Sesiones))
	rg.handle("POST /api/auth/logout", lectura(logout(s.Sesiones)))
	rg.handle("GET /api/auth/me", lectura(quienSoy(s.Sesiones)))

	org := s.Organizacion
	for _, n := range organizacion.Niveles {
		base := "/api/" + n.Recurso
		rg.handle("GET "+base, lectura(listarUnidades(org, n)))
		rg.handle("GET "+base+"/{id}", lectura(obtenerUnidad(org, n)))
		rg.handle("POST "+base, escritura(crearUnidad(org, n)))
		rg.handle("PUT "+base+"/{id}", escritura(actualizarUnidad(org, n)))
		rg.handle("POST "+base+"/{id}/desactivar", escritura(cambiarEstadoUnidad(n,
			func(r *http.Request, n *organizacion.Nivel, id int64) (*organizacion.Registro, error) {
				return org.Desactivar(r.Context(), n, id)
			})))
		rg.handle("POST "+base+"/{id}/activar", escritura(cambiarEstadoUnidad(n,
			func(r *http.Request, n *organizacion.Nivel, id int64) (*organizacion.Registro, error) {
				return org.Activar(r.Context(), n, id)
			})))
	}

	u := s.Usuarios
	rg.handle("GET /api/usuarios", lectura(listarUsuarios(u)))
	rg.handle("GET /api/usuarios/{id}", lectura(obtenerUsuario(u)))
	rg.handle("POST /api/usuarios", escritura(crearUsuario(u)))
	rg.handle("PUT /api/usuarios/{id}", escritura(actualizarUsuario(u)))
	rg.handle("PUT /api/usuarios/{id}/contrasena", escritura(fijarContrasena(u)))
	rg.handle("POST /api/usuarios/{id}/desactivar", escritura(desactivarUsuario(u)))
	rg.handle("POST /api/usuarios/{id}/activar", escritura(activarUsuario(u)))

	cat := s.Catalogo
	rg.handle("GET /api/catalogos/mapeo", lectura(mapeoOpciones(cat)))
	rg.handle("GET /api/catalogos/{c}", lectura(listarOpciones(cat)))
	rg.handle("POST /api/catalogos/{c}", escritura(crearOpcion(cat)))
	rg.handle("PUT /api/catalogos/{c}/{id}", escritura(actualizarOpcion(cat)))
	rg.handle("POST /api/catalogos/{c}/{id}/desactivar", escritura(cambiarEstadoOpcion(
		func(r *http.Request, c *catalogo.Catalogo, id int64) (*catalogo.Opcion, error) {
			return cat.DesactivarOpcion(r.Context(), c, id)
		})))
	rg.handle("POST /api/catalogos/{c}/{id}/activar", escritura(cambiarEstadoOpcion(
		func(r *http.Request, c *catalogo.Catalogo, id int64) (*catalogo.Opcion, error) {
			return cat.ActivarOpcion(r.Context(), c, id)
		})))

	rg.handle("GET /api/servicios-n1", lectura(listarN1(cat)))
	rg.handle("GET /api/servicios-n1/{id}", lectura(obtenerN1(cat)))
	rg.handle("POST /api/servicios-n1", escritura(crearN1(cat)))
	rg.handle("PUT /api/servicios-n1/{id}", escritura(actualizarN1(cat)))
	rg.handle("POST /api/servicios-n1/{id}/desactivar", escritura(cambiarEstado("desactivar servicio n1",
		func(r *http.Request, id int64) (*catalogo.ServicioN1, error) {
			return cat.DesactivarN1(r.Context(), id)
		})))
	rg.handle("POST /api/servicios-n1/{id}/activar", escritura(cambiarEstado("activar servicio n1",
		func(r *http.Request, id int64) (*catalogo.ServicioN1, error) { return cat.ActivarN1(r.Context(), id) })))

	rg.handle("GET /api/servicios", lectura(listarN2(cat)))
	rg.handle("GET /api/servicios/{id}", lectura(obtenerN2(cat)))
	rg.handle("POST /api/servicios", escritura(crearN2(cat)))
	rg.handle("PUT /api/servicios/{id}", escritura(actualizarN2(cat)))
	rg.handle("PUT /api/servicios/{id}/responsable", escritura(asignarResponsable(cat)))
	rg.handle("POST /api/servicios/{id}/desactivar", escritura(cambiarEstado("desactivar servicio",
		func(r *http.Request, id int64) (*catalogo.ServicioN2, error) {
			return cat.DesactivarN2(r.Context(), id)
		})))
	rg.handle("POST /api/servicios/{id}/activar", escritura(cambiarEstado("activar servicio",
		func(r *http.Request, id int64) (*catalogo.ServicioN2, error) { return cat.ActivarN2(r.Context(), id) })))

	rg.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		EscribirError(w, http.StatusNotFound, "NO_ENCONTRADO", "La ruta "+r.URL.Path+" no existe.")
	})
	return rg.mux, rg.rutas
}

func healthz(salud Salud) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := salud.Salud(ctx); err != nil {
			slog.Error("healthz: la base no responde", "error", err)
			EscribirJSON(w, http.StatusServiceUnavailable, map[string]string{"estado": "sin_base"})
			return
		}
		EscribirJSON(w, http.StatusOK, map[string]string{"estado": "ok"})
	}
}

// interfaz sirve los archivos embebidos. Si el archivo no existe y la ruta no tiene
// extension, devuelve index.html para que el enrutador del cliente resuelva la ruta.
func interfaz(ui fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
			return
		}
		nombre := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if nombre == "" {
			nombre = "index.html"
		}
		if info, err := fs.Stat(ui, nombre); err != nil || info.IsDir() {
			if path.Ext(nombre) != "" {
				http.NotFound(w, r)
				return
			}
			nombre = "index.html"
		}
		if nombre == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(nombre, "assets/") {
			w.Header().Set("Cache-Control", "max-age=31536000, immutable")
		}
		http.ServeFileFS(w, r, ui, nombre)
	})
}
