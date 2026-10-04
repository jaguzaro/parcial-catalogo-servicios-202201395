package importador

import (
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Hoja es la unica hoja que se importa.
const Hoja = "Servicios Externos"

const (
	filaEncabezados = 4
	primeraFilaDato = 5
	rotuloOpciones  = "OPCIONES"
)

// encabezados de A4:L4, texto exacto. Si alguno no coincide la corrida falla.
var encabezados = [12]string{
	"COD.N1", "SERVICIO - Nivel 1", "COD.N2", "SERVICIO - Nivel 2", "ACTIVO",
	"CLASE DE SERVICIO", "CRITICIDAD", "TIPO DE SERVICIO", "Descripción", "Métrica",
	"Minimo", "Maximo",
}

var codigoEstandar = regexp.MustCompile(`^SE\.[0-9]{2}\.[0-9]{2}$`)

// Opcion es una entrada de un catalogo leida del bloque de opciones.
type Opcion struct {
	Valor string
	Celda string
	Orden int
}

// ServicioN1 es un codigo de nivel 1 con las filas de servicio que resolvieron a el.
type ServicioN1 struct {
	Codigo string
	Nombre string
	Filas  []int
	Celdas map[string]string
}

// ServicioN2 es un servicio de nivel 2 ya resuelto: una celda C principal y sus filas.
// Los punteros nulos son ausencias, no ceros (D11).
type ServicioN2 struct {
	Codigo           string
	N1               string
	Nombre           string
	Activo           string
	Clase            *string
	Criticidad       *string
	Tipo             *string
	Descripcion      *string
	Metrica          *string
	Minimo           *string
	Maximo           *string
	RequiereRevision bool
	Filas            []int
	Celdas           map[string]string
}

// Incidencia es una observacion de la corrida. N1 y N2 son los codigos del registro
// afectado; los ids se resuelven al escribir en la base.
type Incidencia struct {
	Tipo     string
	Regla    string
	Fila     int
	Celdas   string
	Codigo   string
	N1       string
	N2       string
	Original any
	Aplicado any
	Mensaje  string
}

// Plan es todo lo que sale del archivo, antes de tocar la base.
type Plan struct {
	Clases       []Opcion
	Criticidades []Opcion
	Tipos        []Opcion
	N1           []*ServicioN1
	N2           []*ServicioN2
	Incidencias  []Incidencia
	// FilasNoImportables son las filas FILA_SIN_CODIGO: cuentan como omitidas.
	FilasNoImportables int
	// RepetidosN2 son celdas C principales con un codigo ya visto (CODIGO_N2_REPETIDO).
	RepetidosN2 int
	Controles   map[string]int
}

// hoja da acceso a las celdas propias y a las resueltas a traves de las combinaciones.
type hoja struct {
	filas     [][]string
	principal map[string]string // celda -> celda principal del rango que la cubre
	rango     map[string][2]int // celda principal -> filas primera y ultima del rango
}

func ref(col byte, fila int) string { return fmt.Sprintf("%c%d", col, fila) }

func vacio(v string) bool { return strings.TrimSpace(v) == "" }

func (h *hoja) crudo(col byte, fila int) string {
	c := int(col - 'A')
	if fila-1 < 0 || fila-1 >= len(h.filas) || c >= len(h.filas[fila-1]) {
		return ""
	}
	return h.filas[fila-1][c]
}

// propio es el valor de la propia celda. Una celda combinada que no es la principal se
// lee siempre vacia, sin depender de lo que devuelva la libreria.
func (h *hoja) propio(col byte, fila int) string {
	r := ref(col, fila)
	if p, ok := h.principal[r]; ok && p != r {
		return ""
	}
	return h.crudo(col, fila)
}

// celda devuelve la referencia y el valor efectivos: la celda principal si la celda esta
// en un rango combinado, si no la propia.
func (h *hoja) celda(col byte, fila int) (string, string) {
	r := ref(col, fila)
	if p, ok := h.principal[r]; ok {
		c, f, err := excelize.CellNameToCoordinates(p)
		if err == nil {
			return p, h.crudo(byte('A'+c-1), f)
		}
	}
	return r, h.crudo(col, fila)
}

func (h *hoja) combinada(col byte, fila int) bool {
	_, ok := h.principal[ref(col, fila)]
	return ok
}

func (h *hoja) filaVacia(fila int) bool {
	for col := byte('A'); col <= 'L'; col++ {
		if !vacio(h.propio(col, fila)) || h.combinada(col, fila) {
			return false
		}
	}
	return true
}

func leerHoja(r io.Reader) (*hoja, error) {
	f, err := excelize.OpenReader(r, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("abrir el libro: %w", err)
	}
	defer f.Close()

	if idx, err := f.GetSheetIndex(Hoja); err != nil || idx < 0 {
		return nil, fmt.Errorf("el libro no tiene la hoja %q", Hoja)
	}
	filas, err := f.GetRows(Hoja, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, fmt.Errorf("leer la hoja: %w", err)
	}
	combinadas, err := f.GetMergeCells(Hoja)
	if err != nil {
		return nil, fmt.Errorf("leer las celdas combinadas: %w", err)
	}
	h := &hoja{filas: filas, principal: map[string]string{}, rango: map[string][2]int{}}
	for _, mc := range combinadas {
		inicio := strings.ToUpper(mc.GetStartAxis())
		c1, f1, err1 := excelize.CellNameToCoordinates(inicio)
		c2, f2, err2 := excelize.CellNameToCoordinates(mc.GetEndAxis())
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("rango combinado ilegible: %s:%s", mc.GetStartAxis(), mc.GetEndAxis())
		}
		h.rango[inicio] = [2]int{f1, f2}
		for c := c1; c <= c2; c++ {
			for fl := f1; fl <= f2; fl++ {
				nombre, _ := excelize.CoordinatesToCellName(c, fl)
				h.principal[nombre] = inicio
			}
		}
	}
	return h, nil
}

