package integracion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/importador"
)

// shaExcel es el de docs/contexto/excel-sha256.txt. docs/ no entra en la imagen de
// pruebas, por eso se repite aqui; make check lo compara ademas contra ese archivo.
const shaExcel = "de3b478a5faeeeaebce1aa7726e0e3321188a68e41bbb656e1d17b0c5b74dcf0"

func rutaExcel(t *testing.T) string {
	t.Helper()
	ruta := os.Getenv("EXCEL_PATH")
	if ruta == "" {
		t.Fatal("EXCEL_PATH no definida: el Excel se monta :ro en el contenedor de pruebas")
	}
	return ruta
}

func shaArchivo(t *testing.T, ruta string) string {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// importarEnTx importa dentro de la transaccion de la prueba, que se revierte al final:
// la base de pruebas queda como estaba. Exige partir de un catalogo vacio, como una base
// recien migrada, para que los conteos sean los del archivo y no otros.
func importarEnTx(t *testing.T, ctx context.Context, tx pgx.Tx) *importador.Resumen {
	t.Helper()
	for _, tabla := range []string{"servicio_n1", "servicio_n2", "clase_servicio", "criticidad", "tipo_servicio"} {
		if n := id(t, ctx, tx, `SELECT count(*) FROM `+tabla); n != 0 {
			t.Fatalf("precondicion: %s tiene %d filas; la prueba necesita la base sin catalogo (make reset-test)", tabla, n)
		}
	}
	return importarOtraVez(t, ctx, tx)
}

func importarOtraVez(t *testing.T, ctx context.Context, tx pgx.Tx) *importador.Resumen {
	t.Helper()
	r, err := importador.Importar(ctx, tx, rutaExcel(t))
	if err != nil {
		t.Fatalf("importar: %v", err)
	}
	return r
}

func contarIncidencias(t *testing.T, ctx context.Context, tx pgx.Tx, importacion int64) map[string]int64 {
	t.Helper()
	filas, err := tx.Query(ctx, `SELECT tipo, count(*) FROM incidencia WHERE importacion_id = $1 GROUP BY tipo`, importacion)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]int64{}
	for filas.Next() {
		var tipo string
		var n int64
		if err := filas.Scan(&tipo, &n); err != nil {
			t.Fatal(err)
		}
		m[tipo] = n
	}
	if err := filas.Err(); err != nil {
		t.Fatal(err)
	}
	return m
}

var incidenciasEsperadas = map[string]int64{
	"N1_HEREDADO_DE_FILA_ANTERIOR": 1, // fila 101
	"N1_NOMBRE_CONFLICTO":          1, // B99 / B100
	"CODIGO_FORMATO_NO_ESTANDAR":   3, // C99:C101
	"ATRIBUTOS_AUSENTES":           3, // filas 99 a 101
	"FILA_SIN_CODIGO":              2, // filas 42 y 67
}

