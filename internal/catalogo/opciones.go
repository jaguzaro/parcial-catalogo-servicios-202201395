package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/db"
)

// Catalogo describe una lista de opciones. Los nombres de tabla y columna son constantes
// de este archivo: nunca vienen de la peticion.
type Catalogo struct {
	Recurso  string // segmento de la ruta: "clases"
	Tabla    string // "clase_servicio"
	Columna  string // columna de servicio_n2 que la usa: "clase_id"
	Nombre   string // para mensajes: "clase"
	Femenino bool
}

var (
	Clases       = &Catalogo{Recurso: "clases", Tabla: "clase_servicio", Columna: "clase_id", Nombre: "clase", Femenino: true}
	Criticidades = &Catalogo{Recurso: "criticidades", Tabla: "criticidad", Columna: "criticidad_id", Nombre: "criticidad", Femenino: true}
	Tipos        = &Catalogo{Recurso: "tipos", Tabla: "tipo_servicio", Columna: "tipo_id", Nombre: "tipo"}

	// Catalogos en el orden en que se muestran.
	Catalogos = []*Catalogo{Clases, Criticidades, Tipos}
)

// CatalogoDe busca un catalogo por su segmento de ruta.
func CatalogoDe(recurso string) (*Catalogo, bool) {
	for _, c := range Catalogos {
		if c.Recurso == recurso {
			return c, true
		}
	}
	return nil, false
}

func (c *Catalogo) un() string {
	if c.Femenino {
		return "una " + c.Nombre
	}
	return "un " + c.Nombre
}

func (c *Catalogo) inactivo() string {
	if c.Femenino {
		return "inactiva"
	}
	return "inactivo"
}

// OrdenMaximo es el limite de la columna smallint.
const OrdenMaximo = 32767

// Opcion es un valor de un catalogo. ValorOrigen y OrigenCelda son nulos en las opciones
// creadas desde la aplicacion.
type Opcion struct {
	ID            int64     `json:"id"`
	Nombre        string    `json:"nombre"`
	ValorOrigen   *string   `json:"valor_origen"`
	OrigenCelda   *string   `json:"origen_celda"`
	Orden         int       `json:"orden"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
}

// OpcionMapeo es una opcion cuyo nombre ya no es el del archivo.
type OpcionMapeo struct {
	Catalogo string `json:"catalogo"`
	Opcion
}

// EntradaOpcion es el cuerpo de alta y de modificacion.
type EntradaOpcion struct {
	Nombre Valor
	Orden  Valor
}

func (c *Catalogo) selectOpcion() string {
	return fmt.Sprintf(`SELECT id, nombre, valor_origen, origen_celda, orden, activo, creado_en, actualizado_en FROM %s`, c.Tabla)
}

func escanearOpcion(fila pgx.Row) (*Opcion, error) {
	var o Opcion
	if err := fila.Scan(&o.ID, &o.Nombre, &o.ValorOrigen, &o.OrigenCelda, &o.Orden, &o.Activo, &o.CreadoEn, &o.ActualizadoEn); err != nil {
		return nil, err
	}
	return &o, nil
}

func leerOpcion(ctx context.Context, q consultor, c *Catalogo, id int64) (*Opcion, error) {
	// id es smallint: se compara como bigint para que un id fuera de rango sea "no existe"
	// y no un error de conversion.
	o, err := escanearOpcion(q.QueryRow(ctx, c.selectOpcion()+` WHERE id = $1::bigint`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe %s con id %d.", c.un(), id))
	}
	if err != nil {
		return nil, fmt.Errorf("leer %s %d: %w", c.Tabla, id, err)
	}
	return o, nil
}

// ListarOpciones devuelve la lista completa, sin paginar, en el orden del catalogo.
func (s *Servicio) ListarOpciones(ctx context.Context, c *Catalogo, activo *bool) ([]*Opcion, error) {
	sql, args := c.selectOpcion(), []any{}
	if activo != nil {
		sql += ` WHERE activo = $1`
		args = append(args, *activo)
	}
	filas, err := s.conn.Query(ctx, sql+` ORDER BY orden, nombre, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("listar %s: %w", c.Tabla, err)
	}
	defer filas.Close()
	items := []*Opcion{}
	for filas.Next() {
		o, err := escanearOpcion(filas)
		if err != nil {
			return nil, fmt.Errorf("leer %s: %w", c.Tabla, err)
		}
		items = append(items, o)
	}
	return items, filas.Err()
}