// Analizar lee el libro y aplica las reglas de lectura (docs/diseno/mapeo-excel.md). No
// toca la base. Todo valor del archivo es dato.
func Analizar(r io.Reader) (*Plan, error) {
	h, err := leerHoja(r)
	if err != nil {
		return nil, err
	}
	for i, esperado := range encabezados {
		col := byte('A' + i)
		if v := h.propio(col, filaEncabezados); v != esperado {
			return nil, fmt.Errorf("encabezado %s: se esperaba %q y dice %q", ref(col, filaEncabezados), esperado, v)
		}
	}

	ultima := primeraFilaDato - 1
	for !h.filaVacia(ultima + 1) {
		ultima++
	}
	if ultima < primeraFilaDato {
		return nil, errors.New("la hoja no tiene filas de datos a partir de la fila 5")
	}

	p := &Plan{Controles: map[string]int{}}
	if err := leerOpciones(h, ultima, p); err != nil {
		return nil, err
	}
	if err := leerServicios(h, ultima, p); err != nil {
		return nil, err
	}
	return p, nil
}

// leerOpciones carga los catalogos del bloque de opciones (D12): la fila con OPCIONES en
// E:H y, debajo, cada columna hasta su primera celda vacia.
func leerOpciones(h *hoja, ultima int, p *Plan) error {
	rotulo := 0
	for f := ultima + 1; f <= len(h.filas); f++ {
		todas := true
		for col := byte('E'); col <= 'H'; col++ {
			if h.propio(col, f) != rotuloOpciones {
				todas = false
				break
			}
		}
		if todas {
			rotulo = f
			break
		}
	}
	if rotulo == 0 {
		return fmt.Errorf("no se encontro la fila %s en E:H debajo de los datos", rotuloOpciones)
	}
	lista := func(col byte) []Opcion {
		var ops []Opcion
		for f := rotulo + 1; !vacio(h.propio(col, f)); f++ {
			ops = append(ops, Opcion{Valor: h.propio(col, f), Celda: ref(col, f), Orden: len(ops) + 1})
		}
		return ops
	}
	activo := lista('E')
	if len(activo) != 2 || activo[0].Valor != "S" || activo[1].Valor != "N" {
		return fmt.Errorf("la lista de ACTIVO (columna E bajo %s) no es exactamente S y N", ref('E', rotulo))
	}
	p.Clases, p.Criticidades, p.Tipos = lista('F'), lista('G'), lista('H')
	if len(p.Clases) == 0 || len(p.Criticidades) == 0 || len(p.Tipos) == 0 {
		return errors.New("alguna lista de opciones de F, G o H esta vacia")
	}
	return nil
}

type celdaValor struct{ celda, valor string }

// columna junta los valores no vacios de una columna en las filas de un servicio. Vale el
// de la fila principal o, si esta vacio, el primero no vacio (D21). Devuelve tambien las
// celdas distintas leidas, para la trazabilidad y el conflicto.
func (h *hoja) columna(col byte, filas []int) (string, []celdaValor) {
	var leidas []celdaValor
	vistas := map[string]bool{}
	for _, f := range filas {
		c, v := h.celda(col, f)
		if vacio(v) || vistas[c] {
			continue
		}
		vistas[c] = true
		leidas = append(leidas, celdaValor{c, v})
	}
	if _, v := h.celda(col, filas[0]); !vacio(v) {
		return v, leidas
	}
	if len(leidas) > 0 {
		return leidas[0].valor, leidas
	}
	return "", leidas
}

