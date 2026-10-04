package integracion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/auth"
)

// ---------- ayudas ----------

// catalogoImportado es un servidor cuyo catalogo vive en una transaccion con el Excel
// original ya importado. Al terminar la prueba se revierte: la base queda como estaba.
type catalogoImportado struct {
	srv   *httptest.Server
	admin string
	ctx   context.Context
	tx    pgx.Tx
}

func nuevoCatalogoImportado(t *testing.T) catalogoImportado {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback(ctx) })
	importarEnTx(t, ctx, tx)
	srv := servidorCatalogo(t, tx)
	_, admin := sesionAdmin(t, srv)
	return catalogoImportado{srv: srv, admin: admin, ctx: ctx, tx: tx}
}

func (e catalogoImportado) pedir(t *testing.T, metodo, ruta string, cuerpo any) respuesta {
	t.Helper()
	return pedirAPI(t, e.srv, metodo, ruta, cuerpo, e.admin)
}

// idDe busca un id por codigo dentro de la transaccion.
func (e catalogoImportado) idDe(t *testing.T, tabla, codigo string) int64 {
	t.Helper()
	return id(t, e.ctx, e.tx, fmt.Sprintf(`SELECT id FROM %s WHERE codigo = $1`, tabla), codigo)
}

type opcionJSON struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

type servicioJSON struct {
	ID           int64        `json:"id"`
	Codigo       string       `json:"codigo"`
	Activo       string       `json:"activo"`
	ServicioN1ID int64        `json:"servicio_n1_id"`
	ServicioN1   unidadRef    `json:"servicio_n1"`
	Clase        *opcionJSON  `json:"clase"`
	Criticidad   *opcionJSON  `json:"criticidad"`
	Tipo         *opcionJSON  `json:"tipo"`
	Minimo       *json.Number `json:"minimo"`
	Maximo       *json.Number `json:"maximo"`
	Revision     bool         `json:"requiere_revision"`
	Seccion      *struct {
		ID      int64     `json:"id"`
		Codigo  string    `json:"codigo"`
		Empresa unidadRef `json:"empresa"`
	} `json:"seccion_responsable"`
	Usuario *struct {
		ID      int64  `json:"id"`
		Usuario string `json:"usuario"`
		Activo  bool   `json:"activo"`
	} `json:"usuario_responsable"`
	Origen *struct {
		Hoja string `json:"hoja"`
	} `json:"origen"`
	Incidencias []struct {
		Tipo string `json:"tipo"`
	} `json:"incidencias"`
}

type listadoServicios struct {
	Items  []servicioJSON `json:"items"`
	Total  int            `json:"total"`
	Pagina int            `json:"pagina"`
	Tamano int            `json:"tamano"`
}

func numero(n *json.Number) string {
	if n == nil {
		return "null"
	}
	return n.String()
}

// minMaxEnBase devuelve minimo y maximo como texto, "null" si no hay dato.
func minMaxEnBase(t *testing.T, ctx context.Context, c interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, codigo string) (string, string) {
	t.Helper()
	var mn, mx string
	if err := c.QueryRow(ctx, `SELECT coalesce(minimo::text, 'null'), coalesce(maximo::text, 'null') FROM servicio_n2 WHERE codigo = $1`,
		codigo).Scan(&mn, &mx); err != nil {
		t.Fatalf("leer %s: %v", codigo, err)
	}
	return mn, mx
}

// ---------- P09 ----------

