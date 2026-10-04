// Package integracion prueba contra PostgreSQL real que el esquema hace cumplir sus reglas.
package integracion

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"catalogo/internal/db"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		os.Stderr.WriteString("TEST_DATABASE_URL no definida: se omiten las pruebas de integracion\n")
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	if _, _, err := db.Migrar(ctx, url); err != nil {
		os.Stderr.WriteString("migrar: " + err.Error() + "\n")
		os.Exit(1)
	}
	p, err := db.Abrir(ctx, url)
	cancel()
	if err != nil {
		os.Stderr.WriteString("abrir: " + err.Error() + "\n")
		os.Exit(1)
	}
	pool = p.Pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// enTx abre una transaccion que siempre se revierte, para no dejar datos entre pruebas.
func enTx(t *testing.T, f func(ctx context.Context, tx pgx.Tx)) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	f(ctx, tx)
}

func exigir(t *testing.T, ctx context.Context, tx pgx.Tx, sql string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("preparacion fallo: %v\nSQL: %s", err, sql)
	}
}

func id(t *testing.T, ctx context.Context, tx pgx.Tx, sql string, args ...any) int64 {
	t.Helper()
	var v int64
	if err := tx.QueryRow(ctx, sql, args...).Scan(&v); err != nil {
		t.Fatalf("preparacion fallo: %v\nSQL: %s", err, sql)
	}
	return v
}

// rechaza comprueba que la operacion falla con el codigo SQLSTATE y la restriccion esperados.
func rechaza(t *testing.T, ctx context.Context, tx pgx.Tx, sqlstate, restriccion, sql string, args ...any) {
	t.Helper()
	_, err := tx.Exec(ctx, sql, args...)
	if err == nil {
		t.Fatalf("la base acepto la operacion; debia rechazarla con %s (%s)", restriccion, sqlstate)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error que no es de PostgreSQL: %v", err)
	}
	if pgErr.Code != sqlstate || pgErr.ConstraintName != restriccion {
		t.Fatalf("rechazo por otra causa: codigo=%s restriccion=%q mensaje=%q; esperado %s %q",
			pgErr.Code, pgErr.ConstraintName, pgErr.Message, sqlstate, restriccion)
	}
}

const (
	violacionUnica = "23505"
	violacionFK    = "23503"
	violacionCheck = "23514"
)

func TestEsquema_CatorceTablas(t *testing.T) {
	esperadas := []string{
		"empresa", "area", "departamento", "seccion", "puesto", "usuario", "sesion",
		"servicio_n1", "servicio_n2", "clase_servicio", "criticidad", "tipo_servicio",
		"importacion", "incidencia",
	}
	rows, err := pool.Query(context.Background(),
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name <> 'goose_db_version'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	hay := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		hay[n] = true
	}
	for _, n := range esperadas {
		if !hay[n] {
			t.Errorf("falta la tabla %s", n)
		}
	}
	if len(hay) != len(esperadas) {
		t.Errorf("hay %d tablas, se esperaban %d: %v", len(hay), len(esperadas), hay)
	}
}

func TestEsquema_ServicioN1CodigoDuplicado(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		exigir(t, ctx, tx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ('SE.01', 'Uno')`)
		rechaza(t, ctx, tx, violacionUnica, "servicio_n1_codigo_uk",
			`INSERT INTO servicio_n1 (codigo, nombre) VALUES ('SE.01', 'Otro nombre')`)
	})
}

func TestEsquema_AreaCodigoPorEmpresa(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		e1 := id(t, ctx, tx, `INSERT INTO empresa (codigo, nombre) VALUES ('E1', 'Empresa 1') RETURNING id`)
		e2 := id(t, ctx, tx, `INSERT INTO empresa (codigo, nombre) VALUES ('E2', 'Empresa 2') RETURNING id`)
		exigir(t, ctx, tx, `INSERT INTO area (empresa_id, codigo, nombre) VALUES ($1, 'A1', 'Area')`, e1)

		// En otra empresa el mismo codigo si se permite.
		exigir(t, ctx, tx, `INSERT INTO area (empresa_id, codigo, nombre) VALUES ($1, 'A1', 'Area')`, e2)

		// Dentro de la misma empresa, no.
		rechaza(t, ctx, tx, violacionUnica, "area_codigo_uk",
			`INSERT INTO area (empresa_id, codigo, nombre) VALUES ($1, 'A1', 'Repetida')`, e1)
	})
}

func TestEsquema_HijoConPadreInexistente(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		rechaza(t, ctx, tx, violacionFK, "area_empresa_fk",
			`INSERT INTO area (empresa_id, codigo, nombre) VALUES (999999999, 'A1', 'Huerfana')`)
	})
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		rechaza(t, ctx, tx, violacionFK, "servicio_n2_n1_fk",
			`INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre) VALUES (999999999, 'SE.99.01', 'Huerfano')`)
	})
}

func TestEsquema_ServicioN2ActivoValoresValidos(t *testing.T) {
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		n1 := id(t, ctx, tx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ('SE.01', 'Uno') RETURNING id`)
		for i, v := range []string{"S", "N", "DESCONOCIDO"} {
			exigir(t, ctx, tx, `INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre, activo) VALUES ($1, $2, 'ok', $3)`,
				n1, "SE.01.0"+string(rune('1'+i)), v)
		}
	})
}

func TestEsquema_ServicioN2ActivoFueraDeDominio(t *testing.T) {
	for i, malo := range []string{"X", "s", "Si", ""} {
		enTx(t, func(ctx context.Context, tx pgx.Tx) {
			n1 := id(t, ctx, tx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ('SE.01', 'Uno') RETURNING id`)
			rechaza(t, ctx, tx, violacionCheck, "servicio_n2_activo_ck",
				`INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre, activo) VALUES ($1, $2, 'mal', $3)`,
				n1, "SE.01.9"+string(rune('0'+i)), malo)
		})
	}
}

func TestEsquema_MinimoMayorQueMaximo(t *testing.T) {
	ins := `INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre, minimo, maximo) VALUES ($1, $2, 'm', $3, $4)`
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		n1 := id(t, ctx, tx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ('SE.01', 'Uno') RETURNING id`)
		// Aceptados: ordenados, iguales, y cualquiera de los dos ausente.
		exigir(t, ctx, tx, ins, n1, "SE.01.01", 1, 100)
		exigir(t, ctx, tx, ins, n1, "SE.01.02", 5, 5)
		exigir(t, ctx, tx, ins, n1, "SE.01.03", 10, nil)
		exigir(t, ctx, tx, ins, n1, "SE.01.04", nil, 10)
		exigir(t, ctx, tx, ins, n1, "SE.01.05", nil, nil)

		// Un faltante no se convierte en cero.
		var minimo *float64
		if err := tx.QueryRow(ctx, `SELECT minimo::float8 FROM servicio_n2 WHERE codigo = 'SE.01.04'`).Scan(&minimo); err != nil {
			t.Fatal(err)
		}
		if minimo != nil {
			t.Fatalf("el minimo ausente se guardo como %v, debia quedar NULL", *minimo)
		}

		rechaza(t, ctx, tx, violacionCheck, "servicio_n2_minmax_ck", ins, n1, "SE.01.06", 24, 12)
	})
}
