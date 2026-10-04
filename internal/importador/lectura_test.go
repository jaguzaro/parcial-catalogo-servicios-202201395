package importador

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// libroEnsayo arma en memoria un libro con la forma del original: encabezados en la fila 4,
// datos desde la 5 y el bloque de opciones debajo. No escribe nada en disco.
func libroEnsayo(t *testing.T, celdas map[string]any, combinar ...[2]string) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", Hoja); err != nil {
		t.Fatal(err)
	}
	for i, e := range encabezados {
		f.SetCellValue(Hoja, ref(byte('A'+i), 4), e)
	}
	opciones := map[byte][]string{'E': {"S", "N"}, 'F': {"A DEMANDA", "RECURRENTE"}, 'G': {"Normal", "High"}, 'H': {"Front End"}}
	for col, vs := range opciones {
		f.SetCellValue(Hoja, ref(col, 20), rotuloOpciones)
		for i, v := range vs {
			f.SetCellValue(Hoja, ref(col, 21+i), v)
		}
	}
	for c, v := range celdas {
		f.SetCellValue(Hoja, c, v)
	}
	for _, r := range combinar {
		if err := f.MergeCell(Hoja, r[0], r[1]); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func contar(p *Plan, tipo string) []Incidencia {
	var r []Incidencia
	for _, i := range p.Incidencias {
		if i.Tipo == tipo {
			r = append(r, i)
		}
	}
	return r
}

// Dos filas de un mismo servicio (C5:C6 combinadas) con ACTIVO distinto: gana la fila
// principal y queda CONFLICTO_ATRIBUTO con los dos valores (D21).
func TestLectura_ConflictoAtributoGanaLaFilaPrincipal(t *testing.T) {
	r := libroEnsayo(t, map[string]any{
		"A5": "SE.01", "B5": "Nivel uno", "C5": "SE.01.01", "D5": "Servicio",
		"E5": "S", "F5": "A DEMANDA", "G5": "Normal", "H5": "Front End",
		"E6": "N", "F6": "A DEMANDA", "G6": "Normal", "H6": "Front End",
	}, [2]string{"A5", "A6"}, [2]string{"B5", "B6"}, [2]string{"C5", "C6"}, [2]string{"D5", "D6"})
	p, err := Analizar(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.N2) != 1 {
		t.Fatalf("servicios de nivel 2 = %d, se esperaba 1 (las filas combinadas son un solo servicio)", len(p.N2))
	}
	s := p.N2[0]
	if s.Activo != "S" {
		t.Errorf("activo = %q, se esperaba S, el de la fila principal", s.Activo)
	}
	if got := strings.Join([]string{s.Celdas["E5"], s.Celdas["E6"]}, ","); got != "S,N" {
		t.Errorf("origen no conserva las dos celdas: %v", s.Celdas)
	}
	c := contar(p, "CONFLICTO_ATRIBUTO")
	if len(c) != 1 {
		t.Fatalf("CONFLICTO_ATRIBUTO = %d, se esperaba 1: %+v", len(c), p.Incidencias)
	}
	orig := c[0].Original.(map[string]string)
	if orig["E5"] != "S" || orig["E6"] != "N" || c[0].Regla != "D21" || c[0].N2 != "SE.01.01" {
		t.Errorf("incidencia incompleta: %+v", c[0])
	}
}

// Una fila sin C fuera de todo rango no se pega al servicio anterior (D10), y un valor
// fuera de catalogo o un minimo mayor que el maximo dejan nulos, no ceros (D11).
func TestLectura_FilaSinCodigoYValoresInvalidos(t *testing.T) {
	r := libroEnsayo(t, map[string]any{
		"A5": "SE.01", "B5": "Nivel uno", "C5": "SE.01.01", "D5": "Servicio",
		"E5": "X", "F5": "OTRA", "G5": "Normal", "H5": "Front End", "K5": 10, "L5": 2,
		"E6": "S", "F6": "A DEMANDA",
	}, [2]string{"A5", "A6"}, [2]string{"B5", "B6"})
	p, err := Analizar(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.N2) != 1 || p.FilasNoImportables != 1 {
		t.Fatalf("n2 = %d, filas no importables = %d; se esperaba 1 y 1", len(p.N2), p.FilasNoImportables)
	}
	s := p.N2[0]
	if s.Activo != "DESCONOCIDO" || s.Clase != nil || s.Minimo != nil || s.Maximo != nil || !s.RequiereRevision {
		t.Errorf("servicio = %+v", s)
	}
	for _, tipo := range []string{"FILA_SIN_CODIGO", "VALOR_FUERA_DE_DOMINIO", "VALOR_FUERA_DE_CATALOGO", "MINIMO_MAYOR_QUE_MAXIMO"} {
		if n := len(contar(p, tipo)); n != 1 {
			t.Errorf("%s = %d, se esperaba 1", tipo, n)
		}
	}
}

func TestLectura_EncabezadoMovidoFalla(t *testing.T) {
	r := libroEnsayo(t, map[string]any{"C4": "CODIGO", "A5": "SE.01", "B5": "x", "C5": "SE.01.01", "D5": "y"})
	if _, err := Analizar(r); err == nil || !strings.Contains(err.Error(), "C4") {
		t.Fatalf("se esperaba error por el encabezado C4, llego %v", err)
	}
}
