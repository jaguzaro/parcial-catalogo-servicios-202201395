package auth

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHashContrasena_FormatoYVerificacion(t *testing.T) {
	clave := rand.Text()
	h, err := HashContrasena(clave)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatalf("formato inesperado: %s", h)
	}
	if ok, err := VerificarContrasena(clave, h); err != nil || !ok {
		t.Fatalf("la contrasena correcta no verifica: ok=%v err=%v", ok, err)
	}
	if ok, err := VerificarContrasena(clave+"x", h); err != nil || ok {
		t.Fatalf("una contrasena distinta verifica: ok=%v err=%v", ok, err)
	}
}

func TestHashContrasena_SalDistintaCadaVez(t *testing.T) {
	clave := rand.Text()
	a, _ := HashContrasena(clave)
	b, _ := HashContrasena(clave)
	if a == b {
		t.Fatal("dos hashes de la misma contrasena son iguales: la sal no es aleatoria")
	}
}

func TestHashContrasena_Largo(t *testing.T) {
	for _, c := range []string{"", "1234567", strings.Repeat("a", LargoMaximo+1)} {
		if _, err := HashContrasena(c); err != ErrLargoContrasena {
			t.Errorf("largo %d: err = %v, se esperaba ErrLargoContrasena", len(c), err)
		}
	}
	// Se cuentan caracteres, no bytes.
	if _, err := HashContrasena(strings.Repeat("ñ", LargoMaximo)); err != nil {
		t.Errorf("%d caracteres de dos bytes: %v", LargoMaximo, err)
	}
}

func TestVerificarContrasena_HashMalFormado(t *testing.T) {
	for _, h := range []string{"", "texto-plano", "$2a$10$bcrypt", "$argon2id$v=19$m=65536,t=3,p=4$$", "$argon2i$v=19$m=65536,t=3,p=4$YWJj$YWJj"} {
		if ok, err := VerificarContrasena("cualquiera", h); err == nil || ok {
			t.Errorf("%q: ok=%v err=%v, se esperaba error", h, ok, err)
		}
	}
}

func TestRequerirRol(t *testing.T) {
	var codigo string
	fallar := func(w http.ResponseWriter, estado int, c, _ string, _ ...string) {
		codigo = c
		w.WriteHeader(estado)
	}
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RequerirRol(fallar, siguiente, RolAdministrador)

	casos := []struct {
		nombre string
		u      *Usuario
		estado int
		codigo string
	}{
		{"sin usuario", nil, 401, "NO_AUTENTICADO"},
		{"consulta", &Usuario{ID: 1, Rol: RolConsulta}, 403, "PROHIBIDO"},
		{"administrador", &Usuario{ID: 2, Rol: RolAdministrador}, 200, ""},
	}
	for _, c := range casos {
		codigo = ""
		req := httptest.NewRequest("POST", "/api/algo", nil)
		if c.u != nil {
			req = req.WithContext(ConUsuario(req.Context(), c.u))
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != c.estado || codigo != c.codigo {
			t.Errorf("%s: estado=%d codigo=%q, se esperaba %d %q", c.nombre, rr.Code, codigo, c.estado, c.codigo)
		}
	}
}
