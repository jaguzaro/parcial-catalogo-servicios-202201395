package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"unicode/utf8"

	"catalogo/internal/auth"
)

const maxCuerpo = 1 << 20

// exigirJSON rechaza con 415 los POST y PUT que traen cuerpo sin declarar
// Content-Type: application/json. Sin cuerpo no hay nada que interpretar, y asi cerrar
// sesion funciona como lo llama un navegador. La defensa contra POST de otro origen es
// SameSite=Lax en la cookie.
func exigirJSON(sig http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ContentLength es -1 cuando el largo no se conoce (chunked): eso cuenta como cuerpo.
		traeCuerpo := r.ContentLength != 0
		if traeCuerpo && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
			tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || tipo != "application/json" {
				EscribirError(w, http.StatusUnsupportedMediaType, "TIPO_NO_SOPORTADO",
					"El cuerpo debe enviarse con Content-Type: application/json.")
				return
			}
		}
		sig.ServeHTTP(w, r)
	})
}

// leerJSON decodifica el cuerpo rechazando campos desconocidos. Si falla, ya respondio 400.
func leerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		EscribirError(w, http.StatusBadRequest, "JSON_INVALIDO", "El cuerpo no es JSON valido o trae campos desconocidos.")
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		EscribirError(w, http.StatusBadRequest, "JSON_INVALIDO", "El cuerpo debe tener un solo objeto JSON.")
		return false
	}
	return true
}

func errorInterno(w http.ResponseWriter, contexto string, err error) {
	slog.Error(contexto, "error", err)
	EscribirError(w, http.StatusInternalServerError, "INTERNO", "Error interno.")
}

type respuestaUsuario struct {
	Usuario *auth.Usuario `json:"usuario"`
}

func login(sesiones *auth.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cuerpo struct {
			Login      string `json:"login"`
			Contrasena string `json:"contrasena"`
		}
		if !leerJSON(w, r, &cuerpo) {
			return
		}
		var campos []string
		if cuerpo.Login == "" || utf8.RuneCountInString(cuerpo.Login) > 254 {
			campos = append(campos, "login")
		}
		if cuerpo.Contrasena == "" || utf8.RuneCountInString(cuerpo.Contrasena) > auth.LargoMaximo {
			campos = append(campos, "contrasena")
		}
		if len(campos) > 0 {
			EscribirError(w, http.StatusUnprocessableEntity, "VALIDACION",
				"Indique usuario o correo y contrasena (hasta 256 caracteres).", campos...)
			return
		}

		u, token, err := sesiones.IniciarSesion(r.Context(), cuerpo.Login, cuerpo.Contrasena)
		if errors.Is(err, auth.ErrCredencialesInvalidas) {
			EscribirError(w, http.StatusUnauthorized, "CREDENCIALES_INVALIDAS", "Usuario o contrasena incorrectos.")
			return
		}
		if err != nil {
			errorInterno(w, "iniciar sesion", err)
			return
		}
		sesiones.PonerCookie(w, token)
		EscribirJSON(w, http.StatusOK, respuestaUsuario{u})
	}
}

func logout(sesiones *auth.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sesiones.CerrarSesion(r.Context(), auth.LeerToken(r)); err != nil {
			errorInterno(w, "cerrar sesion", err)
			return
		}
		sesiones.BorrarCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}

func quienSoy(sesiones *auth.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, _ := auth.UsuarioDe(r.Context())
		perfil, err := sesiones.Perfil(r.Context(), u)
		if err != nil {
			errorInterno(w, "leer perfil", err)
			return
		}
		EscribirJSON(w, http.StatusOK, respuestaUsuario{perfil})
	}
}
