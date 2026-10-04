package http

import (
	"fmt"
	"net/http"

	"catalogo/internal/catalogo"
)

// ---------- cuerpos ----------

// Los campos son catalogo.Valor para distinguir ausente, null y valor: un dato que no
// llega no se convierte en cero.

type cuerpoN1 struct {
	Codigo catalogo.Valor `json:"codigo"`
	Nombre catalogo.Valor `json:"nombre"`
}

type cuerpoOpcion struct {
	Nombre catalogo.Valor `json:"nombre"`
	Orden  catalogo.Valor `json:"orden"`
}

// cuerpoAltaN2 es el de POST /api/servicios: sin requiere_revision ni responsable.
type cuerpoAltaN2 struct {
	ServicioN1ID catalogo.Valor `json:"servicio_n1_id"`
	Codigo       catalogo.Valor `json:"codigo"`
	Nombre       catalogo.Valor `json:"nombre"`
	Activo       catalogo.Valor `json:"activo"`
	ClaseID      catalogo.Valor `json:"clase_id"`
	CriticidadID catalogo.Valor `json:"criticidad_id"`
	TipoID       catalogo.Valor `json:"tipo_id"`
	Descripcion  catalogo.Valor `json:"descripcion"`
	Metrica      catalogo.Valor `json:"metrica"`
	Minimo       catalogo.Valor `json:"minimo"`
	Maximo       catalogo.Valor `json:"maximo"`
}

func (c cuerpoAltaN2) entrada() catalogo.EntradaN2 {
	return catalogo.EntradaN2{
		ServicioN1ID: c.ServicioN1ID, Codigo: c.Codigo, Nombre: c.Nombre, Activo: c.Activo,
		ClaseID: c.ClaseID, CriticidadID: c.CriticidadID, TipoID: c.TipoID,
		Descripcion: c.Descripcion, Metrica: c.Metrica, Minimo: c.Minimo, Maximo: c.Maximo,
	}
}

// cuerpoEdicionN2 es el de PUT /api/servicios/{id}: lo mismo mas requiere_revision. El
// responsable tiene su propia ruta.
type cuerpoEdicionN2 struct {
	cuerpoAltaN2
	RequiereRevision catalogo.Valor `json:"requiere_revision"`
}

type cuerpoResponsable struct {
	SeccionID *int64 `json:"seccion_id"`
	UsuarioID *int64 `json:"usuario_id"`
}

// ---------- catalogos de opciones ----------

// catalogoRuta lee el {c} de la ruta. Si no existe responde 404 y devuelve false.
func catalogoRuta(w http.ResponseWriter, r *http.Request) (*catalogo.Catalogo, bool) {
	c, ok := catalogo.CatalogoDe(r.PathValue("c"))
	if !ok {
		EscribirError(w, http.StatusNotFound, "NO_ENCONTRADO",
			fmt.Sprintf("El catalogo %q no existe. Use clases, criticidades o tipos.", r.PathValue("c")))
	}
	return c, ok
}

func listarOpciones(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := catalogoRuta(w, r)
		if !ok {
			return
		}
		activo, err := leerBool(r, "activo")
		if err != nil {
			responderError(w, "", err)
			return
		}
		items, err := svc.ListarOpciones(r.Context(), c, activo)
		if err != nil {
			responderError(w, "listar "+c.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	}
}

func mapeoOpciones(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.Mapeo(r.Context())
		if err != nil {
			responderError(w, "mapeo de catalogos", err)
			return
		}
		EscribirJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
	}
}

func crearOpcion(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := catalogoRuta(w, r)
		if !ok {
			return
		}
		var b cuerpoOpcion
		if !leerJSON(w, r, &b) {
			return
		}
		o, err := svc.CrearOpcion(r.Context(), c, catalogo.EntradaOpcion{Nombre: b.Nombre, Orden: b.Orden})
		if err != nil {
			responderError(w, "crear "+c.Tabla, err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/catalogos/%s/%d", c.Recurso, o.ID))
		EscribirJSON(w, http.StatusCreated, o)
	}
}

func actualizarOpcion(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := catalogoRuta(w, r)
		if !ok {
			return
		}
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var b cuerpoOpcion
		if !leerJSON(w, r, &b) {
			return
		}
		o, err := svc.ActualizarOpcion(r.Context(), c, id, catalogo.EntradaOpcion{Nombre: b.Nombre, Orden: b.Orden})
		if err != nil {
			responderError(w, "actualizar "+c.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, o)
	}
}

func cambiarEstadoOpcion(op func(*http.Request, *catalogo.Catalogo, int64) (*catalogo.Opcion, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := catalogoRuta(w, r)
		if !ok {
			return
		}
		id, ok := idRuta(w, r)
		if !ok || !leerCuerpoVacio(w, r) {
			return
		}
		o, err := op(r, c, id)
		if err != nil {
			responderError(w, "cambiar estado de "+c.Tabla, err)
			return
		}
		EscribirJSON(w, http.StatusOK, o)
	}
}

// ---------- servicios de nivel 1 ----------

func listarN1(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			f   catalogo.FiltroN1
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
		f.Q = r.URL.Query().Get("q")
		listado, err := svc.ListarN1(r.Context(), f)
		if err != nil {
			responderError(w, "listar servicios n1", err)
			return
		}
		EscribirJSON(w, http.StatusOK, listado)
	}
}