func TestP09_MinimoMayorQueMaximo(t *testing.T) {
	e := nuevoCatalogoImportado(t)
	n1 := e.idDe(t, "servicio_n1", "SE.05")
	suf := sufijo(t)
	crear := func(t *testing.T, codigo string, campos map[string]any) respuesta {
		t.Helper()
		cuerpo := map[string]any{"servicio_n1_id": n1, "codigo": codigo, "nombre": "Servicio de P09"}
		for k, v := range campos {
			cuerpo[k] = v
		}
		return e.pedir(t, "POST", "/api/servicios", cuerpo)
	}
	existe := func(t *testing.T, codigo string) bool {
		t.Helper()
		return id(t, e.ctx, e.tx, `SELECT count(*) FROM servicio_n2 WHERE codigo = $1`, codigo) == 1
	}

	t.Run("alta con minimo 24 y maximo 12 se rechaza y no guarda nada", func(t *testing.T) {
		codigo := "P09." + suf + ".a"
		c := esperarErrorCon(t, crear(t, codigo, map[string]any{"minimo": 24, "maximo": 12}), 422, "MINIMO_MAYOR_QUE_MAXIMO", "24", "12")
		if strings.Join(c.Error.Campos, ",") != "minimo,maximo" {
			t.Errorf("campos = %v, se esperaba [minimo maximo]", c.Error.Campos)
		}
		if existe(t, codigo) {
			t.Error("el servicio rechazado quedo guardado")
		}
	})

	t.Run("alta con minimo 12 y maximo 12 se acepta", func(t *testing.T) {
		codigo := "P09." + suf + ".b"
		r := crear(t, codigo, map[string]any{"minimo": 12, "maximo": 12})
		esperarEstado(t, r, 201)
		sv := decodificar[servicioJSON](t, r)
		if numero(sv.Minimo) != "12" || numero(sv.Maximo) != "12" {
			t.Errorf("respuesta minimo=%s maximo=%s, se esperaba 12 y 12", numero(sv.Minimo), numero(sv.Maximo))
		}
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, codigo); mn != "12" || mx != "12" {
			t.Errorf("en la base minimo=%s maximo=%s", mn, mx)
		}
	})

	t.Run("alta solo con minimo: el maximo queda nulo, no cero", func(t *testing.T) {
		codigo := "P09." + suf + ".c"
		r := crear(t, codigo, map[string]any{"minimo": 12})
		esperarEstado(t, r, 201)
		if !bytes.Contains(r.cuerpo, []byte(`"maximo":null`)) {
			t.Errorf("la respuesta no trae maximo null: %s", r.cuerpo)
		}
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, codigo); mn != "12" || mx != "null" {
			t.Errorf("en la base minimo=%s maximo=%s, se esperaba 12 y null", mn, mx)
		}
	})

	t.Run("alta solo con maximo, o con maximo null explicito: el minimo queda nulo", func(t *testing.T) {
		codigo := "P09." + suf + ".d"
		esperarEstado(t, crear(t, codigo, map[string]any{"minimo": nil, "maximo": 5}), 201)
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, codigo); mn != "null" || mx != "5" {
			t.Errorf("en la base minimo=%s maximo=%s, se esperaba null y 5", mn, mx)
		}
	})

	t.Run("cero es un dato y se distingue de null", func(t *testing.T) {
		codigo := "P09." + suf + ".e"
		esperarEstado(t, crear(t, codigo, map[string]any{"minimo": 0}), 201)
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, codigo); mn != "0" || mx != "null" {
			t.Errorf("en la base minimo=%s maximo=%s, se esperaba 0 y null", mn, mx)
		}
	})

	t.Run("una cadena vacia no se toma como cero", func(t *testing.T) {
		codigo := "P09." + suf + ".f"
		c := esperarErrorCon(t, crear(t, codigo, map[string]any{"minimo": "", "maximo": 3}), 422, "VALIDACION", "minimo")
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "minimo" {
			t.Errorf("campos = %v, se esperaba [minimo]", c.Error.Campos)
		}
		if existe(t, codigo) {
			t.Error("el servicio rechazado quedo guardado")
		}
	})

	t.Run("edicion de un importado: el minimo se compara con el maximo guardado", func(t *testing.T) {
		// SE.05.02 viene del Excel con minimo 12 y maximo 24.
		sv := e.idDe(t, "servicio_n2", "SE.05.02")
		ruta := fmt.Sprintf("/api/servicios/%d", sv)
		esperarErrorCon(t, e.pedir(t, "PUT", ruta, map[string]any{"minimo": 25}), 422, "MINIMO_MAYOR_QUE_MAXIMO", "25", "24", "guardado")
		esperarErrorCon(t, e.pedir(t, "PUT", ruta, map[string]any{"minimo": 24, "maximo": 12}), 422, "MINIMO_MAYOR_QUE_MAXIMO", "24", "12")
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, "SE.05.02"); mn != "12" || mx != "24" {
			t.Fatalf("un rechazo cambio el servicio: minimo=%s maximo=%s", mn, mx)
		}

		esperarEstado(t, e.pedir(t, "PUT", ruta, map[string]any{"minimo": 24}), 200)
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, "SE.05.02"); mn != "24" || mx != "24" {
			t.Errorf("con minimo igual al maximo: minimo=%s maximo=%s", mn, mx)
		}
		r := e.pedir(t, "PUT", ruta, map[string]any{"maximo": nil})
		esperarEstado(t, r, 200)
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, "SE.05.02"); mn != "24" || mx != "null" {
			t.Errorf("al borrar el maximo: minimo=%s maximo=%s, se esperaba 24 y null", mn, mx)
		}
	})

	t.Run("edicion de uno creado: menor rechazado, ausente aceptado", func(t *testing.T) {
		codigo := "P09." + suf + ".b" // el de 12 y 12
		ruta := fmt.Sprintf("/api/servicios/%d", e.idDe(t, "servicio_n2", codigo))
		esperarErrorCon(t, e.pedir(t, "PUT", ruta, map[string]any{"maximo": 11}), 422, "MINIMO_MAYOR_QUE_MAXIMO", "12", "11")
		esperarEstado(t, e.pedir(t, "PUT", ruta, map[string]any{"minimo": nil}), 200)
		if mn, mx := minMaxEnBase(t, e.ctx, e.tx, codigo); mn != "null" || mx != "12" {
			t.Errorf("minimo=%s maximo=%s, se esperaba null y 12", mn, mx)
		}
	})
}

