package integracion

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"catalogo/internal/auth"
	"catalogo/internal/catalogo"
	"catalogo/internal/db"
	apihttp "catalogo/internal/http"
	"catalogo/internal/importador"
	"catalogo/internal/organizacion"
	"catalogo/internal/usuarios"
)

// ---------- ayudas ----------

type usuarioPrueba struct {
	id         int64
	usuario    string
	correo     string
	contrasena string
}

func sufijo(t *testing.T) string {
	t.Helper()
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// crearUsuario inserta una cadena organizacional propia y un usuario con la contrasena
// dada, y los borra al terminar la prueba. Los datos se confirman porque el servidor los
// lee desde otra conexion.
func crearUsuario(t *testing.T, rol, contrasena string) usuarioPrueba {
	t.Helper()
	ctx := context.Background()
	suf := sufijo(t)
	hash, err := auth.HashContrasena(contrasena)
	if err != nil {
		t.Fatal(err)
	}
	u := usuarioPrueba{usuario: "u" + suf, correo: "u" + suf + "@prueba.test", contrasena: contrasena}
	codigo := "T" + suf
	var empresa, area, depto, seccion, puesto int64
	pasos := []struct {
		sql  string
		args []any
		dest *int64
	}{
		{`INSERT INTO empresa (codigo, nombre) VALUES ($1, 'Empresa de prueba') RETURNING id`, []any{codigo}, &empresa},
		{`INSERT INTO area (empresa_id, codigo, nombre) VALUES ($1, $2, 'Area de prueba') RETURNING id`, []any{&empresa, codigo}, &area},
		{`INSERT INTO departamento (area_id, codigo, nombre) VALUES ($1, $2, 'Departamento de prueba') RETURNING id`, []any{&area, codigo}, &depto},
		{`INSERT INTO seccion (departamento_id, codigo, nombre) VALUES ($1, $2, 'Seccion de prueba') RETURNING id`, []any{&depto, codigo}, &seccion},
		{`INSERT INTO puesto (seccion_id, codigo, nombre) VALUES ($1, $2, 'Puesto de prueba') RETURNING id`, []any{&seccion, codigo}, &puesto},
		{`INSERT INTO usuario (puesto_id, nombre, usuario, correo, hash_contrasena, rol)
		  VALUES ($1, 'Usuario de prueba', $2, $3, $4, $5) RETURNING id`, []any{&puesto, u.usuario, u.correo, hash, rol}, &u.id},
	}
	for _, p := range pasos {
		args := make([]any, len(p.args))
		for i, a := range p.args {
			if ptr, ok := a.(*int64); ok {
				a = *ptr
			}
			args[i] = a
		}
		if err := pool.QueryRow(ctx, p.sql, args...).Scan(p.dest); err != nil {
			t.Fatalf("preparacion fallo: %v\nSQL: %s", err, p.sql)
		}
	}
	t.Cleanup(func() {
		for _, q := range []struct {
			sql string
			id  int64
		}{
			{`DELETE FROM sesion WHERE usuario_id = $1`, u.id},
			{`DELETE FROM usuario WHERE id = $1`, u.id},
			{`DELETE FROM puesto WHERE id = $1`, puesto},
			{`DELETE FROM seccion WHERE id = $1`, seccion},
			{`DELETE FROM departamento WHERE id = $1`, depto},
			{`DELETE FROM area WHERE id = $1`, area},
			{`DELETE FROM empresa WHERE id = $1`, empresa},
		} {
			if _, err := pool.Exec(ctx, q.sql, q.id); err != nil {
				t.Errorf("limpieza: %v (%s)", err, q.sql)
			}
		}
	})
	return u
}

func servidor(t *testing.T) *httptest.Server {
	t.Helper()
	return servidorCatalogo(t, pool)
}

// servidorCatalogo arma el servidor con el catalogo sobre conn. Con una pgx.Tx, todo lo
// que el catalogo lee y escribe queda en esa transaccion y se revierte al final; sesiones,
// organizacion y usuarios siguen en el pool. Las peticiones de una prueba son
// secuenciales, asi que la transaccion nunca se usa desde dos peticiones a la vez.
func servidorCatalogo(t *testing.T, conn catalogo.Conexion) *httptest.Server {
	t.Helper()
	ui := fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}
	srv := httptest.NewServer(apihttp.Nuevo(apihttp.Servicios{
		Salud:         &db.Pool{Pool: pool},
		Sesiones:      auth.Nuevo(pool, time.Hour, false),
		Organizacion:  organizacion.Nuevo(pool),
		Usuarios:      usuarios.Nuevo(pool),
		Catalogo:      catalogo.Nuevo(conn),
		Importaciones: importador.NuevoLector(conn),
	}, ui))
	t.Cleanup(srv.Close)
	return srv
}