func obtenerN1(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		n, err := svc.ObtenerN1(r.Context(), id)
		if err != nil {
			responderError(w, "obtener servicio n1", err)
			return
		}
		EscribirJSON(w, http.StatusOK, n)
	}
}

func crearN1(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b cuerpoN1
		if !leerJSON(w, r, &b) {
			return
		}
		n, err := svc.CrearN1(r.Context(), catalogo.EntradaN1{Codigo: b.Codigo, Nombre: b.Nombre})
		if err != nil {
			responderError(w, "crear servicio n1", err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/servicios-n1/%d", n.ID))
		EscribirJSON(w, http.StatusCreated, n)
	}
}

func actualizarN1(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var b cuerpoN1
		if !leerJSON(w, r, &b) {
			return
		}
		n, err := svc.ActualizarN1(r.Context(), id, catalogo.EntradaN1{Codigo: b.Codigo, Nombre: b.Nombre})
		if err != nil {
			responderError(w, "actualizar servicio n1", err)
			return
		}
		EscribirJSON(w, http.StatusOK, n)
	}
}

// cambiarEstado atiende desactivar y activar de los servicios de los dos niveles.
func cambiarEstado[T any](contexto string, op func(*http.Request, int64) (T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok || !leerCuerpoVacio(w, r) {
			return
		}
		v, err := op(r, id)
		if err != nil {
			responderError(w, contexto, err)
			return
		}
		EscribirJSON(w, http.StatusOK, v)
	}
}

// ---------- servicios de nivel 2 ----------

func listarN2(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			f   catalogo.FiltroN2
			err error
		)
		if f.Pagina, err = leerPagina(r); err != nil {
			responderError(w, "", err)
			return
		}
		for _, c := range []struct {
			nombre string
			dest   **int64
		}{
			{"n1_id", &f.N1ID}, {"clase_id", &f.ClaseID}, {"criticidad_id", &f.CriticidadID},
			{"tipo_id", &f.TipoID}, {"seccion_id", &f.SeccionID},
		} {
			if *c.dest, err = leerIDConsulta(r, c.nombre); err != nil {
				responderError(w, "", err)
				return
			}
		}
		if f.RequiereRevision, err = leerBool(r, "requiere_revision"); err != nil {
			responderError(w, "", err)
			return
		}
		f.Q = r.URL.Query().Get("q")
		f.Activo = r.URL.Query().Get("activo")
		listado, err := svc.ListarN2(r.Context(), f)
		if err != nil {
			responderError(w, "listar servicios", err)
			return
		}
		EscribirJSON(w, http.StatusOK, listado)
	}
}

func obtenerN2(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		sv, err := svc.ObtenerN2(r.Context(), id)
		if err != nil {
			responderError(w, "obtener servicio", err)
			return
		}
		EscribirJSON(w, http.StatusOK, sv)
	}
}

func crearN2(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b cuerpoAltaN2
		if !leerJSON(w, r, &b) {
			return
		}
		sv, err := svc.CrearN2(r.Context(), b.entrada())
		if err != nil {
			responderError(w, "crear servicio", err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/servicios/%d", sv.ID))
		EscribirJSON(w, http.StatusCreated, sv)
	}
}

func actualizarN2(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var b cuerpoEdicionN2
		if !leerJSON(w, r, &b) {
			return
		}
		e := b.entrada()
		e.RequiereRevision = b.RequiereRevision
		sv, err := svc.ActualizarN2(r.Context(), id, e)
		if err != nil {
			responderError(w, "actualizar servicio", err)
			return
		}
		EscribirJSON(w, http.StatusOK, sv)
	}
}

func asignarResponsable(svc *catalogo.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var b cuerpoResponsable
		if !leerJSON(w, r, &b) {
			return
		}
		sv, err := svc.AsignarResponsable(r.Context(), id, b.SeccionID, b.UsuarioID)
		if err != nil {
			responderError(w, "asignar responsable", err)
			return
		}
		EscribirJSON(w, http.StatusOK, sv)
	}
}
