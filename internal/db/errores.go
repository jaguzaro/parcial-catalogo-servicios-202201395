package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Codigos de error de dominio. El paquete http los traduce a estado HTTP segun
// docs/diseno/arquitectura.md, "Formato de error".
const (
	ErrNoEncontrado          = "NO_ENCONTRADO"
	ErrValidacion            = "VALIDACION"
	ErrReferenciaInexistente = "REFERENCIA_INEXISTENTE"
	ErrCodigoDuplicado       = "CODIGO_DUPLICADO"
	ErrPadreInactivo         = "PADRE_INACTIVO"
	ErrDependenciasActivas   = "DEPENDENCIAS_ACTIVAS"
	ErrAsignacionesAfectadas = "ASIGNACIONES_AFECTADAS"
	ErrUltimoAdministrador   = "ULTIMO_ADMINISTRADOR"
	ErrServicioInactivo      = "SERVICIO_INACTIVO"
	ErrMinimoMayorQueMaximo  = "MINIMO_MAYOR_QUE_MAXIMO"
	ErrFueraDeSeccion        = "RESPONSABLE_FUERA_DE_SECCION"
)

// MaxDependencias es cuantos dependientes se listan como mucho en un error.
const MaxDependencias = 50

// Ref identifica un registro en un mensaje de error o una advertencia.
type Ref struct {
	Entidad string `json:"entidad"`
	ID      int64  `json:"id"`
	Codigo  string `json:"codigo"`
	Nombre  string `json:"nombre,omitempty"`
}

// Error es un error de regla de negocio. El mensaje es para el usuario: dice que paso y
// con que dato, sin detalles internos.
type Error struct {
	Codigo            string
	Mensaje           string
	Campos            []string
	Dependencias      []Ref
	TotalDependencias int
	Detalle           any
}

func (e *Error) Error() string { return e.Codigo + ": " + e.Mensaje }

// NuevoError arma un error de dominio.
func NuevoError(codigo, mensaje string, campos ...string) *Error {
	return &Error{Codigo: codigo, Mensaje: mensaje, Campos: campos}
}

// Validacion arma un error VALIDACION.
func Validacion(mensaje string, campos ...string) *Error {
	return NuevoError(ErrValidacion, mensaje, campos...)
}

// RestriccionViolada devuelve el SQLSTATE y el nombre de la restriccion si err viene de
// PostgreSQL. Si no, devuelve cadenas vacias.
func RestriccionViolada(err error) (sqlstate, restriccion string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

// EnTx ejecuta f dentro de una transaccion. Confirma si f no devuelve error y revierte si
// lo devuelve.
func EnTx(ctx context.Context, pool *pgxpool.Pool, f func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("abrir transaccion: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := f(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("confirmar transaccion: %w", err)
	}
	return nil
}