type respuesta struct {
	estado int
	cuerpo []byte
	cookie *http.Cookie // la cookie de sesion de Set-Cookie, si vino
}

// pedirAPI hace la peticion sin tarro de cookies: cada prueba decide que cookie manda.
func pedirAPI(t *testing.T, srv *httptest.Server, metodo, ruta string, cuerpo any, token string) respuesta {
	t.Helper()
	var r io.Reader
	if cuerpo != nil {
		b, err := json.Marshal(cuerpo)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metodo, srv.URL+ruta, r)
	if err != nil {
		t.Fatal(err)
	}
	if cuerpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.NombreCookie, Value: token})
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	out := respuesta{estado: resp.StatusCode, cuerpo: b}
	for _, c := range resp.Cookies() {
		if c.Name == auth.NombreCookie {
			out.cookie = c
		}
	}
	return out
}

func iniciar(t *testing.T, srv *httptest.Server, login, contrasena string) respuesta {
	t.Helper()
	return pedirAPI(t, srv, "POST", "/api/auth/login", map[string]string{"login": login, "contrasena": contrasena}, "")
}

// iniciarOK inicia sesion y devuelve el token de la cookie.
func iniciarOK(t *testing.T, srv *httptest.Server, u usuarioPrueba) string {
	t.Helper()
	r := iniciar(t, srv, u.usuario, u.contrasena)
	if r.estado != 200 || r.cookie == nil || r.cookie.Value == "" {
		t.Fatalf("login de %s: estado %d, cookie %v, cuerpo %s", u.usuario, r.estado, r.cookie, r.cuerpo)
	}
	return r.cookie.Value
}

func esperarError(t *testing.T, r respuesta, estado int, codigo string) {
	t.Helper()
	if r.estado != estado {
		t.Fatalf("estado = %d, se esperaba %d; cuerpo %s", r.estado, estado, r.cuerpo)
	}
	var e struct {
		Error struct {
			Codigo  string `json:"codigo"`
			Mensaje string `json:"mensaje"`
		} `json:"error"`
	}
	if err := json.Unmarshal(r.cuerpo, &e); err != nil {
		t.Fatalf("el cuerpo no es JSON: %v (%s)", err, r.cuerpo)
	}
	if e.Error.Codigo != codigo || e.Error.Mensaje == "" {
		t.Fatalf("error = %+v, se esperaba codigo %s con mensaje", e.Error, codigo)
	}
}

type cuerpoUsuario struct {
	Usuario struct {
		ID        int64  `json:"id"`
		Usuario   string `json:"usuario"`
		Rol       string `json:"rol"`
		Activo    bool   `json:"activo"`
		Jerarquia struct {
			Empresa struct {
				Codigo string `json:"codigo"`
			} `json:"empresa"`
		} `json:"jerarquia"`
	} `json:"usuario"`
}

func leerUsuario(t *testing.T, r respuesta) cuerpoUsuario {
	t.Helper()
	var c cuerpoUsuario
	if err := json.Unmarshal(r.cuerpo, &c); err != nil {
		t.Fatalf("el cuerpo no es JSON: %v (%s)", err, r.cuerpo)
	}
	return c
}

