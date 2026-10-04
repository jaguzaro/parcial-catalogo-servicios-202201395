package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

// FuncError escribe una respuesta de error con el formato unico de la API. La pasa el
// paquete http, para que este paquete no dependa de el.
type FuncError func(w http.ResponseWriter, estado int, codigo, mensaje string, campos ...string)

type claveUsuario struct{}

// UsuarioDe devuelve el usuario autenticado que dejo RequerirSesion en el contexto.
func UsuarioDe(ctx context.Context) (*Usuario, bool) {
	u, ok := ctx.Value(claveUsuario{}).(*Usuario)
	return u, ok
}

// ConUsuario guarda el usuario autenticado en el contexto.
func ConUsuario(ctx context.Context, u *Usuario) context.Context {
	return context.WithValue(ctx, claveUsuario{}, u)
}

// RequerirSesion deja pasar solo peticiones con una sesion valida de un usuario activo.
// Si no la hay responde 401 NO_AUTENTICADO y borra la cookie.
func (s *Servicio) RequerirSesion(fallar FuncError, sig http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := LeerToken(r)
		u, err := s.Autenticar(r.Context(), token)
		if err != nil {
			if !errors.Is(err, ErrNoAutenticado) {
				slog.Error("autenticar peticion", "error", err)
				fallar(w, http.StatusInternalServerError, "INTERNO", "Error interno.")
				return
			}
			if token != "" {
				s.BorrarCookie(w)
			}
			fallar(w, http.StatusUnauthorized, "NO_AUTENTICADO", "Debe iniciar sesion.")
			return
		}
		sig.ServeHTTP(w, r.WithContext(ConUsuario(r.Context(), u)))
	})
}

// RequerirRol deja pasar solo a los roles dados; al resto responde 403 PROHIBIDO. Va
// siempre dentro de RequerirSesion: sin sesion la respuesta es 401, no 403.
func RequerirRol(fallar FuncError, sig http.Handler, roles ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UsuarioDe(r.Context())
		if !ok {
			fallar(w, http.StatusUnauthorized, "NO_AUTENTICADO", "Debe iniciar sesion.")
			return
		}
		for _, rol := range roles {
			if u.Rol == rol {
				sig.ServeHTTP(w, r)
				return
			}
		}
		fallar(w, http.StatusForbidden, "PROHIBIDO", "Su rol no permite esta operacion.")
	})
}
