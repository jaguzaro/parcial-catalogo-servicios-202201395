// Package catalogo mantiene los servicios de nivel 1 y nivel 2, los catalogos de clase,
// criticidad y tipo, y la asignacion de responsable. Las reglas estan en
// docs/diseno/reglas.md: codigo unico por nivel, minimo no mayor que maximo sin convertir
// ausencias en cero, nada colgando de un padre inactivo, baja rechazada mientras haya
// dependientes activos y un usuario responsable que pertenezca a la seccion responsable.
package catalogo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"catalogo/internal/db"
)

// Limites de longitud, en caracteres. Los fija este paquete, igual que organizacion: el
// diseno no los da. Solo se aplican a lo que llega por la API; los datos importados no se
// recortan.
const (
	LargoCodigo = 50
	LargoNombre = 200
	LargoTexto  = 4000
)

// Conexion es lo que el servicio necesita de la base: un *pgxpool.Pool, o una pgx.Tx en
// las pruebas y en sembrar-demo. Cada escritura abre su propia transaccion con Begin, que
// sobre una pgx.Tx es un punto de guardado.
type Conexion interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// consultor es lo comun entre la conexion y una transaccion.
type consultor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Servicio aplica las reglas del catalogo.
type Servicio struct {
	conn Conexion
}

// Nuevo crea el servicio.
func Nuevo(conn Conexion) *Servicio { return &Servicio{conn: conn} }

func (s *Servicio) enTx(ctx context.Context, f func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.conn, f)
}

// Ref resume un registro relacionado: el nivel 1 de un servicio, por ejemplo.
type Ref struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
	Activo *bool  `json:"activo,omitempty"`
}

// ---------- campos del cuerpo ----------

// Valor es un campo del cuerpo JSON que distingue ausente, null y un valor. En una
// modificacion, ausente deja el dato como estaba y null lo borra. Asi un dato que no
// llego nunca se convierte en cero ni en vacio.
type Valor struct {
	Presente bool
	JSON     json.RawMessage
}

// UnmarshalJSON solo se llama si el campo viene, tambien con null.
func (v *Valor) UnmarshalJSON(b []byte) error {
	v.Presente = true
	v.JSON = append(v.JSON[:0], b...)
	return nil
}

func (v Valor) nulo() bool { return string(bytes.TrimSpace(v.JSON)) == "null" }

// errores acumula los campos con problemas para responder un solo 422.
type errores struct {
	campos   []string
	mensajes []string
}

func (e *errores) falla(campo, mensaje string) {
	e.campos = append(e.campos, campo)
	e.mensajes = append(e.mensajes, mensaje)
}

func (e *errores) err() error {
	if len(e.campos) == 0 {
		return nil
	}
	return db.Validacion(strings.Join(e.mensajes, " "), e.campos...)
}

// texto lee un campo de texto. null es nil.
func (e *errores) texto(v Valor, campo string) (*string, bool) {
	if v.nulo() {
		return nil, true
	}
	var s string
	if err := json.Unmarshal(v.JSON, &s); err != nil {
		e.falla(campo, fmt.Sprintf("%s debe ser texto o null.", campo))
		return nil, false
	}
	return &s, true
}

// codigo valida un codigo: obligatorio, sin espacios al inicio ni al final (no se recorta
// en silencio) y sin caracteres de control.
func (e *errores) codigo(v Valor) (string, bool) {
	s, ok := e.texto(v, "codigo")
	switch {
	case !ok:
		return "", false
	case s == nil || *s == "" || utf8.RuneCountInString(*s) > LargoCodigo:
		e.falla("codigo", fmt.Sprintf("El codigo es obligatorio y admite hasta %d caracteres.", LargoCodigo))
	case strings.TrimSpace(*s) != *s:
		e.falla("codigo", fmt.Sprintf("El codigo %q tiene espacios al inicio o al final.", *s))
	case strings.ContainsFunc(*s, unicode.IsControl):
		e.falla("codigo", "El codigo tiene caracteres de control.")
	default:
		return *s, true
	}
	return "", false
}

// nombre valida un texto obligatorio con largo maximo.
func (e *errores) nombre(v Valor, campo string, largo int) (string, bool) {
	s, ok := e.texto(v, campo)
	if !ok {
		return "", false
	}
	if s == nil || strings.TrimSpace(*s) == "" || utf8.RuneCountInString(*s) > largo {
		e.falla(campo, fmt.Sprintf("%s es obligatorio y admite hasta %d caracteres.", mayuscula(campo), largo))
		return "", false
	}
	return *s, true
}