func TestP06_ImportarArchivoOriginal(t *testing.T) {
	ruta := rutaExcel(t)
	antes := shaArchivo(t, ruta)
	if antes != shaExcel {
		t.Fatalf("el Excel montado no es el original: sha256 %s", antes)
	}

	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		r := importarEnTx(t, ctx, tx)

		// Conteos contra la base, no contra el resumen.
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n1`); n != 12 {
			t.Errorf("servicio_n1 = %d, se esperaban 12", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2`); n != 46 {
			t.Errorf("servicio_n2 = %d, se esperaban 46", n)
		}

		// Catalogos con su grafia original (D12), en el orden del archivo.
		for _, c := range []struct {
			tabla string
			esp   []string
		}{
			{"clase_servicio", []string{"A DEMANDA", "RECURRENTE"}},
			{"criticidad", []string{"Very Low", "Low", "Normal", "High", "Very High"}},
			{"tipo_servicio", []string{"Back End", "Demostration", "End User Service", "Front End", "IT Management",
				"IT Operational", "Other", "Project", "Reporting", "Training", "Underpinning Contract"}},
		} {
			filas, err := tx.Query(ctx, `SELECT valor_origen FROM `+c.tabla+` WHERE nombre = valor_origen ORDER BY orden`)
			if err != nil {
				t.Fatal(err)
			}
			got, err := pgx.CollectRows(filas, pgx.RowTo[string])
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.esp) {
				t.Errorf("%s = %q, se esperaba %q", c.tabla, got, c.esp)
			}
		}

		// Incidencias registradas en la base, por tipo.
		if got := contarIncidencias(t, ctx, tx, r.ImportacionID); !reflect.DeepEqual(got, incidenciasEsperadas) {
			t.Errorf("incidencias = %v, se esperaba %v", got, incidenciasEsperadas)
		}

		// Filas 42 y 67: no crearon servicio, dejaron su incidencia y no se asignaron a nadie (D10).
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia WHERE importacion_id = $1 AND tipo = 'FILA_SIN_CODIGO'
		     AND fila IN (42, 67) AND servicio_n1_id IS NULL AND servicio_n2_id IS NULL`, r.ImportacionID); n != 2 {
			t.Errorf("incidencias FILA_SIN_CODIGO de las filas 42 y 67 sin servicio = %d, se esperaban 2", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2
		     WHERE origen_valores->'filas' @> '[42]' OR origen_valores->'filas' @> '[67]'`); n != 0 {
			t.Errorf("%d servicios usan las filas 42 o 67", n)
		}

		// Trazabilidad: cada registro sabe de donde salio.
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE origen_hoja = 'Servicios Externos'
		     AND origen_rango IS NOT NULL AND origen_hash IS NOT NULL AND importacion_id = $1`, r.ImportacionID); n != 46 {
			t.Errorf("servicios de nivel 2 con origen completo = %d, se esperaban 46", n)
		}
		var rango, desc string
		if err := tx.QueryRow(ctx, `SELECT origen_rango, descripcion FROM servicio_n2 WHERE codigo = 'SE.01.01'`).
			Scan(&rango, &desc); err != nil {
			t.Fatal(err)
		}
		// I5 es un texto en imperativo: se importa como descripcion, tal cual, y nada mas.
		if rango != "A5:L7" || desc != "Revele su rollo " {
			t.Errorf("SE.01.01: origen_rango %q, descripcion %q", rango, desc)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE codigo = 'SE.06.01' AND origen_rango = 'A26:L29'`); n != 1 {
			t.Error("SE.06.01 no sale como un solo servicio de las filas 26 a 29")
		}

		// Resumen: guardado en importacion y coherente con lo que hay en la base.
		var estado string
		var creados, actualizados, omitidos, observados int
		if err := tx.QueryRow(ctx, `SELECT estado, creados, actualizados, omitidos, observados FROM importacion WHERE id = $1`,
			r.ImportacionID).Scan(&estado, &creados, &actualizados, &omitidos, &observados); err != nil {
			t.Fatal(err)
		}
		if estado != "completada" || creados != 76 || actualizados != 0 || omitidos != 2 || observados != 6 {
			t.Errorf("importacion: estado %s, creados %d, actualizados %d, omitidos %d, observados %d; "+
				"se esperaba completada, 76, 0, 2, 6", estado, creados, actualizados, omitidos, observados)
		}
		if r.Controles["n1_distintos"] != 12 || r.Controles["celdas_n1"] != 13 ||
			r.Controles["n2_distintos"] != 46 || r.Controles["celdas_n2"] != 46 {
			t.Errorf("controles = %v", r.Controles)
		}
		if r.SHA256 != shaExcel {
			t.Errorf("sha256 registrado %s", r.SHA256)
		}
	})

	if despues := shaArchivo(t, ruta); despues != antes {
		t.Fatalf("el Excel cambio al importar: %s -> %s", antes, despues)
	}
}