// Mapeo devuelve las opciones cuyo nombre se cambio respecto del valor del archivo.
func (s *Servicio) Mapeo(ctx context.Context) ([]*OpcionMapeo, error) {
	items := []*OpcionMapeo{}
	for _, c := range Catalogos {
		filas, err := s.conn.Query(ctx, c.selectOpcion()+` WHERE valor_origen IS NOT NULL AND nombre <> valor_origen ORDER BY orden, id`)
		if err != nil {
			return nil, fmt.Errorf("mapeo de %s: %w", c.Tabla, err)
		}
		for filas.Next() {
			o, err := escanearOpcion(filas)
			if err != nil {
				filas.Close()
				return nil, fmt.Errorf("mapeo de %s: %w", c.Tabla, err)
			}
			items = append(items, &OpcionMapeo{Catalogo: c.Recurso, Opcion: *o})
		}
		filas.Close()
		if err := filas.Err(); err != nil {
			return nil, fmt.Errorf("mapeo de %s: %w", c.Tabla, err)
		}
	}
	return items, nil
}

func validarOpcion(e EntradaOpcion) (string, int, error) {
	var errs errores
	nombre, _ := errs.nombre(e.Nombre, "nombre", LargoNombre)
	if nombre != "" && strings.TrimSpace(nombre) != nombre {
		errs.falla("nombre", fmt.Sprintf("El nombre %q tiene espacios al inicio o al final.", nombre))
	}
	// null deja n en 0, que tambien se rechaza.
	var n int64
	if !e.Orden.Presente || json.Unmarshal(e.Orden.JSON, &n) != nil || n < 1 || n > OrdenMaximo {
		errs.falla("orden", fmt.Sprintf("El orden es obligatorio: un entero entre 1 y %d.", OrdenMaximo))
	}
	return nombre, int(n), errs.err()
}

func traducirOpcion(c *Catalogo, err error, nombre string) error {
	if estado, _ := db.RestriccionViolada(err); estado == "23505" {
		return db.NuevoError(db.ErrCodigoDuplicado,
			fmt.Sprintf("Ya existe %s con nombre %s. Las opciones dadas de baja tampoco se reutilizan.", c.un(), nombre), "nombre")
	}
	return err
}

// CrearOpcion agrega una opcion activa. valor_origen queda nulo: no viene del archivo.
func (s *Servicio) CrearOpcion(ctx context.Context, c *Catalogo, e EntradaOpcion) (*Opcion, error) {
	nombre, orden, err := validarOpcion(e)
	if err != nil {
		return nil, err
	}
	var o *Opcion
	err = s.enTx(ctx, func(tx pgx.Tx) error {
		var id int64
		if err := tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO %s (nombre, orden) VALUES ($1, $2) RETURNING id`, c.Tabla),
			nombre, orden).Scan(&id); err != nil {
			return traducirOpcion(c, err, nombre)
		}
		o, err = leerOpcion(ctx, tx, c, id)
		return err
	})
	return o, err
}

// bloquearOpcion lee la opcion con FOR UPDATE.
func bloquearOpcion(ctx context.Context, tx pgx.Tx, c *Catalogo, id int64) (activo bool, nombre string, err error) {
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT activo, nombre FROM %s WHERE id = $1::bigint FOR UPDATE`, c.Tabla), id).Scan(&activo, &nombre)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe %s con id %d.", c.un(), id))
	}
	if err != nil {
		return false, "", fmt.Errorf("bloquear %s %d: %w", c.Tabla, id, err)
	}
	return activo, nombre, nil
}

