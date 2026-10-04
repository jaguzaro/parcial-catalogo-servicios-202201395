package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"catalogo/internal/db"
)

// cuerpoError es el formato unico de error de docs/diseno/arquitectura.md.
type cuerpoError struct {
	Error detalleError `json:"error"`
}

type detalleError struct {
	Codigo            string   `json:"codigo"`
	Mensaje           string   `json:"mensaje"`
	Campos            []string `json:"campos,omitempty"`
	Dependencias      []db.Ref `json:"dependencias,omitempty"`
	TotalDependencias int      `json:"total_dependencias,omitempty"`
	Detalle           any      `json:"detalle,omitempty"`
}

// estadoPorCodigo traduce los errores de dominio a estado HTTP.
var estadoPorCodigo = map[string]int{
	db.ErrNoEncontrado:          http.StatusNotFound,
	db.ErrValidacion:            http.StatusUnprocessableEntity,
	db.ErrReferenciaInexistente: http.StatusUnprocessableEntity,
	db.ErrCodigoDuplicado:       http.StatusConflict,
	db.ErrPadreInactivo:         http.StatusConflict,
	db.ErrDependenciasActivas:   http.StatusConflict,
	db.ErrAsignacionesAfectadas: http.StatusConflict,
	db.ErrUltimoAdministrador:   http.StatusConflict,
}

// EscribirJSON responde con un cuerpo JSON y el estado dado.
func EscribirJSON(w http.ResponseWriter, estado int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	if err := json.NewEncoder(w).Encode(cuerpo); err != nil {
		slog.Error("escribir respuesta JSON", "error", err)
	}
}

// EscribirError responde con el formato de error unico.
func EscribirError(w http.ResponseWriter, estado int, codigo, mensaje string, campos ...string) {
	EscribirJSON(w, estado, cuerpoError{Error: detalleError{Codigo: codigo, Mensaje: mensaje, Campos: campos}})
}

// responderError escribe un error de dominio con su estado. Cualquier otro error es
// interno: el detalle va al log y la respuesta es generica.
func responderError(w http.ResponseWriter, contexto string, err error) {
	var e *db.Error
	if errors.As(err, &e) {
		if estado, ok := estadoPorCodigo[e.Codigo]; ok {
			EscribirJSON(w, estado, cuerpoError{Error: detalleError{
				Codigo:            e.Codigo,
				Mensaje:           e.Mensaje,
				Campos:            e.Campos,
				Dependencias:      e.Dependencias,
				TotalDependencias: e.TotalDependencias,
				Detalle:           e.Detalle,
			}})
			return
		}
	}
	errorInterno(w, contexto, err)
}