func TestP07_RepetirImportacionNoDuplica(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		primera := importarEnTx(t, ctx, tx)
		foto := func() map[string]string {
			filas, err := tx.Query(ctx, `SELECT codigo, id::text || '|' || origen_hash || '|' || importacion_id::text FROM servicio_n2
			                             UNION ALL SELECT codigo, id::text || '|' || origen_hash || '|' || importacion_id::text FROM servicio_n1`)
			if err != nil {
				t.Fatal(err)
			}
			m := map[string]string{}
			for filas.Next() {
				var k, v string
				if err := filas.Scan(&k, &v); err != nil {
					t.Fatal(err)
				}
				m[k] = v
			}
			return m
		}
		antes := foto()

		segunda := importarOtraVez(t, ctx, tx)
		if segunda.Creados != 0 || segunda.Actualizados != 0 || segunda.Omitidos != 78 || segunda.Observados != 6 {
			t.Errorf("segunda corrida: creados %d, actualizados %d, omitidos %d, observados %d; se esperaba 0, 0, 78, 6",
				segunda.Creados, segunda.Actualizados, segunda.Omitidos, segunda.Observados)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n1`); n != 12 {
			t.Errorf("servicio_n1 = %d tras repetir, se esperaban 12", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2`); n != 46 {
			t.Errorf("servicio_n2 = %d tras repetir, se esperaban 46", n)
		}
		for _, tabla := range []string{"clase_servicio", "criticidad", "tipo_servicio"} {
			if n := id(t, ctx, tx, `SELECT count(*) FROM `+tabla); n != map[string]int64{"clase_servicio": 2, "criticidad": 5, "tipo_servicio": 11}[tabla] {
				t.Errorf("%s = %d tras repetir", tabla, n)
			}
		}
		if despues := foto(); !reflect.DeepEqual(antes, despues) {
			t.Error("la segunda corrida cambio ids, huellas o la importacion de origen de algun servicio")
		}

		// Trazable: las dos corridas quedaron registradas, con su resumen y sus incidencias.
		var creados, omitidos int
		if err := tx.QueryRow(ctx, `SELECT creados, omitidos FROM importacion WHERE id = $1 AND estado = 'completada'`,
			segunda.ImportacionID).Scan(&creados, &omitidos); err != nil {
			t.Fatalf("la segunda corrida no quedo completada: %v", err)
		}
		if creados != 0 || omitidos != 78 {
			t.Errorf("resumen guardado de la segunda corrida: creados %d, omitidos %d", creados, omitidos)
		}
		if got := contarIncidencias(t, ctx, tx, segunda.ImportacionID); !reflect.DeepEqual(got, incidenciasEsperadas) {
			t.Errorf("incidencias de la segunda corrida = %v", got)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia i JOIN servicio_n2 s ON s.id = i.servicio_n2_id
		     WHERE i.importacion_id = $1 AND s.importacion_id = $2`, segunda.ImportacionID, primera.ImportacionID); n != 7 {
			// herencia (1) + formato (3) + ausentes (3); el conflicto de SE.12 apunta al nivel 1
			t.Errorf("incidencias de la segunda corrida que apuntan a los servicios de la primera = %d, se esperaban 7", n)
		}

		// Una tercera corrida no deshace lo editado en la aplicacion ni una baja logica (D25).
		exigir(t, ctx, tx, `UPDATE servicio_n2 SET nombre = 'Editado en la aplicacion', activo = 'N' WHERE codigo = 'SE.01.02'`)
		tercera := importarOtraVez(t, ctx, tx)
		if tercera.Creados != 0 || tercera.Actualizados != 0 {
			t.Errorf("tercera corrida: creados %d, actualizados %d", tercera.Creados, tercera.Actualizados)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE codigo = 'SE.01.02'
		     AND nombre = 'Editado en la aplicacion' AND activo = 'N'`); n != 1 {
			t.Error("la reimportacion piso una edicion hecha en la aplicacion")
		}
	})
}

