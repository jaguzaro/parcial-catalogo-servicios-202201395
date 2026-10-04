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
)

// Salud comprueba que la base responde.
type Salud interface {
	Salud(ctx context.Context) error
}

// Nuevo arma el enrutador: /healthz, /api/ y la interfaz embebida.
func Nuevo(salud Salud, sesiones *auth.Servicio, ui fs.FS) http.Handler {
	conSesion := func(h http.HandlerFunc) http.Handler {
		return sesiones.RequerirSesion(EscribirError, h)
	}

	api := http.NewServeMux()
	api.Handle("POST /api/auth/login", login(sesiones))
	api.Handle("POST /api/auth/logout", conSesion(logout(sesiones)))
	api.Handle("GET /api/auth/me", conSesion(quienSoy(sesiones)))
	api.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		EscribirError(w, http.StatusNotFound, "NO_ENCONTRADO", "La ruta "+r.URL.Path+" no existe.")
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz(salud))
	mux.Handle("/api/", exigirJSON(api))
	mux.Handle("/", interfaz(ui))
	return mux
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
