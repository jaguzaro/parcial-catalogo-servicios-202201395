package http

import (
	"fmt"
	"net/http"

	"catalogo/internal/usuarios"
)

// cuerpoAltaUsuario es el cuerpo de POST /api/usuarios.
type cuerpoAltaUsuario struct {
	Nombre     string  `json:"nombre"`
	Usuario    string  `json:"usuario"`
	Correo     *string `json:"correo"`
	Contrasena string  `json:"contrasena"`
	Rol        string  `json:"rol"`
	PuestoID   int64   `json:"puesto_id"`
}

// cuerpoEdicionUsuario es el de PUT /api/usuarios/{id}: sin contrasena, que tiene su ruta.
type cuerpoEdicionUsuario struct {
	Nombre   string  `json:"nombre"`
	Usuario  string  `json:"usuario"`
	Correo   *string `json:"correo"`
	Rol      string  `json:"rol"`
	PuestoID int64   `json:"puesto_id"`
}

type respuestaBaja struct {
	Usuario      *usuarios.Usuario      `json:"usuario"`
	Advertencias []usuarios.Advertencia `json:"advertencias"`
}

func listarUsuarios(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			f   usuarios.Filtro
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
		if f.PuestoID, err = leerIDConsulta(r, "puesto_id"); err != nil {
			responderError(w, "", err)
			return
		}
		if f.SeccionID, err = leerIDConsulta(r, "seccion_id"); err != nil {
			responderError(w, "", err)
			return
		}
		f.Q = r.URL.Query().Get("q")
		f.Rol = r.URL.Query().Get("rol")
		listado, err := svc.Listar(r.Context(), f)
		if err != nil {
			responderError(w, "listar usuarios", err)
			return
		}
		EscribirJSON(w, http.StatusOK, listado)
	}
}

func obtenerUsuario(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		u, err := svc.Obtener(r.Context(), id)
		if err != nil {
			responderError(w, "obtener usuario", err)
			return
		}
		EscribirJSON(w, http.StatusOK, u)
	}
}

func crearUsuario(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c cuerpoAltaUsuario
		if !leerJSON(w, r, &c) {
			return
		}
		u, err := svc.Crear(r.Context(), usuarios.Entrada{
			Nombre: c.Nombre, Usuario: c.Usuario, Correo: c.Correo, Contrasena: c.Contrasena, Rol: c.Rol, PuestoID: c.PuestoID,
		})
		if err != nil {
			responderError(w, "crear usuario", err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf("/api/usuarios/%d", u.ID))
		EscribirJSON(w, http.StatusCreated, u)
	}
}

func actualizarUsuario(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var c cuerpoEdicionUsuario
		if !leerJSON(w, r, &c) {
			return
		}
		u, err := svc.Actualizar(r.Context(), id, usuarios.Entrada{
			Nombre: c.Nombre, Usuario: c.Usuario, Correo: c.Correo, Rol: c.Rol, PuestoID: c.PuestoID,
		})
		if err != nil {
			responderError(w, "actualizar usuario", err)
			return
		}
		EscribirJSON(w, http.StatusOK, u)
	}
}

func fijarContrasena(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok {
			return
		}
		var c struct {
			Contrasena string `json:"contrasena"`
		}
		if !leerJSON(w, r, &c) {
			return
		}
		if err := svc.FijarContrasena(r.Context(), id, c.Contrasena); err != nil {
			responderError(w, "fijar contrasena", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func desactivarUsuario(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok || !leerCuerpoVacio(w, r) {
			return
		}
		u, adv, err := svc.Desactivar(r.Context(), id)
		if err != nil {
			responderError(w, "desactivar usuario", err)
			return
		}
		EscribirJSON(w, http.StatusOK, respuestaBaja{Usuario: u, Advertencias: adv})
	}
}

func activarUsuario(svc *usuarios.Servicio) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := idRuta(w, r)
		if !ok || !leerCuerpoVacio(w, r) {
			return
		}
		u, err := svc.Activar(r.Context(), id)
		if err != nil {
			responderError(w, "activar usuario", err)
			return
		}
		EscribirJSON(w, http.StatusOK, u)
	}
}
