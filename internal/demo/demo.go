// Package demo siembra la estructura organizacional de demostracion y las cuentas de
// evaluacion. Ver docs/diseno/reglas.md, "Cuentas de evaluacion".
//
// Todo lo que crea lleva el codigo DEMO. Es idempotente: lo que ya existe no se toca, ni
// siquiera la contrasena de una cuenta existente.
package demo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/auth"
)

// Codigo es el codigo de todas las unidades de demostracion.
const Codigo = "DEMO"

// Cuenta es una cuenta de evaluacion leida del entorno.
type Cuenta struct {
	Usuario    string
	Contrasena string
	Rol        string
	Nombre     string
}

// CargarCuentas lee y valida las cuentas de las variables DEMO_*. Si falta algo, falla:
// no inventa una contrasena.
func CargarCuentas(lookup func(string) (string, bool)) ([]Cuenta, error) {
	var problemas []string
	leer := func(nombre string) string {
		v, _ := lookup(nombre)
		v = strings.TrimSpace(v)
		if v == "" {
			problemas = append(problemas, "falta la variable "+nombre)
		}
		return v
	}
	cuentas := []Cuenta{
		{Usuario: leer("DEMO_ADMIN_USUARIO"), Contrasena: leer("DEMO_ADMIN_CONTRASENA"),
			Rol: auth.RolAdministrador, Nombre: "Administrador de demostracion"},
		{Usuario: leer("DEMO_CONSULTA_USUARIO"), Contrasena: leer("DEMO_CONSULTA_CONTRASENA"),
			Rol: auth.RolConsulta, Nombre: "Consulta de demostracion"},
	}
	for _, c := range cuentas {
		if strings.Contains(c.Usuario, "@") {
			problemas = append(problemas, fmt.Sprintf("el usuario de la cuenta %s no puede llevar @", c.Rol))
		}
		if c.Contrasena != "" && auth.ValidarLargo(c.Contrasena) != nil {
			problemas = append(problemas, fmt.Sprintf("la contrasena de la cuenta %s: %v", c.Rol, auth.ErrLargoContrasena))
		}
	}
	if cuentas[0].Usuario != "" && strings.EqualFold(cuentas[0].Usuario, cuentas[1].Usuario) {
		problemas = append(problemas, "DEMO_ADMIN_USUARIO y DEMO_CONSULTA_USUARIO no pueden ser iguales")
	}
	if len(problemas) > 0 {
		return nil, errors.New("cuentas de demostracion: " + strings.Join(problemas, "; "))
	}
	return cuentas, nil
}