func contrasenaPrueba() string { return rand.Text() }

// ---------- P01 ----------

func TestP01_InicioDeSesionValidoEInvalido(t *testing.T) {
	srv := servidor(t)
	u := crearUsuario(t, auth.RolAdministrador, contrasenaPrueba())

	t.Run("valido por usuario", func(t *testing.T) {
		r := iniciar(t, srv, u.usuario, u.contrasena)
		if r.estado != 200 {
			t.Fatalf("estado = %d, cuerpo %s", r.estado, r.cuerpo)
		}
		c := r.cookie
		if c == nil || c.Value == "" {
			t.Fatal("no vino la cookie de sesion")
		}
		if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 3600 || c.Secure {
			t.Errorf("atributos de la cookie: HttpOnly=%v SameSite=%v Path=%q MaxAge=%d Secure=%v",
				c.HttpOnly, c.SameSite, c.Path, c.MaxAge, c.Secure)
		}
		b := leerUsuario(t, r)
		if b.Usuario.ID != u.id || b.Usuario.Usuario != u.usuario || b.Usuario.Rol != auth.RolAdministrador ||
			b.Usuario.Jerarquia.Empresa.Codigo == "" {
			t.Errorf("cuerpo inesperado: %s", r.cuerpo)
		}

		// En la base esta el SHA-256 del token y no el token.
		var n int
		if err := pool.QueryRow(context.Background(),
			`SELECT count(*) FROM sesion WHERE usuario_id = $1 AND token_hash = $2 AND revocada_en IS NULL
			   AND expira_en > now()`, u.id, auth.HashToken(c.Value)).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("sesiones con sha256(token) = %d, se esperaba 1", n)
		}
		if err := pool.QueryRow(context.Background(),
			`SELECT count(*) FROM sesion WHERE token_hash = $1`, []byte(c.Value)).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Error("el token aparece tal cual en la tabla sesion")
		}
	})

	t.Run("valido por correo y sin distinguir mayusculas", func(t *testing.T) {
		for _, login := range []string{u.correo, strings.ToUpper(u.correo), strings.ToUpper(u.usuario)} {
			if r := iniciar(t, srv, login, u.contrasena); r.estado != 200 || r.cookie == nil {
				t.Errorf("login %q: estado %d, cuerpo %s", login, r.estado, r.cuerpo)
			}
		}
	})

	t.Run("invalido", func(t *testing.T) {
		malaClave := iniciar(t, srv, u.usuario, u.contrasena+"x")
		esperarError(t, malaClave, 401, "CREDENCIALES_INVALIDAS")
		if malaClave.cookie != nil {
			t.Error("un login fallido entrego cookie de sesion")
		}

		// Usuario inexistente y contrasena incorrecta responden exactamente lo mismo.
		for _, login := range []string{"no-existe-" + sufijo(t), "nadie-" + sufijo(t) + "@prueba.test"} {
			r := iniciar(t, srv, login, u.contrasena)
			esperarError(t, r, 401, "CREDENCIALES_INVALIDAS")
			if !bytes.Equal(r.cuerpo, malaClave.cuerpo) {
				t.Errorf("usuario inexistente %q: cuerpo %s distinto de contrasena incorrecta %s", login, r.cuerpo, malaClave.cuerpo)
			}
		}
	})
}

// ---------- P02 ----------

