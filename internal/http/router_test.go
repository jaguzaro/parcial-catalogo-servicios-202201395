package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type saludFalsa struct{ err error }

func (s saludFalsa) Salud(context.Context) error { return s.err }

var uiPrueba = fstest.MapFS{
	"index.html":    {Data: []byte("<html>inicio</html>")},
	"assets/app.js": {Data: []byte("console.log(1)")},
	"favicon.svg":   {Data: []byte("<svg/>")},
}

func pedir(h http.Handler, metodo, ruta string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(metodo, ruta, nil))
	return rr
}

func TestHealthz(t *testing.T) {
	casos := []struct {
		nombre string
		err    error
		estado int
		cuerpo string
	}{
		{"base arriba", nil, 200, `{"estado":"ok"}`},
		{"base caida", errors.New("conexion rechazada"), 503, `{"estado":"sin_base"}`},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			rr := pedir(Nuevo(saludFalsa{c.err}, uiPrueba), "GET", "/healthz")
			if rr.Code != c.estado {
				t.Errorf("estado = %d, se esperaba %d", rr.Code, c.estado)
			}
			if got := strings.TrimSpace(rr.Body.String()); got != c.cuerpo {
				t.Errorf("cuerpo = %s, se esperaba %s", got, c.cuerpo)
			}
		})
	}
}

func TestApiDesconocidaDa404JSON(t *testing.T) {
	rr := pedir(Nuevo(saludFalsa{}, uiPrueba), "GET", "/api/no-existe")
	if rr.Code != 404 {
		t.Fatalf("estado = %d, se esperaba 404", rr.Code)
	}
	var cuerpo cuerpoError
	if err := json.Unmarshal(rr.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("el cuerpo no es JSON: %v (%s)", err, rr.Body.String())
	}
	if cuerpo.Error.Codigo != "NO_ENCONTRADO" || cuerpo.Error.Mensaje == "" {
		t.Errorf("error mal formado: %+v", cuerpo)
	}
}

func TestInterfaz(t *testing.T) {
	h := Nuevo(saludFalsa{}, uiPrueba)
	casos := []struct {
		ruta, contiene, cache string
		estado                int
	}{
		{"/", "inicio", "no-cache", 200},
		{"/servicios/12", "inicio", "no-cache", 200},
		{"/assets/app.js", "console.log", "max-age=31536000, immutable", 200},
		{"/favicon.svg", "<svg", "", 200},
		{"/assets/falta.js", "", "", 404},
	}
	for _, c := range casos {
		rr := pedir(h, "GET", c.ruta)
		if rr.Code != c.estado {
			t.Errorf("%s: estado = %d, se esperaba %d", c.ruta, rr.Code, c.estado)
			continue
		}
		if !strings.Contains(rr.Body.String(), c.contiene) {
			t.Errorf("%s: el cuerpo no contiene %q", c.ruta, c.contiene)
		}
		if got := rr.Header().Get("Cache-Control"); got != c.cache {
			t.Errorf("%s: Cache-Control = %q, se esperaba %q", c.ruta, got, c.cache)
		}
	}
	if rr := pedir(h, "POST", "/"); rr.Code != 405 {
		t.Errorf("POST /: estado = %d, se esperaba 405", rr.Code)
	}
}