// ActualizarOpcion cambia nombre y orden. valor_origen no se toca: si el nombre deja de
// coincidir, la opcion aparece en el mapeo de etiquetas.
func (s *Servicio) ActualizarOpcion(ctx context.Context, c *Catalogo, id int64, e EntradaOpcion) (*Opcion, error) {
	nombre, orden, err := validarOpcion(e)
	if err != nil {
		return nil, err
	}
	var o *Opcion
	err = s.enTx(ctx, func(tx pgx.Tx) error {
		if _, _, err := bloquearOpcion(ctx, tx, c, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET nombre = $2, orden = $3, actualizado_en = now() WHERE id = $1`, c.Tabla),
			id, nombre, orden); err != nil {
			return traducirOpcion(c, err, nombre)
		}
		o, err = leerOpcion(ctx, tx, c, id)
		return err
	})
	return o, err
}

// DesactivarOpcion la quita de los formularios. Se rechaza mientras la use algun servicio
// con activo S o DESCONOCIDO (D22).
func (s *Servicio) DesactivarOpcion(ctx context.Context, c *Catalogo, id int64) (*Opcion, error) {
	var o *Opcion
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		activo, nombre, err := bloquearOpcion(ctx, tx, c, id)
		if err != nil {
			return err
		}
		if activo {
			refs, total, err := listarRefs(ctx, tx, "servicio_n2",
				fmt.Sprintf(`SELECT id, codigo, nombre FROM servicio_n2 WHERE %s = $1 AND activo <> 'N'`, c.Columna), id)
			if err != nil {
				return err
			}
			if total > 0 {
				return dependenciasActivas(fmt.Sprintf("No se puede desactivar %s %s: la usan %s. Cambie su %s o desactivelos primero.",
					c.Nombre, nombre, cantidad(total, "servicio activo", "servicios activos"), c.Nombre), refs, total)
			}
			if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET activo = false, actualizado_en = now() WHERE id = $1`, c.Tabla), id); err != nil {
				return fmt.Errorf("desactivar %s %d: %w", c.Tabla, id, err)
			}
		}
		o, err = leerOpcion(ctx, tx, c, id)
		return err
	})
	return o, err
}

// ActivarOpcion la devuelve a los formularios. Las opciones no tienen padre.
func (s *Servicio) ActivarOpcion(ctx context.Context, c *Catalogo, id int64) (*Opcion, error) {
	var o *Opcion
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		activo, _, err := bloquearOpcion(ctx, tx, c, id)
		if err != nil {
			return err
		}
		if !activo {
			if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET activo = true, actualizado_en = now() WHERE id = $1`, c.Tabla), id); err != nil {
				return fmt.Errorf("activar %s %d: %w", c.Tabla, id, err)
			}
		}
		o, err = leerOpcion(ctx, tx, c, id)
		return err
	})
	return o, err
}

// verificarOpcion comprueba que una opcion que se fija en un servicio exista y este
// activa. La lee con FOR SHARE para que no la desactiven a la vez.
func verificarOpcion(ctx context.Context, tx pgx.Tx, c *Catalogo, id int64) error {
	var (
		nombre string
		activo bool
	)
	err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT nombre, activo FROM %s WHERE id = $1::bigint FOR SHARE`, c.Tabla), id).Scan(&nombre, &activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.NuevoError(db.ErrReferenciaInexistente,
			fmt.Sprintf("No existe %s con id %d (campo %s).", c.un(), id, c.Columna), c.Columna)
	}
	if err != nil {
		return fmt.Errorf("leer %s %d: %w", c.Tabla, id, err)
	}
	if !activo {
		e := db.NuevoError(db.ErrPadreInactivo,
			fmt.Sprintf("La opcion %s de %s esta %s: no se puede asignar a un servicio.", nombre, c.Nombre, c.inactivo()), c.Columna)
		e.Detalle = db.Ref{Entidad: c.Tabla, ID: id, Codigo: nombre, Nombre: nombre}
		return e
	}
	return nil
}
