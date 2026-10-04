package integracion

import (
	"bytes"
	"context"
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