func distintos(leidas []celdaValor) int {
	m := map[string]bool{}
	for _, l := range leidas {
		m[l.valor] = true
	}
	return len(m)
}

func comoMapa(leidas []celdaValor) map[string]string {
	m := make(map[string]string, len(leidas))
	for _, l := range leidas {
		m[l.celda] = l.valor
	}
	return m
}

func listaCeldas(leidas []celdaValor) string {
	s := make([]string, len(leidas))
	for i, l := range leidas {
		s[i] = l.celda
	}
	return strings.Join(s, ",")
}

func texto(v string) *string {
	if vacio(v) {
		return nil
	}
	return &v
}

// resolverN1 aplica D08 a la fila principal de un servicio: su celda A, la principal del
// rango que la cubre o, si no, el ultimo valor no vacio de A hacia arriba. desde > 0 indica
// que se uso el tercer paso y de que fila salio el valor.
func (h *hoja) resolverN1(fila int) (codigo, celda string, desde int) {
	if c, v := h.celda('A', fila); !vacio(v) {
		return v, c, 0
	}
	for f := fila - 1; f >= primeraFilaDato; f-- {
		if c, v := h.celda('A', f); !vacio(v) {
			_, origen, _ := excelize.CellNameToCoordinates(c)
			return v, c, origen
		}
	}
	return "", "", 0
}

