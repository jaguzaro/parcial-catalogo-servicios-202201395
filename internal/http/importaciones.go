package http

import (
	"net/http"

	"catalogo/internal/importador"
)

func listarImportaciones(svc *importador.Lector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := leerPagina(r)
		if err != nil {
			responderError(w, "", err)
			return
		}
		listado, err := svc.Listar(r.Context(), p)
		if err != nil {
			responderError(w, "listar importaciones", err)
			return
		}
		EscribirJSON(w, http.StatusOK, listado)
	}
}

func obtenerImportacion(svc *importador.Lector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		c, err := svc.Obtener(r.Context(), id)
		if err != nil {
			responderError(w, "obtener importacion", err)
			return
		}
		EscribirJSON(w, http.StatusOK, c)
	}
}
