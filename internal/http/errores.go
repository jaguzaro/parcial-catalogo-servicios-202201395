package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// cuerpoError es el formato unico de error de docs/diseno/arquitectura.md.
type cuerpoError struct {
	Error detalleError `json:"error"`
}

type detalleError struct {
	Codigo  string   `json:"codigo"`
	Mensaje string   `json:"mensaje"`
	Campos  []string `json:"campos,omitempty"`
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