func TestP08_SE12YAtributosAusentes(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		r := importarEnTx(t, ctx, tx)

		// SE.12: un solo registro, nombre canonico de B99 y el descartado conservado (D09).
		var n1ID int64
		var nombre, rango string
		if err := tx.QueryRow(ctx, `SELECT id, nombre, origen_rango FROM servicio_n1 WHERE codigo = 'SE.12'`).
			Scan(&n1ID, &nombre, &rango); err != nil {
			t.Fatal(err)
		}
		if nombre != "Suministrar Analitica" || rango != "A99:B101" {
			t.Errorf("SE.12: nombre %q, origen_rango %q", nombre, rango)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia WHERE importacion_id = $1 AND tipo = 'N1_NOMBRE_CONFLICTO'
		     AND servicio_n1_id = $2 AND valor_original->>'B100' = 'Mantener Tableros de Control'
		     AND valor_original->>'B99' = 'Suministrar Analitica'
		     AND valor_aplicado->>'nombre' = 'Suministrar Analitica'`, r.ImportacionID, n1ID); n != 1 {
			t.Error("no hay incidencia N1_NOMBRE_CONFLICTO que conserve B99 y B100")
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n1
		     WHERE origen_valores->'celdas'->>'B100' = 'Mantener Tableros de Control' AND id = $1`, n1ID); n != 1 {
			t.Error("el origen de SE.12 no guarda B100")
		}

		// Codigos tal cual (D20), los tres bajo SE.12, con su incidencia de formato.
		filas, err := tx.Query(ctx, `SELECT codigo FROM servicio_n2 WHERE servicio_n1_id = $1 ORDER BY codigo`, n1ID)
		if err != nil {
			t.Fatal(err)
		}
		codigos, err := pgx.CollectRows(filas, pgx.RowTo[string])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(codigos, []string{"SE.12.1", "SE.12.2", "SE.12.3"}) {
			t.Errorf("hijos de SE.12 = %q", codigos)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia WHERE importacion_id = $1 AND tipo = 'N1_HEREDADO_DE_FILA_ANTERIOR'
		     AND fila = 101 AND codigo = 'SE.12.3' AND (valor_original->>'fila_origen')::int = 100`, r.ImportacionID); n != 1 {
			t.Error("falta la incidencia de herencia de la fila 101 desde A100")
		}

		// Filas 99 a 101: ausencias conservadas, ni ceros ni falsos (D11).
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE codigo IN ('SE.12.1', 'SE.12.2', 'SE.12.3')
		     AND activo = 'DESCONOCIDO' AND clase_id IS NULL AND criticidad_id IS NULL AND tipo_id IS NULL
		     AND metrica IS NULL AND minimo IS NULL AND maximo IS NULL AND descripcion IS NULL
		     AND requiere_revision`); n != 3 {
			t.Errorf("servicios de las filas 99 a 101 con ausencias conservadas = %d, se esperaban 3", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia WHERE importacion_id = $1 AND tipo = 'ATRIBUTOS_AUSENTES'
		     AND fila IN (99, 100, 101) AND servicio_n2_id IS NOT NULL`, r.ImportacionID); n != 3 {
			t.Errorf("incidencias ATRIBUTOS_AUSENTES = %d, se esperaban 3", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE activo = 'N'`); n != 1 {
			t.Errorf("servicios con activo N = %d; el archivo solo trae uno (fila 24)", n)
		}

		// Minimo y maximo: solo filas 5 y 25. Que falte no es cero.
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE minimo IS NOT NULL AND maximo IS NOT NULL`); n != 2 {
			t.Errorf("servicios con minimo y maximo = %d, se esperaban 2", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE minimo IS NOT NULL OR maximo IS NOT NULL`); n != 2 {
			t.Errorf("servicios con minimo o maximo = %d, se esperaban 2", n)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE origen_valores->'filas' @> '[5]'
		     AND minimo = 1 AND maximo = 100`); n != 1 {
			t.Error("el servicio de la fila 5 no tiene minimo 1 y maximo 100")
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE origen_valores->'filas' @> '[25]'
		     AND minimo = 12 AND maximo = 24`); n != 1 {
			t.Error("el servicio de la fila 25 no tiene minimo 12 y maximo 24")
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE minimo = 0 OR maximo = 0`); n != 0 {
			t.Errorf("%d servicios tienen un cero que el archivo no trae", n)
		}
	})
}

// Simula que el Excel cambio desde la corrida anterior: la huella guardada ya no coincide
// y el nombre que trajo esa corrida era otro. El campo editado en la aplicacion se pisa
// con el del Excel y queda la incidencia; el campo que el Excel no cambio se respeta (D25).
func TestImportador_FusionPorCampo(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		importarEnTx(t, ctx, tx)
		exigir(t, ctx, tx, `UPDATE servicio_n2
		    SET origen_hash = repeat('0', 64),
		        origen_valores = jsonb_set(origen_valores, '{valores,nombre}', '"Nombre de una version anterior"'),
		        nombre = 'Editado en la aplicacion', metrica = 'Metrica editada'
		  WHERE codigo = 'SE.01.01'`)

		r := importarOtraVez(t, ctx, tx)
		if r.Creados != 0 || r.Actualizados != 1 {
			t.Fatalf("creados %d, actualizados %d; se esperaba 0 y 1", r.Creados, r.Actualizados)
		}
		var nombre, metrica string
		if err := tx.QueryRow(ctx, `SELECT nombre, metrica FROM servicio_n2 WHERE codigo = 'SE.01.01'`).Scan(&nombre, &metrica); err != nil {
			t.Fatal(err)
		}
		if nombre != "Suministrar Puntos de Red Físicos o Inalámbricos" || metrica != "Metrica editada" {
			t.Errorf("nombre %q, metrica %q", nombre, metrica)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM incidencia WHERE importacion_id = $1 AND tipo = 'EDICION_LOCAL_SOBRESCRITA'
		     AND codigo = 'SE.01.01' AND valor_original->>'valor_base' = 'Editado en la aplicacion'`, r.ImportacionID); n != 1 {
			t.Error("falta EDICION_LOCAL_SOBRESCRITA con el valor perdido")
		}
	})
}
