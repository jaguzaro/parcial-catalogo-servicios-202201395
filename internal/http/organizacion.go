package http

import (
	"fmt"
	"net/http"

	"catalogo/internal/organizacion"
)

// cuerpoUnidad acepta el campo padre de cualquier nivel; despues se comprueba que solo
// venga el del nivel que corresponde.
type cuerpoUnidad struct {
	Codigo         *string `json:"codigo"`
	Nombre         *string `json:"nombre"`
	EmpresaID      *int64  `json:"empresa_id"`
	AreaID         *int64  `json:"area_id"`
	DepartamentoID *int64  `json:"departamento_id"`
	SeccionID      *int64  `json:"seccion_id"`
}

func (c cuerpoUnidad) padres() map[string]*int64 {
	return map[string]*int64{
		"empresa_id": c.EmpresaID, "area_id": c.AreaID, "departamento_id": c.DepartamentoID, "seccion_id": c.SeccionID,
	}
}

// leerUnidad decodifica el cuerpo de alta o modificacion. Si falla, ya respondio.
func leerUnidad(w http.ResponseWriter, r *http.Request, n *organizacion.Nivel) (organizacion.Entrada, bool) {
	var c cuerpoUnidad
	if !leerJSON(w, r, &c) {
		return organizacion.Entrada{}, false
	}
	var e organizacion.Entrada
	for campo, v := range c.padres() {
		if v == nil {
			continue
		}
		if campo != n.CampoPadre {
			EscribirError(w, http.StatusBadRequest, "JSON_INVALIDO",
				fmt.Sprintf("El campo %s no corresponde a %s.", campo, n.Plural), campo)
			return e, false
		}
		e.PadreID = *v
	}
	if c.Codigo != nil {
		e.Codigo = *c.Codigo
	}
	if c.Nombre != nil {
		e.Nombre = *c.Nombre
	}
	return e, true
}

func listarUnidades(org *organizacion.Servicio, n *organizacion.Nivel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			f   organizacion.Filtro
			err error
		)
		if f.Pagina, err = leerPagina(r); err != nil {
			responderError(w, "", err)
			return
		}
		if f.Activo, err = leerBool(r, "activo"); err != nil {
			responderError(w, "", err)
			return
		}
		if n.Padre != nil {
			if f.PadreID, err = leerIDConsulta(r, n.CampoPadre); err != nil {
				responderError(w, "", err)
				return
			}
		}
		f.Q = r.URL.Query().Get("q")
		listado, err := org.Listar(r.Context(), n, f)
		if err != nil {
			responderError(w, "listar "+n.Plural, err)
			return
		}
		EscribirJSON(w, http.StatusOK, listado)
	}
}

func obtenerUnidad(org *organizacion.Servicio, n *organizacion.Nivel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		reg, err := org.Obtener(r.Context(), n, id)
		if err != nil {
			responderError(w, "obtener "+n.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, reg)
	}
}

func crearUnidad(org *organizacion.Servicio, n *organizacion.Nivel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, ok := leerUnidad(w, r, n)
		if !ok {
			return
		}
		reg, err := org.Crear(r.Context(), n, e)
		if err != nil {
			responderError(w, "crear "+n.Tabla, err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/%s/%d", n.Recurso, reg.ID))
		EscribirJSON(w, http.StatusCreated, reg)
	}
}

func actualizarUnidad(org *organizacion.Servicio, n *organizacion.Nivel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		e, ok := leerUnidad(w, r, n)
		if !ok {
			return
		}
		reg, err := org.Actualizar(r.Context(), n, id, e)
		if err != nil {
			responderError(w, "actualizar "+n.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, reg)
	}
}

// cambiarEstadoUnidad atiende desactivar y activar.
func cambiarEstadoUnidad(n *organizacion.Nivel, op func(*http.Request, *organizacion.Nivel, int64) (*organizacion.Registro, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok || !leerCuerpoVacio(w, r) {
			return
		}
		reg, err := op(r, n, id)
		if err != nil {
			responderError(w, "cambiar estado de "+n.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, reg)
	}
}
