package integracion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"catalogo/internal/auth"
	apihttp "catalogo/internal/http"
)

// ---------- ayudas ----------

var tablaDe = map[string]string{
	"empresas": "empresa", "areas": "area", "departamentos": "departamento",
	"secciones": "seccion", "puestos": "puesto", "usuarios": "usuario",
	"servicios-n1": "servicio_n1", "servicios": "servicio_n2",
	"catalogos": "clase_servicio", "catalogos/clases": "clase_servicio",
}

// recursosP03 son los recursos cuyo registro P03 intenta modificar.
var recursosP03 = []string{"empresas", "areas", "departamentos", "secciones", "puestos", "usuarios",
	"servicios-n1", "servicios", "catalogos"}

// limpiarAlTerminar borra el registro al final de la prueba. t.Cleanup corre en orden
// inverso, asi que los hijos, creados despues, se borran antes que sus padres.
func limpiarAlTerminar(t *testing.T, tabla string, id int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		if tabla == "usuario" {
			if _, err := pool.Exec(ctx, `DELETE FROM sesion WHERE usuario_id = $1`, id); err != nil {
				t.Errorf("limpieza de sesiones: %v", err)
			}
		}
		if _, err := pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, tabla), id); err != nil {
			t.Errorf("limpieza de %s %d: %v", tabla, id, err)
		}
	})
}

// crearAPI crea un registro por la API, exige 201 y registra su limpieza.
func crearAPI(t *testing.T, srv *httptest.Server, token, recurso string, cuerpo map[string]any) int64 {
	t.Helper()
	r := pedirAPI(t, srv, "POST", "/api/"+recurso, cuerpo, token)
	if r.estado != 201 {
		t.Fatalf("crear %s %v: estado %d, cuerpo %s", recurso, cuerpo, r.estado, r.cuerpo)
	}
	var c struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(r.cuerpo, &c); err != nil || c.ID == 0 {
		t.Fatalf("crear %s: cuerpo sin id: %s", recurso, r.cuerpo)
	}
	limpiarAlTerminar(t, tablaDe[recurso], c.ID)
	return c.ID
}

func sesionAdmin(t *testing.T, srv *httptest.Server) (usuarioPrueba, string) {
	t.Helper()
	u := crearUsuario(t, auth.RolAdministrador, contrasenaPrueba())
	return u, iniciarOK(t, srv, u)
}

type jerarquiaPrueba struct {
	empresaCodigo                         string
	empresa, area, depto, seccion, puesto int64
}

// crearJerarquia crea por la API una cadena completa. Los codigos por debajo de la
// empresa se repiten entre pruebas a proposito: solo son unicos dentro de su padre.
func crearJerarquia(t *testing.T, srv *httptest.Server, token string) jerarquiaPrueba {
	t.Helper()
	j := jerarquiaPrueba{empresaCodigo: "J" + sufijo(t)}
	j.empresa = crearAPI(t, srv, token, "empresas", map[string]any{"codigo": j.empresaCodigo, "nombre": "Empresa de prueba"})
	j.area = crearAPI(t, srv, token, "areas", map[string]any{"codigo": "AR", "nombre": "Area de prueba", "empresa_id": j.empresa})
	j.depto = crearAPI(t, srv, token, "departamentos", map[string]any{"codigo": "DP", "nombre": "Departamento de prueba", "area_id": j.area})
	j.seccion = crearAPI(t, srv, token, "secciones", map[string]any{"codigo": "SC", "nombre": "Seccion de prueba", "departamento_id": j.depto})
	j.puesto = crearAPI(t, srv, token, "puestos", map[string]any{"codigo": "PU", "nombre": "Puesto de prueba", "seccion_id": j.seccion})
	return j
}

// crearServicio inserta un servicio N1 y uno N2 a cargo de la seccion (y del usuario, si
// no es 0). Va directo a la base para fijar estados, como DESCONOCIDO, sin pasar por la API.
func crearServicio(t *testing.T, seccionID, usuarioID int64, activo string) (int64, string) {
	t.Helper()
	ctx := context.Background()
	codigo := "T" + sufijo(t)
	var n1, n2 int64
	if err := pool.QueryRow(ctx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ($1, 'Servicio N1 de prueba') RETURNING id`,
		codigo).Scan(&n1); err != nil {
		t.Fatal(err)
	}
	limpiarAlTerminar(t, "servicio_n1", n1)
	var usuario any
	if usuarioID != 0 {
		usuario = usuarioID
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre, activo, seccion_responsable_id, usuario_responsable_id)
		 VALUES ($1, $2, 'Servicio N2 de prueba', $3, $4, $5) RETURNING id`,
		n1, codigo+".1", activo, seccionID, usuario).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	limpiarAlTerminar(t, "servicio_n2", n2)
	return n2, codigo + ".1"
}

type cuerpoError struct {
	Error struct {
		Codigo       string   `json:"codigo"`
		Mensaje      string   `json:"mensaje"`
		Campos       []string `json:"campos"`
		Dependencias []struct {
			Entidad string `json:"entidad"`
			ID      int64  `json:"id"`
			Codigo  string `json:"codigo"`
		} `json:"dependencias"`
		TotalDependencias int            `json:"total_dependencias"`
		Detalle           map[string]any `json:"detalle"`
	} `json:"error"`
}