func TestP02_OperacionesProtegidasRechazadas(t *testing.T) {
	srv := servidor(t)
	ctx := context.Background()

	t.Run("sin sesion", func(t *testing.T) {
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, ""), 401, "NO_AUTENTICADO")
		esperarError(t, pedirAPI(t, srv, "POST", "/api/auth/logout", map[string]string{}, ""), 401, "NO_AUTENTICADO")

		r := pedirAPI(t, srv, "GET", "/api/auth/me", nil, "token-inventado")
		esperarError(t, r, 401, "NO_AUTENTICADO")
		if r.cookie == nil || r.cookie.MaxAge >= 0 {
			t.Errorf("una cookie invalida debe borrarse; Set-Cookie = %v", r.cookie)
		}
	})

	t.Run("misma cookie despues de cerrar sesion", func(t *testing.T) {
		u := crearUsuario(t, auth.RolConsulta, contrasenaPrueba())
		token := iniciarOK(t, srv, u)
		if r := pedirAPI(t, srv, "GET", "/api/auth/me", nil, token); r.estado != 200 {
			t.Fatalf("me con sesion: estado %d, cuerpo %s", r.estado, r.cuerpo)
		}

		r := pedirAPI(t, srv, "POST", "/api/auth/logout", map[string]string{}, token)
		if r.estado != 204 {
			t.Fatalf("logout: estado %d, cuerpo %s", r.estado, r.cuerpo)
		}
		if r.cookie == nil || r.cookie.MaxAge >= 0 {
			t.Errorf("logout debe borrar la cookie; Set-Cookie = %v", r.cookie)
		}

		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
		esperarError(t, pedirAPI(t, srv, "POST", "/api/auth/logout", map[string]string{}, token), 401, "NO_AUTENTICADO")

		var revocada bool
		if err := pool.QueryRow(ctx, `SELECT revocada_en IS NOT NULL FROM sesion WHERE token_hash = $1`,
			auth.HashToken(token)).Scan(&revocada); err != nil {
			t.Fatal(err)
		}
		if !revocada {
			t.Error("la sesion no quedo revocada en la base")
		}
	})

	t.Run("cerrar sesion como lo llama un navegador, sin cuerpo ni Content-Type", func(t *testing.T) {
		u := crearUsuario(t, auth.RolConsulta, contrasenaPrueba())
		token := iniciarOK(t, srv, u)

		// cuerpo nil: pedirAPI no pone Content-Type, igual que fetch(url, {method:'POST'}).
		r := pedirAPI(t, srv, "POST", "/api/auth/logout", nil, token)
		if r.estado != 204 {
			t.Fatalf("logout sin cuerpo: estado %d, cuerpo %s", r.estado, r.cuerpo)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")

		var revocada bool
		if err := pool.QueryRow(ctx, `SELECT revocada_en IS NOT NULL FROM sesion WHERE token_hash = $1`,
			auth.HashToken(token)).Scan(&revocada); err != nil {
			t.Fatal(err)
		}
		if !revocada {
			t.Error("la sesion no quedo revocada en la base")
		}
	})

	t.Run("un cuerpo con Content-Type que no es JSON sigue dando 415", func(t *testing.T) {
		req, err := http.NewRequest("POST", srv.URL+"/api/auth/login", strings.NewReader("login=a&contrasena=b"))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 415 {
			t.Errorf("estado = %d, se esperaba 415", resp.StatusCode)
		}
	})

	t.Run("usuario desactivado con la sesion abierta", func(t *testing.T) {
		u := crearUsuario(t, auth.RolAdministrador, contrasenaPrueba())
		token := iniciarOK(t, srv, u)
		if r := pedirAPI(t, srv, "GET", "/api/auth/me", nil, token); r.estado != 200 {
			t.Fatalf("me con sesion: estado %d, cuerpo %s", r.estado, r.cuerpo)
		}

		// La baja por la API llega con el mantenimiento de usuarios; aqui se hace en la base,
		// sin revocar las sesiones, para probar que cada peticion revisa usuario.activo.
		if _, err := pool.Exec(ctx, `UPDATE usuario SET activo = false WHERE id = $1`, u.id); err != nil {
			t.Fatal(err)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")

		var revocada bool
		if err := pool.QueryRow(ctx, `SELECT revocada_en IS NOT NULL FROM sesion WHERE token_hash = $1`,
			auth.HashToken(token)).Scan(&revocada); err != nil {
			t.Fatal(err)
		}
		if !revocada {
			t.Error("la sesion del usuario inactivo no se revoco")
		}

		// Tampoco puede volver a entrar, y la respuesta es la misma que la de un inexistente.
		inactivo := iniciar(t, srv, u.usuario, u.contrasena)
		esperarError(t, inactivo, 401, "CREDENCIALES_INVALIDAS")
		inexistente := iniciar(t, srv, "no-existe-"+sufijo(t), u.contrasena)
		if !bytes.Equal(inactivo.cuerpo, inexistente.cuerpo) {
			t.Errorf("inactivo %s distinto de inexistente %s", inactivo.cuerpo, inexistente.cuerpo)
		}

		// Reactivarlo no revive la sesion anterior.
		if _, err := pool.Exec(ctx, `UPDATE usuario SET activo = true WHERE id = $1`, u.id); err != nil {
			t.Fatal(err)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
	})

	t.Run("sesion vencida", func(t *testing.T) {
		u := crearUsuario(t, auth.RolConsulta, contrasenaPrueba())
		token := iniciarOK(t, srv, u)
		if _, err := pool.Exec(ctx,
			`UPDATE sesion SET creada_en = now() - interval '9 hours', expira_en = now() - interval '1 hour'
			  WHERE token_hash = $1`, auth.HashToken(token)); err != nil {
			t.Fatal(err)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
	})
}

// ---------- contrasenas y exposicion del hash ----------

func TestAuth_ContrasenaArgon2idConSalPorUsuario(t *testing.T) {
	misma := contrasenaPrueba()
	a := crearUsuario(t, auth.RolAdministrador, misma)
	b := crearUsuario(t, auth.RolConsulta, misma)

	var ha, hb string
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT hash_contrasena FROM usuario WHERE id = $1`, a.id).Scan(&ha); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT hash_contrasena FROM usuario WHERE id = $1`, b.id).Scan(&hb); err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{ha, hb} {
		if !strings.HasPrefix(h, "$argon2id$") {
			t.Errorf("hash guardado sin argon2id: %q", h)
		}
		if strings.Contains(h, misma) {
			t.Error("el hash contiene la contrasena en claro")
		}
	}
	if ha == hb {
		t.Fatal("dos usuarios con la misma contrasena tienen el mismo hash: falta la sal por usuario")
	}

	// Las dos verifican igual con su propio hash.
	srv := servidor(t)
	iniciarOK(t, srv, a)
	iniciarOK(t, srv, b)
}

func TestAuth_QuienSoyNoExponeHash(t *testing.T) {
	srv := servidor(t)
	u := crearUsuario(t, auth.RolConsulta, contrasenaPrueba())

	login := iniciar(t, srv, u.usuario, u.contrasena)
	if login.estado != 200 {
		t.Fatalf("login: estado %d, cuerpo %s", login.estado, login.cuerpo)
	}
	me := pedirAPI(t, srv, "GET", "/api/auth/me", nil, login.cookie.Value)
	if me.estado != 200 {
		t.Fatalf("me: estado %d, cuerpo %s", me.estado, me.cuerpo)
	}
	if b := leerUsuario(t, me); b.Usuario.ID != u.id || b.Usuario.Rol != auth.RolConsulta || !b.Usuario.Activo {
		t.Errorf("me devolvio otro usuario: %s", me.cuerpo)
	}

	var hash string
	if err := pool.QueryRow(context.Background(), `SELECT hash_contrasena FROM usuario WHERE id = $1`, u.id).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	for nombre, r := range map[string]respuesta{"login": login, "me": me} {
		cuerpo := strings.ToLower(string(r.cuerpo))
		for _, prohibido := range []string{"hash", "sal", "argon2", "contrasena", strings.ToLower(hash)} {
			if strings.Contains(cuerpo, prohibido) {
				t.Errorf("la respuesta de %s contiene %q: %s", nombre, prohibido, r.cuerpo)
			}
		}
	}
}