// ---------- P10 ----------

// Conteos del archivo original, verificados contra la base importada.
var (
	hijosPorN1 = map[string]int{
		"SE.01": 3, "SE.02": 6, "SE.03": 3, "SE.04": 3, "SE.05": 2, "SE.06": 10,
		"SE.07": 1, "SE.08": 5, "SE.09": 5, "SE.10": 2, "SE.11": 3, "SE.12": 3,
	}
	porEstado = map[string]int{"S": 42, "N": 1, "DESCONOCIDO": 3}
)

func TestP10_BusquedaYFiltros(t *testing.T) {
	e := nuevoCatalogoImportado(t)
	listar := func(t *testing.T, consulta url.Values) (listadoServicios, respuesta) {
		t.Helper()
		r := e.pedir(t, "GET", "/api/servicios?"+consulta.Encode(), nil)
		esperarEstado(t, r, 200)
		return decodificar[listadoServicios](t, r), r
	}
	con := func(pares ...string) url.Values {
		v := url.Values{"tamano": {"100"}}
		for i := 0; i < len(pares); i += 2 {
			v.Set(pares[i], pares[i+1])
		}
		return v
	}
	opciones := func(t *testing.T, c string) map[string]int64 {
		t.Helper()
		r := e.pedir(t, "GET", "/api/catalogos/"+c, nil)
		esperarEstado(t, r, 200)
		l := decodificar[struct {
			Items []opcionJSON `json:"items"`
		}](t, r)
		m := map[string]int64{}
		for _, o := range l.Items {
			m[o.Nombre] = o.ID
		}
		return m
	}
	clases, criticidades, tipos := opciones(t, "clases"), opciones(t, "criticidades"), opciones(t, "tipos")
	if len(clases) != 2 || len(criticidades) != 5 || len(tipos) != 11 {
		t.Fatalf("catalogos: %d clases, %d criticidades, %d tipos; se esperaban 2, 5 y 11", len(clases), len(criticidades), len(tipos))
	}

	t.Run("filtrar por cada nivel 1 devuelve solo sus hijos y el total coincide", func(t *testing.T) {
		r := e.pedir(t, "GET", "/api/servicios-n1?tamano=100", nil)
		esperarEstado(t, r, 200)
		n1s := decodificar[struct {
			Items []struct {
				ID          int64  `json:"id"`
				Codigo      string `json:"codigo"`
				ServiciosN2 int    `json:"servicios_n2"`
			} `json:"items"`
			Total int `json:"total"`
		}](t, r)
		if n1s.Total != 12 {
			t.Fatalf("servicios de nivel 1 = %d, se esperaban 12", n1s.Total)
		}
		suma := 0
		for _, n := range n1s.Items {
			l, _ := listar(t, con("n1_id", fmt.Sprint(n.ID)))
			if l.Total != hijosPorN1[n.Codigo] || l.Total != n.ServiciosN2 || len(l.Items) != l.Total {
				t.Errorf("%s: total %d, items %d, hijos segun el nivel 1 %d, esperados %d",
					n.Codigo, l.Total, len(l.Items), n.ServiciosN2, hijosPorN1[n.Codigo])
			}
			for _, sv := range l.Items {
				if sv.ServicioN1.Codigo != n.Codigo || sv.ServicioN1ID != n.ID {
					t.Errorf("filtro %s devolvio %s, que es de %s", n.Codigo, sv.Codigo, sv.ServicioN1.Codigo)
				}
			}
			suma += l.Total
		}
		if suma != 46 {
			t.Errorf("la suma de los hijos es %d, se esperaban 46", suma)
		}
	})

	t.Run("buscar un codigo que existe devuelve ese servicio", func(t *testing.T) {
		for _, q := range []string{"SE.06.01", "se.06.01"} {
			l, _ := listar(t, con("q", q))
			if l.Total != 1 || len(l.Items) != 1 || l.Items[0].Codigo != "SE.06.01" {
				t.Errorf("q=%s: total %d, items %+v", q, l.Total, l.Items)
			}
		}
	})

	t.Run("buscar por nombre, sin distinguir mayusculas", func(t *testing.T) {
		l, _ := listar(t, con("q", "TABLERO"))
		var codigos []string
		for _, sv := range l.Items {
			codigos = append(codigos, sv.Codigo)
		}
		if l.Total != 2 || strings.Join(codigos, ",") != "SE.12.1,SE.12.3" {
			t.Errorf("q=TABLERO: total %d, codigos %v; se esperaban SE.12.1 y SE.12.3", l.Total, codigos)
		}
	})

	t.Run("buscar un texto que no existe devuelve cero resultados, no un error", func(t *testing.T) {
		for _, q := range []string{"no-existe-zzz", "%", "_"} {
			l, r := listar(t, con("q", q))
			if l.Total != 0 || !bytes.Contains(r.cuerpo, []byte(`"items":[]`)) {
				t.Errorf("q=%q: total %d, cuerpo %s", q, l.Total, r.cuerpo)
			}
		}
	})

	t.Run("por estado", func(t *testing.T) {
		suma := 0
		for estado, esperado := range porEstado {
			l, _ := listar(t, con("activo", estado))
			if l.Total != esperado {
				t.Errorf("activo=%s: %d, se esperaban %d", estado, l.Total, esperado)
			}
			for _, sv := range l.Items {
				if sv.Activo != estado {
					t.Errorf("activo=%s devolvio %s con %s", estado, sv.Codigo, sv.Activo)
				}
			}
			suma += l.Total
		}
		if suma != 46 {
			t.Errorf("la suma por estado es %d, se esperaban 46", suma)
		}
		l, _ := listar(t, con("activo", "N"))
		if len(l.Items) == 1 && l.Items[0].Codigo != "SE.05.01" {
			t.Errorf("el servicio con activo N es %s, se esperaba SE.05.01", l.Items[0].Codigo)
		}
	})

	t.Run("por clase, criticidad y tipo", func(t *testing.T) {
		for _, c := range []struct {
			param, nombre string
			id            int64
			esperado      int
			de            func(servicioJSON) *opcionJSON
		}{
			{"clase_id", "RECURRENTE", clases["RECURRENTE"], 23, func(s servicioJSON) *opcionJSON { return s.Clase }},
			{"clase_id", "A DEMANDA", clases["A DEMANDA"], 20, func(s servicioJSON) *opcionJSON { return s.Clase }},
			{"criticidad_id", "High", criticidades["High"], 11, func(s servicioJSON) *opcionJSON { return s.Criticidad }},
			{"criticidad_id", "Normal", criticidades["Normal"], 32, func(s servicioJSON) *opcionJSON { return s.Criticidad }},
			{"criticidad_id", "Very High", criticidades["Very High"], 0, func(s servicioJSON) *opcionJSON { return s.Criticidad }},
			{"tipo_id", "Front End", tipos["Front End"], 27, func(s servicioJSON) *opcionJSON { return s.Tipo }},
			{"tipo_id", "Back End", tipos["Back End"], 3, func(s servicioJSON) *opcionJSON { return s.Tipo }},
		} {
			l, _ := listar(t, con(c.param, fmt.Sprint(c.id)))
			if l.Total != c.esperado {
				t.Errorf("%s=%s: %d, se esperaban %d", c.param, c.nombre, l.Total, c.esperado)
			}
			for _, sv := range l.Items {
				if o := c.de(sv); o == nil || o.Nombre != c.nombre {
					t.Errorf("%s=%s devolvio %s con %+v", c.param, c.nombre, sv.Codigo, o)
				}
			}
		}
	})

	t.Run("combinaciones de filtros se aplican con Y", func(t *testing.T) {
		se06 := fmt.Sprint(e.idDe(t, "servicio_n1", "SE.06"))
		se12 := fmt.Sprint(e.idDe(t, "servicio_n1", "SE.12"))
		for _, c := range []struct {
			desc     string
			consulta url.Values
			esperado int
			cumple   func(servicioJSON) bool
		}{
			{"SE.06 y Front End", con("n1_id", se06, "tipo_id", fmt.Sprint(tipos["Front End"])), 9,
				func(s servicioJSON) bool { return s.ServicioN1.Codigo == "SE.06" && s.Tipo.Nombre == "Front End" }},
			{"RECURRENTE y High", con("clase_id", fmt.Sprint(clases["RECURRENTE"]), "criticidad_id", fmt.Sprint(criticidades["High"])), 6,
				func(s servicioJSON) bool { return s.Clase.Nombre == "RECURRENTE" && s.Criticidad.Nombre == "High" }},
			{"activo S y High", con("activo", "S", "criticidad_id", fmt.Sprint(criticidades["High"])), 10,
				func(s servicioJSON) bool { return s.Activo == "S" && s.Criticidad.Nombre == "High" }},
			{"SE.12 y requiere revision", con("n1_id", se12, "requiere_revision", "true"), 3,
				func(s servicioJSON) bool { return s.ServicioN1.Codigo == "SE.12" && s.Revision }},
			{"SE.12 y DESCONOCIDO y texto", con("n1_id", se12, "activo", "DESCONOCIDO", "q", "tablero"), 2,
				func(s servicioJSON) bool { return s.Activo == "DESCONOCIDO" && s.ServicioN1.Codigo == "SE.12" }},
			{"SE.06 y A DEMANDA y Very High", con("n1_id", se06, "clase_id", fmt.Sprint(clases["A DEMANDA"]),
				"criticidad_id", fmt.Sprint(criticidades["Very High"])), 0, func(servicioJSON) bool { return false }},
		} {
			l, _ := listar(t, c.consulta)
			if l.Total != c.esperado || len(l.Items) != c.esperado {
				t.Errorf("%s: total %d, items %d, se esperaban %d", c.desc, l.Total, len(l.Items), c.esperado)
			}
			for _, sv := range l.Items {
				if !c.cumple(sv) {
					t.Errorf("%s devolvio %s, que no cumple el filtro", c.desc, sv.Codigo)
				}
			}
		}
	})

	t.Run("paginacion recorre los 46 sin repetir, en orden de codigo", func(t *testing.T) {
		var codigos []string
		for _, p := range []struct{ pagina, items int }{{1, 20}, {2, 20}, {3, 6}, {4, 0}} {
			l, _ := listar(t, url.Values{"pagina": {fmt.Sprint(p.pagina)}, "tamano": {"20"}})
			if l.Total != 46 || len(l.Items) != p.items || l.Pagina != p.pagina || l.Tamano != 20 {
				t.Errorf("pagina %d: total %d, items %d, pagina %d, tamano %d", p.pagina, l.Total, len(l.Items), l.Pagina, l.Tamano)
			}
			for _, sv := range l.Items {
				codigos = append(codigos, sv.Codigo)
			}
		}
		vistos := map[string]bool{}
		for _, c := range codigos {
			vistos[c] = true
		}
		if len(codigos) != 46 || len(vistos) != 46 || !sort.StringsAreSorted(codigos) {
			t.Errorf("se recorrieron %d codigos, %d distintos, ordenados=%v", len(codigos), len(vistos), sort.StringsAreSorted(codigos))
		}
	})

	t.Run("un filtro con valor fuera de dominio es 422", func(t *testing.T) {
		for _, q := range []string{"activo=X", "activo=s", "n1_id=abc", "clase_id=0", "tamano=101", "pagina=0", "requiere_revision=quiza"} {
			esperarError(t, e.pedir(t, "GET", "/api/servicios?"+q, nil), 422, "VALIDACION")
		}
	})
}