func leerServicios(h *hoja, ultima int, p *Plan) error {
	catalogo := func(ops []Opcion) map[string]bool {
		m := map[string]bool{}
		for _, o := range ops {
			m[o.Valor] = true
		}
		return m
	}
	clases, criticidades, tipos := catalogo(p.Clases), catalogo(p.Criticidades), catalogo(p.Tipos)

	n1PorCodigo := map[string]*ServicioN1{}
	n2Vistos := map[string]string{}

	for f := primeraFilaDato; f <= ultima; f++ {
		if !vacio(h.propio('A', f)) {
			p.Controles["celdas_n1"]++
		}
		if !vacio(h.propio('C', f)) {
			p.Controles["celdas_n2"]++
		}
	}

	for f := primeraFilaDato; f <= ultima; f++ {
		rc := ref('C', f)
		principal, enRango := h.principal[rc]
		if enRango && principal != rc {
			continue // fila de continuacion: la cuenta el servicio de la celda principal
		}
		filas := []int{f}
		if enRango {
			filas = nil
			for x := h.rango[rc][0]; x <= h.rango[rc][1]; x++ {
				filas = append(filas, x)
			}
		}
		codigo := h.propio('C', f)

		if vacio(codigo) {
			// D10: no es un servicio ni se asigna a otro.
			for _, x := range filas {
				valores := map[string]string{}
				for col := byte('A'); col <= 'L'; col++ {
					if v := h.propio(col, x); !vacio(v) {
						valores[ref(col, x)] = v
					}
				}
				p.FilasNoImportables++
				p.Incidencias = append(p.Incidencias, Incidencia{
					Tipo: "FILA_SIN_CODIGO", Regla: "D10", Fila: x,
					Celdas:   fmt.Sprintf("A%d:L%d", x, x),
					Original: valores,
					Mensaje: fmt.Sprintf("La fila %d no tiene codigo de nivel 2 y esta fuera de todo rango combinado de C. "+
						"No se importa ni se asigna a otro servicio.", x),
				})
			}
			continue
		}

		if primera, ok := n2Vistos[codigo]; ok {
			p.RepetidosN2++
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "CODIGO_N2_REPETIDO", Regla: "D13", Fila: f, Celdas: rc, Codigo: codigo, N2: codigo,
				Original: map[string]string{primera: codigo, rc: codigo},
				Mensaje:  fmt.Sprintf("El codigo %s ya aparecio en %s. La celda %s se omite.", codigo, primera, rc),
			})
			continue
		}
		n2Vistos[codigo] = rc

		codN1, celdaN1, desde := h.resolverN1(f)
		if vacio(codN1) {
			return fmt.Errorf("el servicio %s (%s) no tiene codigo de nivel 1 en A ni encima de el", codigo, rc)
		}
		n1 := n1PorCodigo[codN1]
		if n1 == nil {
			n1 = &ServicioN1{Codigo: codN1, Celdas: map[string]string{}}
			n1PorCodigo[codN1] = n1
			p.N1 = append(p.N1, n1)
		}
		n1.Celdas[celdaN1] = codN1
		n1.Filas = append(n1.Filas, filas...)

		s := &ServicioN2{Codigo: codigo, N1: codN1, Filas: filas,
			Celdas: map[string]string{rc: codigo, celdaN1: codN1}}
		p.N2 = append(p.N2, s)

		if desde > 0 {
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "N1_HEREDADO_DE_FILA_ANTERIOR", Regla: "D08", Fila: f, Celdas: celdaN1, Codigo: codigo, N1: codN1, N2: codigo,
				Original: map[string]any{"fila_origen": desde, "celda": celdaN1, "valor": codN1},
				Aplicado: map[string]string{"servicio_n1": codN1},
				Mensaje: fmt.Sprintf("La fila %d no tiene codigo en A ni rango combinado que la cubra. "+
					"Se toma %s de %s, el ultimo valor no vacio de A hacia arriba.", f, codN1, celdaN1),
			})
		}
		if !codigoEstandar.MatchString(codigo) {
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "CODIGO_FORMATO_NO_ESTANDAR", Regla: "D20", Fila: f, Celdas: rc, Codigo: codigo, N1: codN1, N2: codigo,
				Original: codigo, Aplicado: codigo,
				Mensaje: fmt.Sprintf("El codigo %s no sigue el formato SE.NN.NN. Se guarda tal cual, sin normalizar.", codigo),
			})
		}
		if !strings.HasPrefix(codigo, codN1+".") {
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "PREFIJO_INCOHERENTE", Regla: "D08", Fila: f, Celdas: rc, Codigo: codigo, N1: codN1, N2: codigo,
				Original: map[string]string{"codigo": codigo, "servicio_n1": codN1},
				Mensaje:  fmt.Sprintf("El codigo %s no empieza por %s. Se importa bajo %s.", codigo, codN1+".", codN1),
			})
		}

		// Columnas D a L: valor de la fila principal, con conflicto si las filas difieren (D21).
		valores := map[byte]string{}
		campos := map[byte]string{'D': "nombre", 'E': "activo", 'F': "clase", 'G': "criticidad", 'H': "tipo",
			'I': "descripcion", 'J': "metrica", 'K': "minimo", 'L': "maximo"}
		for col := byte('D'); col <= 'L'; col++ {
			v, leidas := h.columna(col, filas)
			valores[col] = v
			for c, x := range comoMapa(leidas) {
				s.Celdas[c] = x
			}
			if distintos(leidas) > 1 {
				p.Incidencias = append(p.Incidencias, Incidencia{
					Tipo: "CONFLICTO_ATRIBUTO", Regla: "D21", Fila: f, Celdas: listaCeldas(leidas), Codigo: codigo, N1: codN1, N2: codigo,
					Original: comoMapa(leidas), Aplicado: map[string]string{campos[col]: v},
					Mensaje: fmt.Sprintf("Las filas de %s traen valores distintos en la columna %c. Vale el de la fila principal.",
						codigo, col),
				})
			}
		}

		s.Nombre = valores['D']
		if vacio(s.Nombre) {
			return fmt.Errorf("el servicio %s (%s) no tiene nombre en D", codigo, rc)
		}
		revision := false
		var vaciasEL []string
		for col := byte('E'); col <= 'L'; col++ {
			if vacio(valores[col]) {
				vaciasEL = append(vaciasEL, string(col))
			}
		}

		switch e := valores['E']; {
		case vacio(e):
			s.Activo = "DESCONOCIDO"
		case e == "S" || e == "N":
			s.Activo = e
		default:
			s.Activo = "DESCONOCIDO"
			revision = true
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "VALOR_FUERA_DE_DOMINIO", Regla: "D11", Fila: f, Celdas: ref('E', f), Codigo: codigo, N1: codN1, N2: codigo,
				Original: e, Aplicado: "DESCONOCIDO",
				Mensaje: fmt.Sprintf("ACTIVO de %s vale %q, que no es S ni N. Queda DESCONOCIDO.", codigo, e),
			})
		}

		enCatalogo := func(col byte, cat map[string]bool, nombre string) *string {
			v := valores[col]
			if vacio(v) {
				return nil
			}
			if !cat[v] {
				revision = true
				p.Incidencias = append(p.Incidencias, Incidencia{
					Tipo: "VALOR_FUERA_DE_CATALOGO", Regla: "D12", Fila: f, Celdas: ref(col, f), Codigo: codigo, N1: codN1, N2: codigo,
					Original: v,
					Mensaje:  fmt.Sprintf("%s de %s vale %q, que no esta en la lista de opciones. Queda nulo.", nombre, codigo, v),
				})
				return nil
			}
			return &v
		}
		s.Clase = enCatalogo('F', clases, "CLASE DE SERVICIO")
		s.Criticidad = enCatalogo('G', criticidades, "CRITICIDAD")
		s.Tipo = enCatalogo('H', tipos, "TIPO DE SERVICIO")
		s.Descripcion = texto(valores['I'])
		s.Metrica = texto(valores['J'])

		numero := func(col byte) *string {
			v := valores[col]
			if vacio(v) {
				return nil // D11: que falte no quiere decir cero
			}
			n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
				revision = true
				p.Incidencias = append(p.Incidencias, Incidencia{
					Tipo: "VALOR_NUMERICO_INVALIDO", Regla: "D11", Fila: f, Celdas: ref(col, f), Codigo: codigo, N1: codN1, N2: codigo,
					Original: v,
					Mensaje:  fmt.Sprintf("La columna %c de %s vale %q, que no es un numero. Queda nulo.", col, codigo, v),
				})
				return nil
			}
			t := strconv.FormatFloat(n, 'f', -1, 64)
			return &t
		}
		s.Minimo, s.Maximo = numero('K'), numero('L')
		if s.Minimo != nil && s.Maximo != nil {
			mn, _ := strconv.ParseFloat(*s.Minimo, 64)
			mx, _ := strconv.ParseFloat(*s.Maximo, 64)
			if mn > mx {
				revision = true
				p.Incidencias = append(p.Incidencias, Incidencia{
					Tipo: "MINIMO_MAYOR_QUE_MAXIMO", Regla: "D11", Fila: f, Celdas: fmt.Sprintf("K%d:L%d", f, filas[len(filas)-1]),
					Codigo: codigo, N1: codN1, N2: codigo,
					Original: map[string]string{"minimo": *s.Minimo, "maximo": *s.Maximo},
					Aplicado: map[string]any{"minimo": nil, "maximo": nil},
					Mensaje:  fmt.Sprintf("En %s el minimo es mayor que el maximo. Los dos quedan nulos.", codigo),
				})
				s.Minimo, s.Maximo = nil, nil
			}
		}

		// D11: faltar E, F, G o H marca revision; lo que falta queda nulo.
		faltaEH := false
		for _, c := range vaciasEL {
			if c <= "H" {
				faltaEH = true
			}
		}
		if faltaEH {
			revision = true
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "ATRIBUTOS_AUSENTES", Regla: "D11", Fila: f,
				Celdas: fmt.Sprintf("E%d:L%d", f, filas[len(filas)-1]), Codigo: codigo, N1: codN1, N2: codigo,
				Original: map[string]any{"columnas_vacias": vaciasEL},
				Aplicado: map[string]any{"activo": s.Activo, "requiere_revision": true},
				Mensaje: fmt.Sprintf("A %s le faltan atributos (%s). Quedan como desconocidos o nulos y el servicio queda para revision.",
					codigo, strings.Join(vaciasEL, ", ")),
			})
		}
		s.RequiereRevision = revision
	}

	// Nombre de cada nivel 1: el de la primera fila de su bloque, con incidencia si hay
	// mas de uno (D09).
	for _, n1 := range p.N1 {
		sort.Ints(n1.Filas)
		nombre, leidas := h.columna('B', n1.Filas)
		if vacio(nombre) {
			return fmt.Errorf("el nivel 1 %s no tiene nombre en B", n1.Codigo)
		}
		n1.Nombre = nombre
		for c, v := range comoMapa(leidas) {
			n1.Celdas[c] = v
		}
		if distintos(leidas) > 1 {
			p.Incidencias = append(p.Incidencias, Incidencia{
				Tipo: "N1_NOMBRE_CONFLICTO", Regla: "D09", Fila: n1.Filas[0], Celdas: listaCeldas(leidas),
				Codigo: n1.Codigo, N1: n1.Codigo,
				Original: comoMapa(leidas), Aplicado: map[string]string{"nombre": nombre},
				Mensaje: fmt.Sprintf("El nivel 1 %s aparece con %d nombres en B. Vale el de la primera fila del bloque, %q; "+
					"los demas quedan como evidencia.", n1.Codigo, distintos(leidas), nombre),
			})
		}
	}

	p.Controles["n1_distintos"] = len(p.N1)
	p.Controles["n2_distintos"] = len(p.N2)
	return nil
}
