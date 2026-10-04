package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NombreCookie es la cookie que lleva el token de sesion.
const NombreCookie = "catalogo_sesion"

// Roles.
const (
	RolAdministrador = "administrador"
	RolConsulta      = "consulta"
)

var (
	// ErrCredencialesInvalidas cubre usuario inexistente, usuario inactivo y contrasena
	// incorrecta, sin distinguirlos.
	ErrCredencialesInvalidas = errors.New("credenciales invalidas")
	// ErrNoAutenticado cubre sin token, token desconocido, vencido o revocado, y usuario inactivo.
	ErrNoAutenticado = errors.New("no autenticado")
)

// Unidad resume un nivel de la jerarquia organizacional.
type Unidad struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

// Jerarquia es la cadena organizacional de un usuario, desde su puesto hasta la empresa.
type Jerarquia struct {
	Puesto       Unidad `json:"puesto"`
	Seccion      Unidad `json:"seccion"`
	Departamento Unidad `json:"departamento"`
	Area         Unidad `json:"area"`
	Empresa      Unidad `json:"empresa"`
}

// Usuario es lo que se expone de un usuario autenticado. No tiene campo para el hash: no
// hay forma de que llegue a una respuesta.
type Usuario struct {
	ID        int64      `json:"id"`
	Nombre    string     `json:"nombre"`
	Usuario   string     `json:"usuario"`
	Correo    *string    `json:"correo"`
	Rol       string     `json:"rol"`
	Activo    bool       `json:"activo"`
	Jerarquia *Jerarquia `json:"jerarquia,omitempty"`
}

// Servicio valida credenciales y administra las sesiones contra la base.
type Servicio struct {
	pool         *pgxpool.Pool
	ttl          time.Duration
	cookieSecure bool
}

// Nuevo crea el servicio. ttl es la vida de la sesion; cookieSecure fija el atributo Secure.
func Nuevo(pool *pgxpool.Pool, ttl time.Duration, cookieSecure bool) *Servicio {
	return &Servicio{pool: pool, ttl: ttl, cookieSecure: cookieSecure}
}

// HashToken es el SHA-256 del token. Es lo unico que se guarda en la base.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func nuevoToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// IniciarSesion valida las credenciales y abre una sesion. Devuelve el usuario y el token,
// que solo viaja en la cookie.
func (s *Servicio) IniciarSesion(ctx context.Context, login, contrasena string) (*Usuario, string, error) {
	campo := "usuario"
	if strings.Contains(login, "@") {
		campo = "correo"
	}
	var (
		u    Usuario
		hash string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT id, nombre, usuario, correo, rol, activo, hash_contrasena
		   FROM usuario WHERE lower(`+campo+`) = lower($1)`, login).
		Scan(&u.ID, &u.Nombre, &u.Usuario, &u.Correo, &u.Rol, &u.Activo, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		verificarFicticio(contrasena)
		return nil, "", ErrCredencialesInvalidas
	}
	if err != nil {
		return nil, "", fmt.Errorf("buscar usuario: %w", err)
	}
	ok, err := VerificarContrasena(contrasena, hash)
	if err != nil {
		return nil, "", fmt.Errorf("verificar contrasena del usuario %d: %w", u.ID, err)
	}
	if !ok || !u.Activo {
		return nil, "", ErrCredencialesInvalidas
	}

	token, err := nuevoToken()
	if err != nil {
		return nil, "", err
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO sesion (usuario_id, token_hash, expira_en)
		 VALUES ($1, $2, now() + $3 * interval '1 second')`,
		u.ID, HashToken(token), int64(s.ttl/time.Second)); err != nil {
		return nil, "", fmt.Errorf("crear sesion: %w", err)
	}
	if u.Jerarquia, err = s.jerarquia(ctx, u.ID); err != nil {
		return nil, "", err
	}
	return &u, token, nil
}

