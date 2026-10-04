package db

import "strings"

// Pagina es la paginacion pedida en un listado. Ya viene validada por la capa http.
type Pagina struct {
	Numero int
	Tamano int
}

// Desplazamiento es el OFFSET de la pagina.
func (p Pagina) Desplazamiento() int { return (p.Numero - 1) * p.Tamano }

// Listado es la respuesta de todo GET de coleccion.
type Listado[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Pagina int `json:"pagina"`
	Tamano int `json:"tamano"`
}

// NuevoListado arma la respuesta. items nunca sale como null.
func NuevoListado[T any](items []T, total int, p Pagina) Listado[T] {
	if items == nil {
		items = []T{}
	}
	return Listado[T]{Items: items, Total: total, Pagina: p.Numero, Tamano: p.Tamano}
}

var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// PatronContiene arma el patron de ILIKE para "contiene q", escapando los comodines.
func PatronContiene(q string) string {
	return "%" + escapeLike.Replace(q) + "%"
}