// esperarErrorCon comprueba estado y codigo, y que el mensaje contenga cada dato.
func esperarErrorCon(t *testing.T, r respuesta, estado int, codigo string, datos ...string) cuerpoError {
	t.Helper()
	esperarError(t, r, estado, codigo)
	var c cuerpoError
	if err := json.Unmarshal(r.cuerpo, &c); err != nil {
		t.Fatal(err)
	}
	for _, d := range datos {
		if !strings.Contains(c.Error.Mensaje, d) {
			t.Errorf("el mensaje %q no nombra %q", c.Error.Mensaje, d)
		}
	}
	return c
}

func esperarEstado(t *testing.T, r respuesta, estado int) {
	t.Helper()
	if r.estado != estado {
		t.Fatalf("estado = %d, se esperaba %d; cuerpo %s", r.estado, estado, r.cuerpo)
	}
}

func activoEnBase(t *testing.T, tabla string, id int64) bool {
	t.Helper()
	var activo bool
	if err := pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT activo FROM %s WHERE id = $1`, tabla), id).Scan(&activo); err != nil {
		t.Fatal(err)
	}
	return activo
}

// ---------- P03 ----------

var parametroRuta = regexp.MustCompile(`\{[^}]+\}`)

// concretar reemplaza los parametros de un patron. {id} toma el id real del recurso y {c}
// el catalogo de clases, al que pertenece el id de "catalogos". Cualquier otro parametro
// vale "1": el rol se revisa antes de mirarlo.
func concretar(patron string, ids map[string]int64) string {
	recurso := strings.Split(strings.TrimPrefix(patron, "/api/"), "/")[0]
	return parametroRuta.ReplaceAllStringFunc(patron, func(p string) string {
		if id, ok := ids[recurso]; ok && p == "{id}" {
			return fmt.Sprint(id)
		}
		if p == "{c}" {
			return "clases"
		}
		return "1"
	})
}

// fotoDatos resume en texto el estado de todo lo que la prueba P03 intenta modificar.
func fotoDatos(t *testing.T, ids map[string]int64) string {
	t.Helper()
	var partes []string
	for _, recurso := range recursosP03 {
		var fila string
		tabla := tablaDe[recurso]
		if err := pool.QueryRow(context.Background(),
			fmt.Sprintf(`SELECT row_to_json(t)::text FROM %s t WHERE id = $1`, tabla), ids[recurso]).Scan(&fila); err != nil {
			t.Fatal(err)
		}
		partes = append(partes, fila)
	}
	for _, tabla := range []string{"empresa", "area", "departamento", "seccion", "puesto", "usuario",
		"servicio_n1", "servicio_n2", "clase_servicio", "criticidad", "tipo_servicio"} {
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+tabla).Scan(&n); err != nil {
			t.Fatal(err)
		}
		partes = append(partes, fmt.Sprintf("%s=%d", tabla, n))
	}
	return strings.Join(partes, "\n")
}

func TestP03_ConsultaNoPuedeEscribir(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)
	clave, suf := contrasenaPrueba(), sufijo(t)
	consultaID := crearAPI(t, srv, admin, "usuarios", map[string]any{
		"nombre": "Consulta P03", "usuario": "c" + suf, "contrasena": clave, "rol": auth.RolConsulta, "puesto_id": j.puesto,
	})
	consulta := iniciarOK(t, srv, usuarioPrueba{usuario: "c" + suf, contrasena: clave})
	n1 := crearAPI(t, srv, admin, "servicios-n1", map[string]any{"codigo": "P03" + suf, "nombre": "Servicio N1 de P03"})
	clase := crearAPI(t, srv, admin, "catalogos/clases", map[string]any{"nombre": "Clase P03 " + suf, "orden": 90})
	n2 := crearAPI(t, srv, admin, "servicios", map[string]any{
		"servicio_n1_id": n1, "codigo": "P03" + suf + ".1", "nombre": "Servicio N2 de P03", "clase_id": clase,
		"minimo": 1, "maximo": 2,
	})
	ids := map[string]int64{
		"empresas": j.empresa, "areas": j.area, "departamentos": j.depto,
		"secciones": j.seccion, "puestos": j.puesto, "usuarios": consultaID,
		"servicios-n1": n1, "servicios": n2, "catalogos": clase,
	}

	// La lista sale de las rutas que registra el enrutador. Toda ruta que no sea GET es de
	// escritura, salvo iniciar y cerrar sesion, que por diseno son para cualquiera.
	sinRol := map[string]bool{"POST /api/auth/login": true, "POST /api/auth/logout": true}
	var escrituras, lecturas []apihttp.Ruta
	registradas := map[string]bool{}
	for _, r := range apihttp.RutasAPI() {
		clave := r.Metodo + " " + r.Patron
		registradas[clave] = true
		switch {
		case sinRol[clave]:
		case r.Metodo == "GET":
			lecturas = append(lecturas, r)
		default:
			escrituras = append(escrituras, r)
		}
	}

	// Control de que la lista es la real: estan todas las de escritura de estructura y
	// usuarios que define arquitectura.md.
	var esperadas []string
	for _, rec := range []string{"empresas", "areas", "departamentos", "secciones", "puestos"} {
		esperadas = append(esperadas, "POST /api/"+rec, "PUT /api/"+rec+"/{id}",
			"POST /api/"+rec+"/{id}/desactivar", "POST /api/"+rec+"/{id}/activar")
	}
	esperadas = append(esperadas, "POST /api/usuarios", "PUT /api/usuarios/{id}", "PUT /api/usuarios/{id}/contrasena",
		"POST /api/usuarios/{id}/desactivar", "POST /api/usuarios/{id}/activar")
	deEstructura := len(esperadas)
	// Las del catalogo: servicios de los dos niveles, responsable y catalogos de opciones.
	esperadas = append(esperadas,
		"POST /api/servicios-n1", "PUT /api/servicios-n1/{id}", "POST /api/servicios-n1/{id}/desactivar", "POST /api/servicios-n1/{id}/activar",
		"POST /api/servicios", "PUT /api/servicios/{id}", "PUT /api/servicios/{id}/responsable",
		"POST /api/servicios/{id}/desactivar", "POST /api/servicios/{id}/activar",
		"POST /api/catalogos/{c}", "PUT /api/catalogos/{c}/{id}", "POST /api/catalogos/{c}/{id}/desactivar", "POST /api/catalogos/{c}/{id}/activar")
	for _, e := range esperadas {
		if !registradas[e] {
			t.Errorf("la ruta de escritura %s no esta registrada", e)
		}
	}
	if len(escrituras) != len(esperadas) {
		t.Errorf("hay %d rutas de escritura registradas y %d en la lista de control: agregue las nuevas", len(escrituras), len(esperadas))
	}
	t.Logf("rutas de escritura recorridas: %d (de estructura y usuarios: %d; de catalogo: %d)",
		len(escrituras), deEstructura, len(esperadas)-deEstructura)

	antes := fotoDatos(t, ids)

	for _, r := range escrituras {
		t.Run("escritura "+r.Metodo+" "+r.Patron, func(t *testing.T) {
			ruta := concretar(r.Patron, ids)
			esperarError(t, pedirAPI(t, srv, r.Metodo, ruta, map[string]any{}, consulta), 403, "PROHIBIDO")
			// Primero la sesion y despues el rol: sin sesion es 401, no 403.
			esperarError(t, pedirAPI(t, srv, r.Metodo, ruta, map[string]any{}, ""), 401, "NO_AUTENTICADO")
		})
	}

	if despues := fotoDatos(t, ids); despues != antes {
		t.Errorf("los intentos de escritura de consulta cambiaron datos.\nantes:\n%s\ndespues:\n%s", antes, despues)
	}

	n := 0
	for _, r := range lecturas {
		recurso := strings.Split(strings.TrimPrefix(r.Patron, "/api/"), "/")[0]
		if _, ok := ids[recurso]; !ok && r.Patron != "/api/auth/me" {
			continue
		}
		n++
		t.Run("lectura "+r.Patron, func(t *testing.T) {
			esperarEstado(t, pedirAPI(t, srv, "GET", concretar(r.Patron, ids), nil, consulta), 200)
		})
	}
	t.Logf("rutas de lectura recorridas con consulta: %d", n)

	t.Run("crear empresa da 403 y listarlas da 200", func(t *testing.T) {
		esperarError(t, pedirAPI(t, srv, "POST", "/api/empresas",
			map[string]any{"codigo": "X" + sufijo(t), "nombre": "No debe existir"}, consulta), 403, "PROHIBIDO")
		esperarEstado(t, pedirAPI(t, srv, "GET", "/api/empresas", nil, consulta), 200)
	})
}

// ---------- P04 ----------

type unidadJSON struct {
	ID        int64  `json:"id"`
	Codigo    string `json:"codigo"`
	Activo    bool   `json:"activo"`
	EmpresaID int64  `json:"empresa_id"`
	SeccionID int64  `json:"seccion_id"`
	Empresa   *struct {
		ID     int64  `json:"id"`
		Codigo string `json:"codigo"`
	} `json:"empresa"`
	Ancestros []struct {
		Entidad string `json:"entidad"`
		ID      int64  `json:"id"`
		Codigo  string `json:"codigo"`
	} `json:"ancestros"`
}

type unidadRef struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
}

type usuarioJSON struct {
	ID        int64  `json:"id"`
	Usuario   string `json:"usuario"`
	Rol       string `json:"rol"`
	Activo    bool   `json:"activo"`
	PuestoID  int64  `json:"puesto_id"`
	Jerarquia struct {
		Puesto       unidadRef `json:"puesto"`
		Seccion      unidadRef `json:"seccion"`
		Departamento unidadRef `json:"departamento"`
		Area         unidadRef `json:"area"`
		Empresa      unidadRef `json:"empresa"`
	} `json:"jerarquia"`
	ServiciosResponsable []struct {
		Codigo string `json:"codigo"`
	} `json:"servicios_responsable"`
}

func decodificar[T any](t *testing.T, r respuesta) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.cuerpo, &v); err != nil {
		t.Fatalf("cuerpo inesperado: %v (%s)", err, r.cuerpo)
	}
	return v
}

func TestP04_JerarquiaCompletaYUsuario(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)

	clave, suf := contrasenaPrueba(), sufijo(t)
	r := pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
		"nombre": "Usuario P04", "usuario": "p" + suf, "correo": "p" + suf + "@prueba.test",
		"contrasena": clave, "rol": auth.RolConsulta, "puesto_id": j.puesto,
	}, admin)
	esperarEstado(t, r, 201)
	creado := decodificar[usuarioJSON](t, r)
	limpiarAlTerminar(t, "usuario", creado.ID)

	t.Run("el usuario se recupera con su jerarquia", func(t *testing.T) {
		r := pedirAPI(t, srv, "GET", fmt.Sprintf("/api/usuarios/%d", creado.ID), nil, admin)
		esperarEstado(t, r, 200)
		u := decodificar[usuarioJSON](t, r)
		jj := u.Jerarquia
		if u.PuestoID != j.puesto || jj.Puesto.ID != j.puesto || jj.Seccion.ID != j.seccion ||
			jj.Departamento.ID != j.depto || jj.Area.ID != j.area || jj.Empresa.ID != j.empresa {
			t.Errorf("jerarquia distinta de la creada: %s", r.cuerpo)
		}
		if jj.Empresa.Codigo != j.empresaCodigo || jj.Puesto.Codigo != "PU" || jj.Seccion.Codigo != "SC" {
			t.Errorf("codigos de la jerarquia: %s", r.cuerpo)
		}
		if u.Usuario != "p"+suf || u.Rol != auth.RolConsulta || !u.Activo || u.ServiciosResponsable == nil {
			t.Errorf("ficha inesperada: %s", r.cuerpo)
		}
	})

	t.Run("el puesto se recupera con la ruta de ancestros", func(t *testing.T) {
		r := pedirAPI(t, srv, "GET", fmt.Sprintf("/api/puestos/%d", j.puesto), nil, admin)
		esperarEstado(t, r, 200)
		p := decodificar[unidadJSON](t, r)
		var ruta []string
		for _, a := range p.Ancestros {
			ruta = append(ruta, a.Entidad+":"+a.Codigo)
		}
		if got, want := strings.Join(ruta, " > "), "empresa:"+j.empresaCodigo+" > area:AR > departamento:DP > seccion:SC"; got != want {
			t.Errorf("ancestros = %s, se esperaba %s", got, want)
		}
		if p.SeccionID != j.seccion {
			t.Errorf("seccion_id = %d, se esperaba %d", p.SeccionID, j.seccion)
		}
	})

	t.Run("los listados filtran por padre y traen al padre", func(t *testing.T) {
		r := pedirAPI(t, srv, "GET", fmt.Sprintf("/api/areas?empresa_id=%d", j.empresa), nil, admin)
		esperarEstado(t, r, 200)
		l := decodificar[struct {
			Items  []unidadJSON `json:"items"`
			Total  int          `json:"total"`
			Pagina int          `json:"pagina"`
			Tamano int          `json:"tamano"`
		}](t, r)
		if l.Total != 1 || len(l.Items) != 1 || l.Pagina != 1 || l.Tamano != 20 {
			t.Fatalf("listado inesperado: %s", r.cuerpo)
		}
		if a := l.Items[0]; a.ID != j.area || a.EmpresaID != j.empresa || a.Empresa == nil || a.Empresa.Codigo != j.empresaCodigo {
			t.Errorf("item sin su padre: %s", r.cuerpo)
		}

		r = pedirAPI(t, srv, "GET", fmt.Sprintf("/api/usuarios?seccion_id=%d", j.seccion), nil, admin)
		esperarEstado(t, r, 200)
		if !strings.Contains(string(r.cuerpo), `"usuario":"p`+suf+`"`) {
			t.Errorf("el usuario no aparece al filtrar por su seccion: %s", r.cuerpo)
		}
	})

	t.Run("el usuario creado inicia sesion y ve su jerarquia", func(t *testing.T) {
		token := iniciarOK(t, srv, usuarioPrueba{usuario: "p" + suf, contrasena: clave})
		r := pedirAPI(t, srv, "GET", "/api/auth/me", nil, token)
		esperarEstado(t, r, 200)
		if b := leerUsuario(t, r); b.Usuario.Jerarquia.Empresa.Codigo != j.empresaCodigo {
			t.Errorf("me: %s", r.cuerpo)
		}
	})
}

// ---------- P05 ----------

func TestP05_CodigoDuplicadoYReferenciaInexistente(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)

	t.Run("dos areas con el mismo codigo en la misma empresa", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/areas", map[string]any{"codigo": "AR", "nombre": "Otra", "empresa_id": j.empresa}, admin)
		c := esperarErrorCon(t, r, 409, "CODIGO_DUPLICADO", "AR", j.empresaCodigo)
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "codigo" {
			t.Errorf("campos = %v, se esperaba [codigo]", c.Error.Campos)
		}
	})

	t.Run("el mismo codigo en otra empresa si se acepta", func(t *testing.T) {
		otra := crearAPI(t, srv, admin, "empresas", map[string]any{"codigo": "J" + sufijo(t), "nombre": "Otra empresa"})
		crearAPI(t, srv, admin, "areas", map[string]any{"codigo": "AR", "nombre": "Area", "empresa_id": otra})
	})

	t.Run("codigo de empresa duplicado", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/empresas", map[string]any{"codigo": j.empresaCodigo, "nombre": "Repetida"}, admin)
		esperarErrorCon(t, r, 409, "CODIGO_DUPLICADO", j.empresaCodigo)
	})

	t.Run("modificar a un codigo que ya existe en el padre", func(t *testing.T) {
		otro := crearAPI(t, srv, admin, "puestos", map[string]any{"codigo": "PU2", "nombre": "Otro", "seccion_id": j.seccion})
		r := pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/puestos/%d", otro),
			map[string]any{"codigo": "PU", "nombre": "Otro", "seccion_id": j.seccion}, admin)
		esperarErrorCon(t, r, 409, "CODIGO_DUPLICADO", "PU", "SC")
	})

	t.Run("departamento con un area que no existe", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/departamentos", map[string]any{"codigo": "D1", "nombre": "Huerfano", "area_id": 987654321}, admin)
		c := esperarErrorCon(t, r, 422, "REFERENCIA_INEXISTENTE", "area_id", "987654321")
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "area_id" {
			t.Errorf("campos = %v, se esperaba [area_id]", c.Error.Campos)
		}
	})

	t.Run("departamento sin area", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/departamentos", map[string]any{"codigo": "D1", "nombre": "Huerfano"}, admin)
		esperarErrorCon(t, r, 422, "VALIDACION", "area_id")
	})

	t.Run("usuario con un puesto que no existe", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
			"nombre": "Sin puesto", "usuario": "s" + sufijo(t), "contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": 987654321,
		}, admin)
		esperarErrorCon(t, r, 422, "REFERENCIA_INEXISTENTE", "puesto_id", "987654321")
	})

	t.Run("usuario y correo repetidos sin distinguir mayusculas", func(t *testing.T) {
		suf := sufijo(t)
		crearAPI(t, srv, admin, "usuarios", map[string]any{
			"nombre": "Original", "usuario": "dup" + suf, "correo": "dup" + suf + "@prueba.test",
			"contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": j.puesto,
		})
		r := pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
			"nombre": "Copia", "usuario": "DUP" + suf, "contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": j.puesto,
		}, admin)
		esperarErrorCon(t, r, 409, "CODIGO_DUPLICADO", "DUP"+suf)
		r = pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
			"nombre": "Copia", "usuario": "otro" + suf, "correo": "DUP" + suf + "@prueba.test",
			"contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": j.puesto,
		}, admin)
		c := esperarErrorCon(t, r, 409, "CODIGO_DUPLICADO", "DUP"+suf+"@prueba.test")
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "correo" {
			t.Errorf("campos = %v, se esperaba [correo]", c.Error.Campos)
		}
	})

	t.Run("codigo con espacios y campos mal formados", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/empresas", map[string]any{"codigo": " E1 ", "nombre": "Con espacios"}, admin)
		esperarErrorCon(t, r, 422, "VALIDACION", "espacios")
		r = pedirAPI(t, srv, "POST", "/api/areas", map[string]any{"codigo": "A9", "nombre": "x", "empresa_id": j.empresa, "area_id": 1}, admin)
		esperarError(t, r, 400, "JSON_INVALIDO")
		r = pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
			"nombre": "Rol raro", "usuario": "r" + sufijo(t), "contrasena": contrasenaPrueba(), "rol": "root", "puesto_id": j.puesto,
		}, admin)
		esperarErrorCon(t, r, 422, "VALIDACION", "root")
	})

	t.Run("id inexistente en la ruta", func(t *testing.T) {
		esperarError(t, pedirAPI(t, srv, "GET", "/api/secciones/987654321", nil, admin), 404, "NO_ENCONTRADO")
		esperarError(t, pedirAPI(t, srv, "POST", "/api/usuarios/987654321/desactivar", map[string]any{}, admin), 404, "NO_ENCONTRADO")
	})
}

// ---------- bajas, padres inactivos y reactivacion ----------

func TestOrganizacion_BajaConDependientesActivos(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)
	area2 := crearAPI(t, srv, admin, "areas", map[string]any{"codigo": "AR2", "nombre": "Segunda area", "empresa_id": j.empresa})

	t.Run("empresa con areas activas", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/empresas/%d/desactivar", j.empresa), map[string]any{}, admin)
		c := esperarErrorCon(t, r, 409, "DEPENDENCIAS_ACTIVAS", j.empresaCodigo, "2 areas activas")
		var codigos []string
		for _, d := range c.Error.Dependencias {
			if d.Entidad != "area" {
				t.Errorf("dependencia de otra entidad: %+v", d)
			}
			codigos = append(codigos, d.Codigo)
		}
		if strings.Join(codigos, ",") != "AR,AR2" || c.Error.TotalDependencias != 2 {
			t.Errorf("dependencias = %v, total %d; se esperaba AR,AR2 y 2", codigos, c.Error.TotalDependencias)
		}
		if !activoEnBase(t, "empresa", j.empresa) || !activoEnBase(t, "area", j.area) {
			t.Error("un rechazo no debe cambiar nada")
		}
	})

	t.Run("puesto con un usuario activo", func(t *testing.T) {
		suf := sufijo(t)
		crearAPI(t, srv, admin, "usuarios", map[string]any{
			"nombre": "Ocupante", "usuario": "o" + suf, "contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": j.puesto,
		})
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/puestos/%d/desactivar", j.puesto), nil, admin)
		c := esperarErrorCon(t, r, 409, "DEPENDENCIAS_ACTIVAS", "1 usuario activo")
		if len(c.Error.Dependencias) != 1 || c.Error.Dependencias[0].Codigo != "o"+suf {
			t.Errorf("dependencias = %+v", c.Error.Dependencias)
		}
	})

	t.Run("seccion con un servicio DESCONOCIDO a su cargo", func(t *testing.T) {
		seccion := crearAPI(t, srv, admin, "secciones", map[string]any{"codigo": "SC2", "nombre": "Sin puestos", "departamento_id": j.depto})
		sv, codigo := crearServicio(t, seccion, 0, "DESCONOCIDO")
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/secciones/%d/desactivar", seccion), map[string]any{}, admin)
		c := esperarErrorCon(t, r, 409, "DEPENDENCIAS_ACTIVAS", "1 servicio activo a su cargo")
		if len(c.Error.Dependencias) != 1 || c.Error.Dependencias[0].Entidad != "servicio_n2" || c.Error.Dependencias[0].Codigo != codigo {
			t.Errorf("dependencias = %+v", c.Error.Dependencias)
		}
		// Con el servicio dado de baja ya no bloquea, y la seccion queda como dato historico.
		if _, err := pool.Exec(context.Background(), `UPDATE servicio_n2 SET activo = 'N' WHERE id = $1`, sv); err != nil {
			t.Fatal(err)
		}
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/secciones/%d/desactivar", seccion), map[string]any{}, admin), 200)
	})

	t.Run("de abajo hacia arriba si se puede, y repetir no cambia nada", func(t *testing.T) {
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/areas/%d/desactivar", area2), map[string]any{}, admin), 200)
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/empresas/%d/desactivar", j.empresa), map[string]any{}, admin)
		c := esperarErrorCon(t, r, 409, "DEPENDENCIAS_ACTIVAS", "1 area activa")
		if c.Error.TotalDependencias != 1 || c.Error.Dependencias[0].Codigo != "AR" {
			t.Errorf("solo debia quedar AR: %s", r.cuerpo)
		}
		r = pedirAPI(t, srv, "POST", fmt.Sprintf("/api/areas/%d/desactivar", area2), map[string]any{}, admin)
		esperarEstado(t, r, 200)
		if u := decodificar[unidadJSON](t, r); u.Activo {
			t.Errorf("segunda baja: %s", r.cuerpo)
		}
	})
}

func TestOrganizacion_PadreInactivo(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)
	area2 := crearAPI(t, srv, admin, "areas", map[string]any{"codigo": "AR2", "nombre": "Area a desactivar", "empresa_id": j.empresa})
	depto2 := crearAPI(t, srv, admin, "departamentos", map[string]any{"codigo": "DP2", "nombre": "Departamento", "area_id": area2})

	esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/departamentos/%d/desactivar", depto2), map[string]any{}, admin), 200)
	esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/areas/%d/desactivar", area2), map[string]any{}, admin), 200)

	t.Run("crear bajo un padre inactivo", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", "/api/departamentos", map[string]any{"codigo": "DP3", "nombre": "Nuevo", "area_id": area2}, admin)
		c := esperarErrorCon(t, r, 409, "PADRE_INACTIVO", "AR2")
		if c.Error.Detalle["entidad"] != "area" || c.Error.Detalle["codigo"] != "AR2" {
			t.Errorf("detalle = %v", c.Error.Detalle)
		}
	})

	t.Run("mover a un padre inactivo", func(t *testing.T) {
		r := pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/departamentos/%d", j.depto),
			map[string]any{"codigo": "DP", "nombre": "Departamento de prueba", "area_id": area2}, admin)
		esperarErrorCon(t, r, 409, "PADRE_INACTIVO", "AR2")
	})

	t.Run("reactivar con el padre inactivo", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/departamentos/%d/activar", depto2), map[string]any{}, admin)
		esperarErrorCon(t, r, 409, "PADRE_INACTIVO", "AR2")
		if activoEnBase(t, "departamento", depto2) {
			t.Error("el departamento se reactivo con el area inactiva")
		}
	})

	t.Run("reactivar de arriba hacia abajo, sin reactivar hijos", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/areas/%d/activar", area2), map[string]any{}, admin)
		esperarEstado(t, r, 200)
		if activoEnBase(t, "departamento", depto2) {
			t.Error("reactivar el area reactivo al departamento")
		}
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/departamentos/%d/activar", depto2), map[string]any{}, admin), 200)
	})

	t.Run("usuario en un puesto inactivo", func(t *testing.T) {
		puesto := crearAPI(t, srv, admin, "puestos", map[string]any{"codigo": "PI", "nombre": "Inactivo", "seccion_id": j.seccion})
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/puestos/%d/desactivar", puesto), map[string]any{}, admin), 200)
		r := pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
			"nombre": "Nadie", "usuario": "n" + sufijo(t), "contrasena": contrasenaPrueba(), "rol": "consulta", "puesto_id": puesto,
		}, admin)
		esperarErrorCon(t, r, 409, "PADRE_INACTIVO", "PI")
	})
}

// ---------- usuarios ----------

func TestUsuarios_UltimoAdministrador(t *testing.T) {
	srv := servidor(t)
	a, admin := sesionAdmin(t, srv)
	ctx := context.Background()

	// Se deja a "a" como unico administrador activo de catalogo_test y se restaura al final.
	var otros []int64
	filas, err := pool.Query(ctx,
		`UPDATE usuario SET activo = false WHERE rol = 'administrador' AND activo AND id <> $1 RETURNING id`, a.id)
	if err != nil {
		t.Fatal(err)
	}
	for filas.Next() {
		var id int64
		if err := filas.Scan(&id); err != nil {
			t.Fatal(err)
		}
		otros = append(otros, id)
	}
	filas.Close()
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `UPDATE usuario SET activo = true WHERE id = ANY($1)`, otros); err != nil {
			t.Errorf("restaurar administradores: %v", err)
		}
	})

	var puesto int64
	if err := pool.QueryRow(ctx, `SELECT puesto_id FROM usuario WHERE id = $1`, a.id).Scan(&puesto); err != nil {
		t.Fatal(err)
	}
	degradar := map[string]any{"nombre": "Usuario de prueba", "usuario": a.usuario, "correo": a.correo, "rol": auth.RolConsulta, "puesto_id": puesto}

	t.Run("desactivar al ultimo", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/usuarios/%d/desactivar", a.id), map[string]any{}, admin)
		esperarErrorCon(t, r, 409, "ULTIMO_ADMINISTRADOR", a.usuario)
	})
	t.Run("pasar a consulta al ultimo", func(t *testing.T) {
		r := pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/usuarios/%d", a.id), degradar, admin)
		esperarErrorCon(t, r, 409, "ULTIMO_ADMINISTRADOR", a.usuario)
	})
	var rol string
	var activo bool
	if err := pool.QueryRow(ctx, `SELECT rol, activo FROM usuario WHERE id = $1`, a.id).Scan(&rol, &activo); err != nil {
		t.Fatal(err)
	}
	if rol != auth.RolAdministrador || !activo {
		t.Fatalf("el ultimo administrador cambio: rol %s, activo %v", rol, activo)
	}

	t.Run("con otro administrador activo si se puede", func(t *testing.T) {
		suf := sufijo(t)
		b := crearAPI(t, srv, admin, "usuarios", map[string]any{
			"nombre": "Segundo admin", "usuario": "b" + suf, "contrasena": contrasenaPrueba(), "rol": auth.RolAdministrador, "puesto_id": puesto,
		})
		esperarEstado(t, pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/usuarios/%d", b),
			map[string]any{"nombre": "Segundo admin", "usuario": "b" + suf, "rol": auth.RolConsulta, "puesto_id": puesto}, admin), 200)
		// Ya no es administrador: "a" vuelve a ser el ultimo.
		esperarErrorCon(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/usuarios/%d/desactivar", a.id), map[string]any{}, admin),
			409, "ULTIMO_ADMINISTRADOR")
	})
}

func TestUsuarios_DesactivarRevocaSesionesYAdvierte(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)
	// Los puestos se crean antes que el usuario: el usuario termina en puesto3, y la
	// limpieza, que corre en orden inverso, tiene que borrarlo antes que al puesto.
	seccion2 := crearAPI(t, srv, admin, "secciones", map[string]any{"codigo": "SC2", "nombre": "Otra", "departamento_id": j.depto})
	puesto2 := crearAPI(t, srv, admin, "puestos", map[string]any{"codigo": "PU", "nombre": "Otro", "seccion_id": seccion2})
	puesto3 := crearAPI(t, srv, admin, "puestos", map[string]any{"codigo": "PU3", "nombre": "Mismo lugar", "seccion_id": j.seccion})
	clave, suf := contrasenaPrueba(), sufijo(t)
	u := crearAPI(t, srv, admin, "usuarios", map[string]any{
		"nombre": "Responsable", "usuario": "r" + suf, "contrasena": clave, "rol": "consulta", "puesto_id": j.puesto,
	})
	_, codigo := crearServicio(t, j.seccion, u, "S")
	token := iniciarOK(t, srv, usuarioPrueba{usuario: "r" + suf, contrasena: clave})

	t.Run("cambiar de seccion a un responsable", func(t *testing.T) {
		r := pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/usuarios/%d", u),
			map[string]any{"nombre": "Responsable", "usuario": "r" + suf, "rol": "consulta", "puesto_id": puesto2}, admin)
		c := esperarErrorCon(t, r, 409, "ASIGNACIONES_AFECTADAS", "r"+suf)
		if len(c.Error.Dependencias) != 1 || c.Error.Dependencias[0].Codigo != codigo {
			t.Errorf("dependencias = %+v", c.Error.Dependencias)
		}
		// Mover el puesto entero a otra seccion tambien la romperia.
		r = pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/puestos/%d", j.puesto),
			map[string]any{"codigo": "PU9", "nombre": "Puesto movido", "seccion_id": seccion2}, admin)
		esperarError(t, r, 409, "ASIGNACIONES_AFECTADAS")
		// A otro puesto de la misma seccion, si.
		esperarEstado(t, pedirAPI(t, srv, "PUT", fmt.Sprintf("/api/usuarios/%d", u),
			map[string]any{"nombre": "Responsable", "usuario": "r" + suf, "rol": "consulta", "puesto_id": puesto3}, admin), 200)
	})

	t.Run("la baja se hace, revoca la sesion y advierte", func(t *testing.T) {
		r := pedirAPI(t, srv, "POST", fmt.Sprintf("/api/usuarios/%d/desactivar", u), map[string]any{}, admin)
		esperarEstado(t, r, 200)
		b := decodificar[struct {
			Usuario      usuarioJSON `json:"usuario"`
			Advertencias []struct {
				Mensaje   string `json:"mensaje"`
				Servicios []struct {
					Codigo string `json:"codigo"`
				} `json:"servicios"`
			} `json:"advertencias"`
		}](t, r)
		if b.Usuario.Activo || len(b.Advertencias) != 1 || len(b.Advertencias[0].Servicios) != 1 ||
			b.Advertencias[0].Servicios[0].Codigo != codigo {
			t.Errorf("respuesta de la baja: %s", r.cuerpo)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
		var vigentes int
		if err := pool.QueryRow(context.Background(),
			`SELECT count(*) FROM sesion WHERE usuario_id = $1 AND revocada_en IS NULL`, u).Scan(&vigentes); err != nil {
			t.Fatal(err)
		}
		if vigentes != 0 {
			t.Errorf("quedaron %d sesiones sin revocar", vigentes)
		}
		var responsable int64
		if err := pool.QueryRow(context.Background(),
			`SELECT usuario_responsable_id FROM servicio_n2 WHERE codigo = $1`, codigo).Scan(&responsable); err != nil {
			t.Fatal(err)
		}
		if responsable != u {
			t.Error("la baja borro la asignacion de responsable")
		}
	})

	t.Run("reactivar no revive la sesion anterior", func(t *testing.T) {
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/usuarios/%d/activar", u), map[string]any{}, admin), 200)
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
		iniciarOK(t, srv, usuarioPrueba{usuario: "r" + suf, contrasena: clave})
	})
}

func TestUsuarios_NingunaRespuestaTraeElHash(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	j := crearJerarquia(t, srv, admin)
	clave, suf := contrasenaPrueba(), sufijo(t)
	alta := pedirAPI(t, srv, "POST", "/api/usuarios", map[string]any{
		"nombre": "Sin secretos", "usuario": "h" + suf, "correo": "h" + suf + "@prueba.test",
		"contrasena": clave, "rol": "consulta", "puesto_id": j.puesto,
	}, admin)
	esperarEstado(t, alta, 201)
	id := decodificar[usuarioJSON](t, alta).ID
	limpiarAlTerminar(t, "usuario", id)
	ruta := fmt.Sprintf("/api/usuarios/%d", id)

	edicion := map[string]any{"nombre": "Sin secretos", "usuario": "h" + suf, "rol": "consulta", "puesto_id": j.puesto}
	respuestas := map[string]respuesta{
		"alta":           alta,
		"listado":        pedirAPI(t, srv, "GET", "/api/usuarios?q=h"+suf, nil, admin),
		"listado total":  pedirAPI(t, srv, "GET", "/api/usuarios?tamano=100", nil, admin),
		"ficha":          pedirAPI(t, srv, "GET", ruta, nil, admin),
		"modificacion":   pedirAPI(t, srv, "PUT", ruta, edicion, admin),
		"baja":           pedirAPI(t, srv, "POST", ruta+"/desactivar", map[string]any{}, admin),
		"reactivacion":   pedirAPI(t, srv, "POST", ruta+"/activar", map[string]any{}, admin),
		"ficha consulta": pedirAPI(t, srv, "GET", ruta, nil, iniciarOK(t, srv, usuarioPrueba{usuario: "h" + suf, contrasena: clave})),
	}
	if !strings.Contains(string(respuestas["listado"].cuerpo), `"usuario":"h`+suf+`"`) {
		t.Fatalf("el listado no trae al usuario: %s", respuestas["listado"].cuerpo)
	}

	var cifrada string
	if err := pool.QueryRow(context.Background(), `SELECT hash_contrasena FROM usuario WHERE id = $1`, id).Scan(&cifrada); err != nil {
		t.Fatal(err)
	}
	for nombre, r := range respuestas {
		if r.estado != 200 && r.estado != 201 {
			t.Errorf("%s: estado %d, cuerpo %s", nombre, r.estado, r.cuerpo)
		}
		cuerpo := strings.ToLower(string(r.cuerpo))
		for _, prohibido := range []string{"hash", `"sal"`, "argon2", "contrasena", strings.ToLower(cifrada)} {
			if strings.Contains(cuerpo, prohibido) {
				t.Errorf("la respuesta de %s contiene %q: %s", nombre, prohibido, r.cuerpo)
			}
		}
	}

	t.Run("cambiar la contrasena revoca las sesiones", func(t *testing.T) {
		token := iniciarOK(t, srv, usuarioPrueba{usuario: "h" + suf, contrasena: clave})
		nueva := contrasenaPrueba()
		r := pedirAPI(t, srv, "PUT", ruta+"/contrasena", map[string]any{"contrasena": nueva}, admin)
		esperarEstado(t, r, 204)
		if len(r.cuerpo) != 0 {
			t.Errorf("204 con cuerpo: %s", r.cuerpo)
		}
		esperarError(t, pedirAPI(t, srv, "GET", "/api/auth/me", nil, token), 401, "NO_AUTENTICADO")
		esperarError(t, iniciar(t, srv, "h"+suf, clave), 401, "CREDENCIALES_INVALIDAS")
		iniciarOK(t, srv, usuarioPrueba{usuario: "h" + suf, contrasena: nueva})
		esperarErrorCon(t, pedirAPI(t, srv, "PUT", ruta+"/contrasena", map[string]any{"contrasena": "corta"}, admin),
			422, "VALIDACION", "8")
	})
}