// Autenticar resuelve el token de una peticion. El usuario y su rol se leen de la base en
// cada llamada. Si el usuario esta inactivo, la sesion se revoca.
func (s *Servicio) Autenticar(ctx context.Context, token string) (*Usuario, error) {
	if token == "" {
		return nil, ErrNoAutenticado
	}
	var (
		sesionID int64
		u        Usuario
	)
	err := s.pool.QueryRow(ctx,
		`SELECT s.id, u.id, u.nombre, u.usuario, u.correo, u.rol, u.activo
		   FROM sesion s JOIN usuario u ON u.id = s.usuario_id
		  WHERE s.token_hash = $1 AND s.revocada_en IS NULL AND s.expira_en > now()`,
		HashToken(token)).
		Scan(&sesionID, &u.ID, &u.Nombre, &u.Usuario, &u.Correo, &u.Rol, &u.Activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoAutenticado
	}
	if err != nil {
		return nil, fmt.Errorf("buscar sesion: %w", err)
	}
	if !u.Activo {
		if _, err := s.pool.Exec(ctx,
			`UPDATE sesion SET revocada_en = now() WHERE id = $1 AND revocada_en IS NULL`, sesionID); err != nil {
			return nil, fmt.Errorf("revocar sesion de usuario inactivo: %w", err)
		}
		return nil, ErrNoAutenticado
	}
	return &u, nil
}

// CerrarSesion revoca la sesion del token. Despues, el mismo token da ErrNoAutenticado.
func (s *Servicio) CerrarSesion(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sesion SET revocada_en = now() WHERE token_hash = $1 AND revocada_en IS NULL`, HashToken(token))
	if err != nil {
		return fmt.Errorf("revocar sesion: %w", err)
	}
	return nil
}

// Perfil devuelve el usuario con su jerarquia.
func (s *Servicio) Perfil(ctx context.Context, u *Usuario) (*Usuario, error) {
	j, err := s.jerarquia(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	p := *u
	p.Jerarquia = j
	return &p, nil
}

func (s *Servicio) jerarquia(ctx context.Context, usuarioID int64) (*Jerarquia, error) {
	var j Jerarquia
	err := s.pool.QueryRow(ctx,
		`SELECT p.id, p.codigo, p.nombre, s.id, s.codigo, s.nombre, d.id, d.codigo, d.nombre,
		        a.id, a.codigo, a.nombre, e.id, e.codigo, e.nombre
		   FROM usuario u
		   JOIN puesto p       ON p.id = u.puesto_id
		   JOIN seccion s      ON s.id = p.seccion_id
		   JOIN departamento d ON d.id = s.departamento_id
		   JOIN area a         ON a.id = d.area_id
		   JOIN empresa e      ON e.id = a.empresa_id
		  WHERE u.id = $1`, usuarioID).
		Scan(&j.Puesto.ID, &j.Puesto.Codigo, &j.Puesto.Nombre, &j.Seccion.ID, &j.Seccion.Codigo, &j.Seccion.Nombre,
			&j.Departamento.ID, &j.Departamento.Codigo, &j.Departamento.Nombre,
			&j.Area.ID, &j.Area.Codigo, &j.Area.Nombre, &j.Empresa.ID, &j.Empresa.Codigo, &j.Empresa.Nombre)
	if err != nil {
		return nil, fmt.Errorf("leer jerarquia del usuario %d: %w", usuarioID, err)
	}
	return &j, nil
}

// PonerCookie entrega el token en una cookie HttpOnly y SameSite=Lax.
func (s *Servicio) PonerCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     NombreCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.ttl / time.Second),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// BorrarCookie pide al navegador que descarte la cookie de sesion.
func (s *Servicio) BorrarCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     NombreCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// LeerToken devuelve el token de la cookie de la peticion, o "" si no viene.
func LeerToken(r *http.Request) string {
	c, err := r.Cookie(NombreCookie)
	if err != nil {
		return ""
	}
	return c.Value
}
