package integracion

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/demo"
)

func TestDemo_SembrarEsIdempotente(t *testing.T) {
	suf := sufijo(t)
	clave := contrasenaPrueba()
	entorno := map[string]string{
		"DEMO_ADMIN_USUARIO":       "admin" + suf,
		"DEMO_ADMIN_CONTRASENA":    clave,
		"DEMO_CONSULTA_USUARIO":    "consulta" + suf,
		"DEMO_CONSULTA_CONTRASENA": clave + "2",
	}
	cuentas, err := demo.CargarCuentas(func(k string) (string, bool) { v, ok := entorno[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}

	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		// Las asignaciones de responsable necesitan servicios importados.
		importarEnTx(t, ctx, tx)
		var primera, segunda bytes.Buffer
		if err := demo.Sembrar(ctx, tx, cuentas, &primera); err != nil {
			t.Fatalf("primera corrida: %v", err)
		}
		if err := demo.Sembrar(ctx, tx, cuentas, &segunda); err != nil {
			t.Fatalf("segunda corrida: %v\nsalida de la primera:\n%s", err, primera.String())
		}

		if !strings.Contains(primera.String(), "creado: usuario admin"+suf) ||
			!strings.Contains(primera.String(), "creado: usuario consulta"+suf) {
			t.Errorf("la primera corrida no informa las cuentas creadas:\n%s", primera.String())
		}
		if strings.Contains(segunda.String(), "creado:") {
			t.Errorf("la segunda corrida creo algo:\n%s", segunda.String())
		}
		for _, s := range []string{primera.String(), segunda.String()} {
			if strings.Contains(s, clave) {
				t.Fatal("la salida contiene una contrasena")
			}
		}

		for _, c := range []struct{ sql, desc string }{
			{`SELECT count(*) FROM empresa WHERE codigo = 'DEMO'`, "empresas DEMO"},
			{`SELECT count(*) FROM puesto p JOIN seccion s ON s.id = p.seccion_id JOIN departamento d ON d.id = s.departamento_id
			   JOIN area a ON a.id = d.area_id JOIN empresa e ON e.id = a.empresa_id
			  WHERE e.codigo = 'DEMO' AND a.codigo = 'DEMO' AND d.codigo = 'DEMO' AND s.codigo = 'DEMO' AND p.codigo = 'DEMO'`, "puestos DEMO"},
			{`SELECT count(*) FROM usuario WHERE usuario = 'admin` + suf + `' AND rol = 'administrador'`, "administradores"},
			{`SELECT count(*) FROM usuario WHERE usuario = 'consulta` + suf + `' AND rol = 'consulta'`, "usuarios de consulta"},
		} {
			if n := id(t, ctx, tx, c.sql); n != 1 {
				t.Errorf("%s = %d, se esperaba 1", c.desc, n)
			}
		}

		// Tres asignaciones validas: seccion DEMO activa, servicio activo y usuario de esa
		// misma seccion. La segunda corrida no agrega ninguna.
		if n := strings.Count(primera.String(), "creado: asignacion de responsable"); n != demo.Asignaciones {
			t.Errorf("la primera corrida informa %d asignaciones, se esperaban %d:\n%s", n, demo.Asignaciones, primera.String())
		}
		validas := id(t, ctx, tx, `
			SELECT count(*) FROM servicio_n2 sv
			  JOIN seccion s ON s.id = sv.seccion_responsable_id AND s.codigo = 'DEMO' AND s.activo
			  JOIN usuario u ON u.id = sv.usuario_responsable_id AND u.activo
			  JOIN puesto p  ON p.id = u.puesto_id AND p.seccion_id = s.id
			 WHERE sv.activo = 'S'`)
		if validas < 3 {
			t.Errorf("asignaciones validas = %d, se esperaban al menos 3", validas)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2 WHERE seccion_responsable_id IS NOT NULL`); n != demo.Asignaciones {
			t.Errorf("servicios con responsable = %d tras dos corridas, se esperaban %d", n, demo.Asignaciones)
		}
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2`); n != 46 {
			t.Errorf("servicio_n2 = %d tras sembrar, se esperaban 46", n)
		}
	})
}

func TestDemo_SinServiciosAvisaYFalla(t *testing.T) {
	suf := sufijo(t)
	entorno := map[string]string{
		"DEMO_ADMIN_USUARIO": "admin" + suf, "DEMO_ADMIN_CONTRASENA": contrasenaPrueba(),
		"DEMO_CONSULTA_USUARIO": "consulta" + suf, "DEMO_CONSULTA_CONTRASENA": contrasenaPrueba(),
	}
	cuentas, err := demo.CargarCuentas(func(k string) (string, bool) { v, ok := entorno[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}
	enTx(t, func(ctx context.Context, tx pgx.Tx) {
		if n := id(t, ctx, tx, `SELECT count(*) FROM servicio_n2`); n != 0 {
			t.Fatalf("precondicion: servicio_n2 tiene %d filas (make reset-test)", n)
		}
		var salida bytes.Buffer
		err := demo.Sembrar(ctx, tx, cuentas, &salida)
		if !errors.Is(err, demo.ErrSinServicios) {
			t.Fatalf("sin servicios debia devolver ErrSinServicios; devolvio %v", err)
		}
		if !strings.Contains(salida.String(), "AVISO") || !strings.Contains(err.Error(), "make import") {
			t.Errorf("no dice por que fallo: error %q, salida:\n%s", err, salida.String())
		}
		// La estructura y las cuentas si quedan creadas.
		if n := id(t, ctx, tx, `SELECT count(*) FROM usuario WHERE usuario IN ($1, $2)`, "admin"+suf, "consulta"+suf); n != 2 {
			t.Errorf("cuentas creadas = %d, se esperaban 2", n)
		}
	})
}

func TestDemo_SinContrasenaFalla(t *testing.T) {
	entorno := map[string]string{"DEMO_ADMIN_USUARIO": "a", "DEMO_CONSULTA_USUARIO": "b", "DEMO_CONSULTA_CONTRASENA": "corta"}
	_, err := demo.CargarCuentas(func(k string) (string, bool) { v, ok := entorno[k]; return v, ok })
	if err == nil {
		t.Fatal("sin contrasena de administrador debia fallar")
	}
	for _, esperado := range []string{"DEMO_ADMIN_CONTRASENA", "consulta"} {
		if !strings.Contains(err.Error(), esperado) {
			t.Errorf("el error no menciona %s: %v", esperado, err)
		}
	}
}