// Conexion es lo que Sembrar necesita de la base: un *pgxpool.Pool, o una pgx.Tx en las
// pruebas, para revertir todo al final.
type Conexion interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Sembrar crea, en una transaccion, lo que falte y escribe en salida que creo y que ya
// existia. Nunca escribe contrasenas.
func Sembrar(ctx context.Context, pool Conexion, cuentas []Cuenta, salida io.Writer) error {
	// Los hashes se calculan antes de abrir la transaccion: argon2id tarda.
	hashes := make([]string, len(cuentas))
	for i, c := range cuentas {
		var existe bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM usuario WHERE lower(usuario) = lower($1))`,
			c.Usuario).Scan(&existe); err != nil {
			return fmt.Errorf("buscar usuario %s: %w", c.Usuario, err)
		}
		if existe {
			continue
		}
		h, err := auth.HashContrasena(c.Contrasena)
		if err != nil {
			return fmt.Errorf("cuenta %s: %w", c.Rol, err)
		}
		hashes[i] = h
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		nombre := "Empresa de demostracion"
		empresa, err := unidad(ctx, tx, salida, "empresa",
			`INSERT INTO empresa (codigo, nombre) VALUES ($1, $2) ON CONFLICT (codigo) DO NOTHING RETURNING id`,
			`SELECT id, activo FROM empresa WHERE codigo = $1`, nil, nombre)
		if err != nil {
			return err
		}
		area, err := unidad(ctx, tx, salida, "area",
			`INSERT INTO area (empresa_id, codigo, nombre) VALUES ($3, $1, $2) ON CONFLICT (empresa_id, codigo) DO NOTHING RETURNING id`,
			`SELECT id, activo FROM area WHERE codigo = $1 AND empresa_id = $2`, &empresa, "Area de demostracion")
		if err != nil {
			return err
		}
		depto, err := unidad(ctx, tx, salida, "departamento",
			`INSERT INTO departamento (area_id, codigo, nombre) VALUES ($3, $1, $2) ON CONFLICT (area_id, codigo) DO NOTHING RETURNING id`,
			`SELECT id, activo FROM departamento WHERE codigo = $1 AND area_id = $2`, &area, "Departamento de demostracion")
		if err != nil {
			return err
		}
		seccion, err := unidad(ctx, tx, salida, "seccion",
			`INSERT INTO seccion (departamento_id, codigo, nombre) VALUES ($3, $1, $2) ON CONFLICT (departamento_id, codigo) DO NOTHING RETURNING id`,
			`SELECT id, activo FROM seccion WHERE codigo = $1 AND departamento_id = $2`, &depto, "Seccion de demostracion")
		if err != nil {
			return err
		}
		puesto, err := unidad(ctx, tx, salida, "puesto",
			`INSERT INTO puesto (seccion_id, codigo, nombre) VALUES ($3, $1, $2) ON CONFLICT (seccion_id, codigo) DO NOTHING RETURNING id`,
			`SELECT id, activo FROM puesto WHERE codigo = $1 AND seccion_id = $2`, &seccion, "Puesto de demostracion")
		if err != nil {
			return err
		}

		for i, c := range cuentas {
			if hashes[i] == "" {
				var rol string
				if err := tx.QueryRow(ctx, `SELECT rol FROM usuario WHERE lower(usuario) = lower($1)`, c.Usuario).Scan(&rol); err != nil {
					return fmt.Errorf("leer usuario %s: %w", c.Usuario, err)
				}
				nota := ""
				if rol != c.Rol {
					nota = fmt.Sprintf(" (AVISO: tiene rol %s, se esperaba %s; no se modifica)", rol, c.Rol)
				}
				fmt.Fprintf(salida, "ya existia: usuario %s, rol %s; no se modifica, tampoco su contrasena%s\n", c.Usuario, rol, nota)
				continue
			}
			tag, err := tx.Exec(ctx,
				`INSERT INTO usuario (puesto_id, nombre, usuario, hash_contrasena, rol)
				 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
				puesto, c.Nombre, c.Usuario, hashes[i], c.Rol)
			if err != nil {
				return fmt.Errorf("crear usuario %s: %w", c.Usuario, err)
			}
			if tag.RowsAffected() == 0 {
				fmt.Fprintf(salida, "ya existia: usuario %s; no se modifica\n", c.Usuario)
				continue
			}
			fmt.Fprintf(salida, "creado: usuario %s, rol %s, en el puesto %s\n", c.Usuario, c.Rol, Codigo)
		}
		return nil
	})
}

// unidad inserta una unidad DEMO si no existe y devuelve su id. insertar recibe
// ($1 codigo, $2 nombre[, $3 padre]) y buscar ($1 codigo[, $2 padre]).
func unidad(ctx context.Context, tx pgx.Tx, salida io.Writer, entidad, insertar, buscar string, padre *int64, nombre string) (int64, error) {
	argsIns := []any{Codigo, nombre}
	argsBus := []any{Codigo}
	if padre != nil {
		argsIns = append(argsIns, *padre)
		argsBus = append(argsBus, *padre)
	}
	var id int64
	err := tx.QueryRow(ctx, insertar, argsIns...).Scan(&id)
	if err == nil {
		fmt.Fprintf(salida, "creado: %s %s\n", entidad, Codigo)
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("crear %s %s: %w", entidad, Codigo, err)
	}
	var activo bool
	if err := tx.QueryRow(ctx, buscar, argsBus...).Scan(&id, &activo); err != nil {
		return 0, fmt.Errorf("leer %s %s: %w", entidad, Codigo, err)
	}
	if !activo {
		return 0, fmt.Errorf("%s %s existe pero esta inactivo: no se cuelga nada de un padre inactivo; reactivelo primero", entidad, Codigo)
	}
	fmt.Fprintf(salida, "ya existia: %s %s\n", entidad, Codigo)
	return id, nil
}