// opcional valida un texto opcional. Vacio o solo espacios se guarda como null: un texto
// vacio no es un dato.
func (e *errores) opcional(v Valor, campo string) (*string, bool) {
	s, ok := e.texto(v, campo)
	if !ok || s == nil {
		return nil, ok
	}
	if utf8.RuneCountInString(*s) > LargoTexto {
		e.falla(campo, fmt.Sprintf("%s admite hasta %d caracteres.", mayuscula(campo), LargoTexto))
		return nil, false
	}
	if strings.TrimSpace(*s) == "" {
		return nil, true
	}
	return s, true
}

// id lee una referencia: un entero positivo o null.
func (e *errores) id(v Valor, campo string) (*int64, bool) {
	if v.nulo() {
		return nil, true
	}
	var n int64
	if err := json.Unmarshal(v.JSON, &n); err != nil || n < 1 {
		e.falla(campo, fmt.Sprintf("%s debe ser un id numerico o null.", campo))
		return nil, false
	}
	return &n, true
}

// numero lee minimo o maximo. null es nil, que no es cero. Una cadena, incluso vacia o con
// un numero dentro, se rechaza: el campo es numerico.
func (e *errores) numero(v Valor, campo string) (*string, bool) {
	if v.nulo() {
		return nil, true
	}
	crudo := string(bytes.TrimSpace(v.JSON))
	var n json.Number
	if crudo == "" || crudo[0] == '"' || json.Unmarshal(v.JSON, &n) != nil {
		e.falla(campo, fmt.Sprintf("%s debe ser un numero o null; llego %s. Un valor vacio no se toma como cero.", campo, crudo))
		return nil, false
	}
	r, ok := new(big.Rat).SetString(n.String())
	if !ok {
		e.falla(campo, fmt.Sprintf("%s debe ser un numero o null; llego %s.", campo, crudo))
		return nil, false
	}
	if r.Sign() < 0 {
		e.falla(campo, fmt.Sprintf("%s no admite negativos; llego %s.", campo, crudo))
		return nil, false
	}
	s := n.String()
	return &s, true
}

// booleano lee true o false. null no se acepta: el campo no admite nulo.
func (e *errores) booleano(v Valor, campo string) (bool, bool) {
	var b bool
	if v.nulo() || json.Unmarshal(v.JSON, &b) != nil {
		e.falla(campo, fmt.Sprintf("%s debe ser true o false.", campo))
		return false, false
	}
	return b, true
}

// ---------- ayudas ----------

func mayuscula(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// cantidad escribe "1 servicio" o "3 servicios".
func cantidad(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// listarRefs corre una consulta que devuelve id, codigo y nombre, y trae como mucho
// MaxDependencias filas junto con el total.
func listarRefs(ctx context.Context, c consultor, entidad, consulta string, id int64) ([]db.Ref, int, error) {
	filas, err := c.Query(ctx, fmt.Sprintf(`SELECT id, codigo, nombre, count(*) OVER () FROM (%s) d ORDER BY codigo, id LIMIT %d`,
		consulta, db.MaxDependencias), id)
	if err != nil {
		return nil, 0, fmt.Errorf("listar dependientes (%s): %w", entidad, err)
	}
	defer filas.Close()
	var (
		refs  []db.Ref
		total int
	)
	for filas.Next() {
		ref := db.Ref{Entidad: entidad}
		if err := filas.Scan(&ref.ID, &ref.Codigo, &ref.Nombre, &total); err != nil {
			return nil, 0, fmt.Errorf("leer dependientes (%s): %w", entidad, err)
		}
		refs = append(refs, ref)
	}
	if err := filas.Err(); err != nil {
		return nil, 0, fmt.Errorf("listar dependientes (%s): %w", entidad, err)
	}
	return refs, total, nil
}

func dependenciasActivas(mensaje string, refs []db.Ref, total int) error {
	e := db.NuevoError(db.ErrDependenciasActivas, mensaje)
	e.Dependencias, e.TotalDependencias = refs, total
	return e
}

// rawONulo convierte un jsonb leido como texto en JSON crudo; null si no hay.
func rawONulo(s *string) json.RawMessage {
	if s == nil {
		return nil
	}
	return json.RawMessage(*s)
}
