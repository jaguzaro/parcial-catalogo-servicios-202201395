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
	"time"

	"catalogo/internal/auth"
)

type saludFalsa struct{ err error }

func (s saludFalsa) Salud(context.Context) error { return s.err }

var uiPrueba = fstest.MapFS{
	"index.html":    {Data: []byte("<html>inicio</html>")},
	"assets/app.js": {Data: []byte("console.log(1)")},
	"favicon.svg":   {Data: []byte("<svg/>")},
}

// nuevoSinBase arma el enrutador con un servicio de sesiones sin pool: sirve para los casos
// que se resuelven antes de llegar a la base.
func nuevoSinBase(salud Salud) http.Handler {
	return Nuevo(salud, auth.Nuevo(nil, time.Hour, false), uiPrueba)
}

func pedir(h http.Handler, metodo, ruta string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(metodo, ruta, nil))
	return rr
}

func pedirCon(h http.Handler, metodo, ruta, tipo, cuerpo string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	if tipo != "" {
		req.Header.Set("Content-Type", tipo)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func codigoError(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var cuerpo cuerpoError
	if err := json.Unmarshal(rr.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("el cuerpo no es JSON: %v (%s)", err, rr.Body.String())
	}
	return cuerpo.Error.Codigo
}

func TestAuthSinBase(t *testing.T) {
	h := nuevoSinBase(saludFalsa{})
	casos := []struct {
		nombre, metodo, ruta, tipo, cuerpo string
		estado                             int
		codigo                             string
	}{
		{"me sin cookie", "GET", "/api/auth/me", "", "", 401, "NO_AUTENTICADO"},
		{"logout sin cookie", "POST", "/api/auth/logout", "application/json", "{}", 401, "NO_AUTENTICADO"},
		{"logout sin cuerpo ni Content-Type y sin cookie", "POST", "/api/auth/logout", "", "", 401, "NO_AUTENTICADO"},
		{"login sin cuerpo ni Content-Type", "POST", "/api/auth/login", "", "", 400, "JSON_INVALIDO"},
		{"login sin Content-Type", "POST", "/api/auth/login", "", `{"login":"a","contrasena":"b"}`, 415, "TIPO_NO_SOPORTADO"},
		{"login como formulario", "POST", "/api/auth/login", "application/x-www-form-urlencoded", "login=a", 415, "TIPO_NO_SOPORTADO"},
		{"login JSON roto", "POST", "/api/auth/login", "application/json", `{"login":`, 400, "JSON_INVALIDO"},
		{"login campo desconocido", "POST", "/api/auth/login", "application/json", `{"login":"a","contrasena":"b","rol":"administrador"}`, 400, "JSON_INVALIDO"},
		{"login vacio", "POST", "/api/auth/login", "application/json; charset=utf-8", `{"login":"","contrasena":""}`, 422, "VALIDACION"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			rr := pedirCon(h, c.metodo, c.ruta, c.tipo, c.cuerpo)
			if rr.Code != c.estado {
				t.Fatalf("estado = %d, se esperaba %d (%s)", rr.Code, c.estado, rr.Body.String())
			}
			if got := codigoError(t, rr); got != c.codigo {
				t.Errorf("codigo = %s, se esperaba %s", got, c.codigo)
			}
		})
	}
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
			rr := pedir(nuevoSinBase(saludFalsa{c.err}), "GET", "/healthz")
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
	rr := pedir(nuevoSinBase(saludFalsa{}), "GET", "/api/no-existe")
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
	h := nuevoSinBase(saludFalsa{})
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