// ---------- P11 ----------

func TestP11_ResponsableDeOtraSeccion(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	a := crearJerarquia(t, srv, admin)
	b := crearJerarquia(t, srv, admin)
	suf := sufijo(t)
	usuarioA := "ra" + suf
	ua := crearAPI(t, srv, admin, "usuarios", map[string]any{
		"nombre": "Responsable de A", "usuario": usuarioA, "contrasena": contrasenaPrueba(), "rol": auth.RolConsulta, "puesto_id": a.puesto,
	})
	n1 := crearAPI(t, srv, admin, "servicios-n1", map[string]any{"codigo": "P11" + suf, "nombre": "Servicio N1 de P11"})
	sv := crearAPI(t, srv, admin, "servicios", map[string]any{"servicio_n1_id": n1, "codigo": "P11" + suf + ".1", "nombre": "Servicio de P11"})
	ruta := fmt.Sprintf("/api/servicios/%d/responsable", sv)
	asignar := func(t *testing.T, seccion, usuario any) respuesta {
		t.Helper()
		return pedirAPI(t, srv, "PUT", ruta, map[string]any{"seccion_id": seccion, "usuario_id": usuario}, admin)
	}
	enBase := func(t *testing.T) string {
		t.Helper()
		var s string
		if err := pool.QueryRow(context.Background(),
			`SELECT coalesce(seccion_responsable_id::text, 'null') || '/' || coalesce(usuario_responsable_id::text, 'null')
			   FROM servicio_n2 WHERE id = $1`, sv).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	t.Run("usuario de otra seccion se rechaza y el mensaje dice por que", func(t *testing.T) {
		c := esperarErrorCon(t, asignar(t, b.seccion, ua), 422, "RESPONSABLE_FUERA_DE_SECCION",
			usuarioA, fmt.Sprintf("id %d", a.seccion), fmt.Sprintf("id %d", b.seccion), "tiene que pertenecer a la seccion responsable")
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "usuario_id" {
			t.Errorf("campos = %v, se esperaba [usuario_id]", c.Error.Campos)
		}
		if got := enBase(t); got != "null/null" {
			t.Errorf("el rechazo cambio la asignacion: %s", got)
		}
	})

	t.Run("usuario de la misma seccion se acepta y la ficha lo muestra", func(t *testing.T) {
		r := asignar(t, a.seccion, ua)
		esperarEstado(t, r, 200)
		f := decodificar[servicioJSON](t, pedirAPI(t, srv, "GET", fmt.Sprintf("/api/servicios/%d", sv), nil, admin))
		if f.Seccion == nil || f.Seccion.ID != a.seccion || f.Seccion.Empresa.Codigo != a.empresaCodigo ||
			f.Usuario == nil || f.Usuario.ID != ua || !f.Usuario.Activo {
			t.Errorf("ficha con responsable incorrecto: seccion %+v, usuario %+v", f.Seccion, f.Usuario)
		}
	})

	t.Run("pasar a la otra seccion con el mismo usuario tambien se rechaza", func(t *testing.T) {
		antes := enBase(t)
		esperarErrorCon(t, asignar(t, b.seccion, ua), 422, "RESPONSABLE_FUERA_DE_SECCION", usuarioA)
		if got := enBase(t); got != antes {
			t.Errorf("el rechazo cambio la asignacion: %s -> %s", antes, got)
		}
	})

	t.Run("solo seccion, sin usuario, se acepta", func(t *testing.T) {
		esperarEstado(t, asignar(t, b.seccion, nil), 200)
		if got := enBase(t); got != fmt.Sprintf("%d/null", b.seccion) {
			t.Errorf("asignacion = %s", got)
		}
	})

	t.Run("usuario sin seccion, seccion inexistente y quitar la asignacion", func(t *testing.T) {
		esperarError(t, asignar(t, nil, ua), 422, "VALIDACION")
		esperarError(t, asignar(t, 999999999, nil), 422, "REFERENCIA_INEXISTENTE")
		esperarError(t, asignar(t, a.seccion, 999999999), 422, "REFERENCIA_INEXISTENTE")
		esperarEstado(t, asignar(t, nil, nil), 200)
		if got := enBase(t); got != "null/null" {
			t.Errorf("asignacion = %s", got)
		}
	})

	t.Run("a un servicio dado de baja no se le asigna responsable", func(t *testing.T) {
		esperarEstado(t, pedirAPI(t, srv, "POST", fmt.Sprintf("/api/servicios/%d/desactivar", sv), map[string]any{}, admin), 200)
		esperarError(t, asignar(t, a.seccion, ua), 409, "SERVICIO_INACTIVO")
	})
}

// ---------- reglas del catalogo ----------

func TestCatalogo_ReferenciasBajasYCodigos(t *testing.T) {
	srv := servidor(t)
	_, admin := sesionAdmin(t, srv)
	suf := sufijo(t)
	pedir := func(t *testing.T, metodo, ruta string, cuerpo any) respuesta {
		t.Helper()
		return pedirAPI(t, srv, metodo, ruta, cuerpo, admin)
	}
	n1 := crearAPI(t, srv, admin, "servicios-n1", map[string]any{"codigo": "C" + suf, "nombre": "Nivel 1 de prueba"})
	clase := crearAPI(t, srv, admin, "catalogos/clases", map[string]any{"nombre": "Clase " + suf, "orden": 50})
	sv := crearAPI(t, srv, admin, "servicios", map[string]any{
		"servicio_n1_id": n1, "codigo": "C" + suf + ".1", "nombre": "Nivel 2 de prueba", "clase_id": clase, "activo": "DESCONOCIDO",
	})

	t.Run("codigo duplicado por nivel y referencias inexistentes", func(t *testing.T) {
		esperarErrorCon(t, pedir(t, "POST", "/api/servicios-n1", map[string]any{"codigo": "C" + suf, "nombre": "Otro"}),
			409, "CODIGO_DUPLICADO", "C"+suf)
		esperarErrorCon(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": "C" + suf + ".1", "nombre": "Otro"}),
			409, "CODIGO_DUPLICADO", "C"+suf+".1")
		c := esperarErrorCon(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": 999999999, "codigo": "X" + suf, "nombre": "Otro"}),
			422, "REFERENCIA_INEXISTENTE", "999999999")
		if len(c.Error.Campos) != 1 || c.Error.Campos[0] != "servicio_n1_id" {
			t.Errorf("campos = %v", c.Error.Campos)
		}
		esperarError(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": "X" + suf, "nombre": "Otro", "tipo_id": 9999}),
			422, "REFERENCIA_INEXISTENTE")
		c = esperarErrorCon(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1}), 422, "VALIDACION")
		if strings.Join(c.Error.Campos, ",") != "codigo,nombre" {
			t.Errorf("campos = %v, se esperaba [codigo nombre]", c.Error.Campos)
		}
		esperarError(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": " X" + suf, "nombre": "Otro"}), 422, "VALIDACION")
		esperarError(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": "X" + suf, "nombre": "Otro", "activo": "SI"}), 422, "VALIDACION")
		esperarError(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": "X" + suf, "nombre": "Otro", "minimo": -1}), 422, "VALIDACION")
		esperarErrorCon(t, pedir(t, "POST", "/api/catalogos/clases", map[string]any{"nombre": "Clase " + suf, "orden": 51}),
			409, "CODIGO_DUPLICADO", "Clase "+suf)
		esperarError(t, pedir(t, "PUT", fmt.Sprintf("/api/catalogos/clases/%d", clase), map[string]any{"nombre": "Y", "orden": 1, "valor_origen": "Y"}),
			400, "JSON_INVALIDO")
		esperarError(t, pedir(t, "GET", "/api/catalogos/colores", nil), 404, "NO_ENCONTRADO")
	})

	t.Run("baja del nivel 1 y de la opcion con un hijo DESCONOCIDO se rechaza", func(t *testing.T) {
		c := esperarErrorCon(t, pedir(t, "POST", fmt.Sprintf("/api/servicios-n1/%d/desactivar", n1), map[string]any{}),
			409, "DEPENDENCIAS_ACTIVAS", "C"+suf)
		if c.Error.TotalDependencias != 1 || len(c.Error.Dependencias) != 1 || c.Error.Dependencias[0].ID != sv {
			t.Errorf("dependencias = %+v", c.Error.Dependencias)
		}
		esperarError(t, pedir(t, "POST", fmt.Sprintf("/api/catalogos/clases/%d/desactivar", clase), map[string]any{}), 409, "DEPENDENCIAS_ACTIVAS")
		if !activoEnBase(t, "servicio_n1", n1) || !activoEnBase(t, "clase_servicio", clase) {
			t.Error("un rechazo desactivo algo")
		}
	})

	t.Run("de abajo hacia arriba si, y nada se reactiva bajo un padre inactivo", func(t *testing.T) {
		r := pedir(t, "POST", fmt.Sprintf("/api/servicios/%d/desactivar", sv), map[string]any{})
		esperarEstado(t, r, 200)
		if s := decodificar[servicioJSON](t, r); s.Activo != "N" {
			t.Errorf("activo = %s tras desactivar", s.Activo)
		}
		esperarEstado(t, pedir(t, "POST", fmt.Sprintf("/api/servicios-n1/%d/desactivar", n1), map[string]any{}), 200)
		esperarError(t, pedir(t, "POST", fmt.Sprintf("/api/servicios/%d/activar", sv), map[string]any{}), 409, "PADRE_INACTIVO")
		esperarError(t, pedir(t, "PUT", fmt.Sprintf("/api/servicios/%d", sv), map[string]any{"activo": "S"}), 409, "PADRE_INACTIVO")
		esperarError(t, pedir(t, "POST", "/api/servicios", map[string]any{"servicio_n1_id": n1, "codigo": "Z" + suf, "nombre": "Bajo inactivo"}),
			409, "PADRE_INACTIVO")
		esperarEstado(t, pedir(t, "POST", fmt.Sprintf("/api/servicios-n1/%d/activar", n1), map[string]any{}), 200)
		r = pedir(t, "POST", fmt.Sprintf("/api/servicios/%d/activar", sv), map[string]any{})
		esperarEstado(t, r, 200)
		if s := decodificar[servicioJSON](t, r); s.Activo != "S" {
			t.Errorf("activo = %s tras activar", s.Activo)
		}
	})

	t.Run("una opcion inactiva no se asigna, pero se conserva donde ya estaba", func(t *testing.T) {
		otra := crearAPI(t, srv, admin, "catalogos/clases", map[string]any{"nombre": "Clase B " + suf, "orden": 52})
		esperarEstado(t, pedir(t, "POST", fmt.Sprintf("/api/catalogos/clases/%d/desactivar", otra), map[string]any{}), 200)
		esperarError(t, pedir(t, "PUT", fmt.Sprintf("/api/servicios/%d", sv), map[string]any{"clase_id": otra}), 409, "PADRE_INACTIVO")

		// La clase original se desactiva con el servicio dado de baja; editar otro campo la conserva.
		esperarEstado(t, pedir(t, "POST", fmt.Sprintf("/api/servicios/%d/desactivar", sv), map[string]any{}), 200)
		esperarEstado(t, pedir(t, "POST", fmt.Sprintf("/api/catalogos/clases/%d/desactivar", clase), map[string]any{}), 200)
		r := pedir(t, "PUT", fmt.Sprintf("/api/servicios/%d", sv), map[string]any{"nombre": "Renombrado", "clase_id": clase})
		esperarEstado(t, r, 200)
		if s := decodificar[servicioJSON](t, r); s.Clase == nil || s.Clase.ID != clase || s.Clase.Activo {
			t.Errorf("clase = %+v, se esperaba la original, inactiva", s.Clase)
		}
		r = pedir(t, "GET", "/api/catalogos/clases?activo=true", nil)
		esperarEstado(t, r, 200)
		if bytes.Contains(r.cuerpo, []byte("Clase "+suf)) {
			t.Error("una opcion inactiva aparece en la lista de opciones activas")
		}
	})
}

func TestCatalogo_FichaConOrigenEIncidencias(t *testing.T) {
	e := nuevoCatalogoImportado(t)

	r := e.pedir(t, "GET", fmt.Sprintf("/api/servicios/%d", e.idDe(t, "servicio_n2", "SE.12.3")), nil)
	esperarEstado(t, r, 200)
	f := decodificar[servicioJSON](t, r)
	if f.ServicioN1.Codigo != "SE.12" || f.Activo != "DESCONOCIDO" || f.Minimo != nil || f.Maximo != nil || !f.Revision {
		t.Errorf("ficha de SE.12.3: n1 %s, activo %s, minimo %s, maximo %s, revision %v",
			f.ServicioN1.Codigo, f.Activo, numero(f.Minimo), numero(f.Maximo), f.Revision)
	}
	if f.Origen == nil || f.Origen.Hoja != "Servicios Externos" {
		t.Errorf("origen = %+v", f.Origen)
	}
	tipos := map[string]bool{}
	for _, i := range f.Incidencias {
		tipos[i.Tipo] = true
	}
	if !tipos["N1_HEREDADO_DE_FILA_ANTERIOR"] || !tipos["ATRIBUTOS_AUSENTES"] {
		t.Errorf("incidencias de SE.12.3 = %v", f.Incidencias)
	}
	for _, campo := range []string{`"descripcion":null`, `"clase":null`, `"seccion_responsable":null`, `"usuario_responsable":null`} {
		if !bytes.Contains(r.cuerpo, []byte(campo)) {
			t.Errorf("la ficha no trae %s: %s", campo, r.cuerpo)
		}
	}

	r = e.pedir(t, "GET", fmt.Sprintf("/api/servicios-n1/%d", e.idDe(t, "servicio_n1", "SE.12")), nil)
	esperarEstado(t, r, 200)
	n1 := decodificar[struct {
		Servicios []struct {
			Codigo string `json:"codigo"`
		} `json:"servicios"`
		Incidencias []struct {
			Tipo string `json:"tipo"`
		} `json:"incidencias"`
	}](t, r)
	if len(n1.Servicios) != 3 {
		t.Errorf("SE.12 tiene %d servicios en su ficha, se esperaban 3", len(n1.Servicios))
	}
	conflicto := false
	for _, i := range n1.Incidencias {
		conflicto = conflicto || i.Tipo == "N1_NOMBRE_CONFLICTO"
	}
	if !conflicto {
		t.Errorf("la ficha de SE.12 no muestra el conflicto de nombre: %v", n1.Incidencias)
	}
}
