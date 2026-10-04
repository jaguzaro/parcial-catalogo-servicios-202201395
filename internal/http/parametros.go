package http

import (
	"fmt"
	"net/http"
	"strconv"

	"catalogo/internal/db"
)

const (
	tamanoPorDefecto = 20
	tamanoMaximo     = 100
)

// leerPagina lee pagina y tamano de la consulta. Un valor fuera de rango es 422.
func leerPagina(r *http.Request) (db.Pagina, error) {
	p := db.Pagina{Numero: 1, Tamano: tamanoPorDefecto}
	if v := r.URL.Query().Get("pagina"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return p, db.Validacion(fmt.Sprintf("pagina debe ser un entero desde 1; llego %q.", v), "pagina")
		}
		p.Numero = n
	}
	if v := r.URL.Query().Get("tamano"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > tamanoMaximo {
			return p, db.Validacion(fmt.Sprintf("tamano debe ser un entero entre 1 y %d; llego %q.", tamanoMaximo, v), "tamano")
		}
		p.Tamano = n
	}
	return p, nil
}

// leerBool lee un filtro true/false. Ausente es nil.
func leerBool(r *http.Request, nombre string) (*bool, error) {
	v := r.URL.Query().Get(nombre)
	switch v {
	case "":
		return nil, nil
	case "true":
		b := true
		return &b, nil
	case "false":
		b := false
		return &b, nil
	}
	return nil, db.Validacion(fmt.Sprintf("%s debe ser true o false; llego %q.", nombre, v), nombre)
}

// leerIDConsulta lee un filtro por id. Ausente es nil.
func leerIDConsulta(r *http.Request, nombre string) (*int64, error) {
	v := r.URL.Query().Get(nombre)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 1 {
		return nil, db.Validacion(fmt.Sprintf("%s debe ser un id numerico; llego %q.", nombre, v), nombre)
	}
	return &n, nil
}

// idRuta lee el {id} de la ruta. Si no es un numero valido responde 404, igual que un id
// que no existe, y devuelve false.
func idRuta(w http.ResponseWriter, r *http.Request) (int64, bool) {
	v := r.PathValue("id")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 1 {
		EscribirError(w, http.StatusNotFound, "NO_ENCONTRADO", fmt.Sprintf("El id %q de la ruta no es valido.", v))
		return 0, false
	}
	return n, true
}

// leerCuerpoVacio acepta un cuerpo ausente o un objeto vacio, como el {} de activar y
// desactivar. Si falla, ya respondio 400.
func leerCuerpoVacio(w http.ResponseWriter, r *http.Request) bool {
	if r.ContentLength == 0 {
		return true
	}
	var nada struct{}
	return leerJSON(w, r, &nada)
}
