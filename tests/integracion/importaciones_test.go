package integracion

import (
	"fmt"
	"testing"
)

// GET /api/importaciones y GET /api/importaciones/{id} piden solo sesion. El listado trae
// el resumen de cada corrida y la ficha trae todas sus incidencias.
func TestImportaciones_ListarYVer(t *testing.T) {
	e := nuevoCatalogoImportado(t)

	esperarEstado(t, pedirAPI(t, e.srv, "GET", "/api/importaciones", nil, ""), 401)

	r := e.pedir(t, "GET", "/api/importaciones", nil)
	esperarEstado(t, r, 200)
	lista := decodificar[struct {
		Items []struct {
			ID         int64  `json:"id"`
			Estado     string `json:"estado"`
			Creados    int    `json:"creados"`
			Observados int    `json:"observados"`
		} `json:"items"`
		Total int `json:"total"`
	}](t, r)
	if lista.Total < 1 || len(lista.Items) < 1 {
		t.Fatalf("el listado no trae la corrida recien hecha: %s", r.cuerpo)
	}
	primera := lista.Items[0]
	if primera.Estado != "completada" || primera.Creados == 0 || primera.Observados == 0 {
		t.Errorf("resumen inesperado: %+v", primera)
	}

	r = e.pedir(t, "GET", fmt.Sprintf("/api/importaciones/%d", primera.ID), nil)
	esperarEstado(t, r, 200)
	ficha := decodificar[struct {
		ID          int64 `json:"id"`
		Incidencias []struct {
			Tipo string `json:"tipo"`
		} `json:"incidencias"`
	}](t, r)
	if ficha.ID != primera.ID || len(ficha.Incidencias) == 0 {
		t.Errorf("la ficha no trae sus incidencias: %s", r.cuerpo)
	}

	esperarEstado(t, e.pedir(t, "GET", "/api/importaciones/999999999", nil), 404)
}
