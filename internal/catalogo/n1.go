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

// Origen es la trazabilidad de un registro importado: hoja, rango y valores leidos.
type Origen struct {
	Hoja    string          `json:"hoja"`
	Rango   *string         `json:"rango"`
	Valores json.RawMessage `json:"valores"`
}

func origen(hoja, rango, valores *string) *Origen {
	if hoja == nil {
		return nil
	}
	return &Origen{Hoja: *hoja, Rango: rango, Valores: rawONulo(valores)}
}

// Incidencia es una observacion de la importacion sobre un servicio.
type Incidencia struct {
	ID            int64           `json:"id"`
	ImportacionID int64           `json:"importacion_id"`
	Tipo          string          `json:"tipo"`
	Regla         *string         `json:"regla"`
	Hoja          string          `json:"hoja"`
	Fila          *int            `json:"fila"`
	Celdas        *string         `json:"celdas"`
	Codigo        *string         `json:"codigo"`
	ValorOriginal json.RawMessage `json:"valor_original"`
	ValorAplicado json.RawMessage `json:"valor_aplicado"`
	Mensaje       string          `json:"mensaje"`
	CreadaEn      time.Time       `json:"creada_en"`
}

// incidencias lee las de la ultima importacion completada que tocan al servicio. Las de
// corridas anteriores repiten las mismas observaciones y quedan en la ficha de cada
// importacion.
func incidencias(ctx context.Context, c consultor, columna string, id int64) ([]Incidencia, error) {
	filas, err := c.Query(ctx, fmt.Sprintf(`
		SELECT id, importacion_id, tipo, regla, hoja, fila, celdas, codigo, valor_original::text, valor_aplicado::text, mensaje, creada_en
		  FROM incidencia
		 WHERE %s = $1
		   AND importacion_id = (SELECT max(id) FROM importacion WHERE estado = 'completada')
		 ORDER BY fila NULLS LAST, id`, columna), id)
	if err != nil {
		return nil, fmt.Errorf("incidencias de %d: %w", id, err)
	}
	defer filas.Close()
	out := []Incidencia{}
	for filas.Next() {
		var (
			i              Incidencia
			original, apli *string
		)
		if err := filas.Scan(&i.ID, &i.ImportacionID, &i.Tipo, &i.Regla, &i.Hoja, &i.Fila, &i.Celdas, &i.Codigo,
			&original, &apli, &i.Mensaje, &i.CreadaEn); err != nil {
			return nil, fmt.Errorf("incidencias de %d: %w", id, err)
		}
		i.ValorOriginal, i.ValorAplicado = rawONulo(original), rawONulo(apli)
		out = append(out, i)
	}
	return out, filas.Err()
}

// ServicioN1 es un servicio de nivel 1. ServiciosN2 cuenta todos sus hijos, activos o no.
type ServicioN1 struct {
	ID            int64     `json:"id"`
	Codigo        string    `json:"codigo"`
	Nombre        string    `json:"nombre"`
	Activo        bool      `json:"activo"`
	ServiciosN2   int       `json:"servicios_n2"`
	Origen        *Origen   `json:"origen"`
	ImportacionID *int64    `json:"importacion_id"`
	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"actualizado_en"`
	// Solo en la ficha.
	Hijos       *[]ResumenN2  `json:"servicios,omitempty"`
	Incidencias *[]Incidencia `json:"incidencias,omitempty"`
}

// ResumenN2 es un servicio de nivel 2 dentro de la ficha de su nivel 1.
type ResumenN2 struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
	Activo string `json:"activo"`
}

// EntradaN1 es el cuerpo de alta y de modificacion.
type EntradaN1 struct {
	Codigo Valor
	Nombre Valor
}

// FiltroN1 es el filtro del listado.
type FiltroN1 struct {
	Q      string
	Activo *bool
	Pagina db.Pagina
}

const selectN1 = `
SELECT n.id, n.codigo, n.nombre, n.activo,
       (SELECT count(*) FROM servicio_n2 h WHERE h.servicio_n1_id = n.id),
       n.origen_hoja, n.origen_rango, n.origen_valores::text, n.importacion_id, n.creado_en, n.actualizado_en
  FROM servicio_n1 n`

func escanearN1(fila pgx.Row) (*ServicioN1, error) {
	var (
		n                    ServicioN1
		hoja, rango, valores *string
	)
	if err := fila.Scan(&n.ID, &n.Codigo, &n.Nombre, &n.Activo, &n.ServiciosN2,
		&hoja, &rango, &valores, &n.ImportacionID, &n.CreadoEn, &n.ActualizadoEn); err != nil {
		return nil, err
	}
	n.Origen = origen(hoja, rango, valores)
	return &n, nil
}

func noEncontradoN1(id int64) error {
	return db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe un servicio de nivel 1 con id %d.", id))
}

func leerN1(ctx context.Context, c consultor, id int64) (*ServicioN1, error) {
	n, err := escanearN1(c.QueryRow(ctx, selectN1+` WHERE n.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noEncontradoN1(id)
	}
	if err != nil {
		return nil, fmt.Errorf("leer servicio_n1 %d: %w", id, err)
	}
	return n, nil
}

// ListarN1 devuelve una pagina ordenada por codigo.
func (s *Servicio) ListarN1(ctx context.Context, f FiltroN1) (db.Listado[*ServicioN1], error) {
	var (
		conds []string
		args  []any
	)
	if f.Q != "" {
		args = append(args, db.PatronContiene(f.Q))
		conds = append(conds, fmt.Sprintf("(n.codigo ILIKE $%d OR n.nombre ILIKE $%d)", len(args), len(args)))
	}
	if f.Activo != nil {
		args = append(args, *f.Activo)
		conds = append(conds, fmt.Sprintf("n.activo = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.conn.QueryRow(ctx, `SELECT count(*) FROM servicio_n1 n`+where, args...).Scan(&total); err != nil {
		return db.Listado[*ServicioN1]{}, fmt.Errorf("contar servicios n1: %w", err)
	}
	args = append(args, f.Pagina.Tamano, f.Pagina.Desplazamiento())
	filas, err := s.conn.Query(ctx, fmt.Sprintf(`%s%s ORDER BY n.codigo, n.id LIMIT $%d OFFSET $%d`,
		selectN1, where, len(args)-1, len(args)), args...)
	if err != nil {
		return db.Listado[*ServicioN1]{}, fmt.Errorf("listar servicios n1: %w", err)
	}
	defer filas.Close()
	var items []*ServicioN1
	for filas.Next() {
		n, err := escanearN1(filas)
		if err != nil {
			return db.Listado[*ServicioN1]{}, fmt.Errorf("leer servicios n1: %w", err)
		}
		items = append(items, n)
	}
	if err := filas.Err(); err != nil {
		return db.Listado[*ServicioN1]{}, fmt.Errorf("listar servicios n1: %w", err)
	}
	return db.NuevoListado(items, total, f.Pagina), nil
}

// ObtenerN1 devuelve la ficha: el nivel 1, sus servicios de nivel 2 y sus incidencias.
func (s *Servicio) ObtenerN1(ctx context.Context, id int64) (*ServicioN1, error) {
	n, err := leerN1(ctx, s.conn, id)
	if err != nil {
		return nil, err
	}
	filas, err := s.conn.Query(ctx, `SELECT id, codigo, nombre, activo FROM servicio_n2 WHERE servicio_n1_id = $1 ORDER BY codigo, id`, id)
	if err != nil {
		return nil, fmt.Errorf("hijos de servicio_n1 %d: %w", id, err)
	}
	hijos := []ResumenN2{}
	for filas.Next() {
		var h ResumenN2
		if err := filas.Scan(&h.ID, &h.Codigo, &h.Nombre, &h.Activo); err != nil {
			filas.Close()
			return nil, fmt.Errorf("hijos de servicio_n1 %d: %w", id, err)
		}
		hijos = append(hijos, h)
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("hijos de servicio_n1 %d: %w", id, err)
	}
	inc, err := incidencias(ctx, s.conn, "servicio_n1_id", id)
	if err != nil {
		return nil, err
	}
	n.Hijos, n.Incidencias = &hijos, &inc
	return n, nil
}

func validarN1(e EntradaN1) (codigo, nombre string, err error) {
	var errs errores
	codigo, _ = errs.codigo(e.Codigo)
	nombre, _ = errs.nombre(e.Nombre, "nombre", LargoNombre)
	return codigo, nombre, errs.err()
}

func codigoDuplicado(nivel, codigo string) error {
	return db.NuevoError(db.ErrCodigoDuplicado,
		fmt.Sprintf("Ya existe un servicio de %s con codigo %s. Los codigos dados de baja tampoco se reutilizan.", nivel, codigo), "codigo")
}

func traducirN1(err error, codigo string) error {
	if estado, _ := db.RestriccionViolada(err); estado == "23505" {
		return codigoDuplicado("nivel 1", codigo)
	}
	return err
}

// CrearN1 da de alta un servicio de nivel 1 activo.
func (s *Servicio) CrearN1(ctx context.Context, e EntradaN1) (*ServicioN1, error) {
	codigo, nombre, err := validarN1(e)
	if err != nil {
		return nil, err
	}
	var n *ServicioN1
	err = s.enTx(ctx, func(tx pgx.Tx) error {
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO servicio_n1 (codigo, nombre) VALUES ($1, $2) RETURNING id`, codigo, nombre).
			Scan(&id); err != nil {
			return traducirN1(err, codigo)
		}
		n, err = leerN1(ctx, tx, id)
		return err
	})
	return n, err
}

// bloquearN1 lee el nivel 1 con FOR UPDATE.
func bloquearN1(ctx context.Context, tx pgx.Tx, id int64) (activo bool, codigo string, err error) {
	err = tx.QueryRow(ctx, `SELECT activo, codigo FROM servicio_n1 WHERE id = $1 FOR UPDATE`, id).Scan(&activo, &codigo)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", noEncontradoN1(id)
	}
	if err != nil {
		return false, "", fmt.Errorf("bloquear servicio_n1 %d: %w", id, err)
	}
	return activo, codigo, nil
}

// ActualizarN1 cambia codigo y nombre. No toca el estado.
func (s *Servicio) ActualizarN1(ctx context.Context, id int64, e EntradaN1) (*ServicioN1, error) {
	codigo, nombre, err := validarN1(e)
	if err != nil {
		return nil, err
	}
	var n *ServicioN1
	err = s.enTx(ctx, func(tx pgx.Tx) error {
		if _, _, err := bloquearN1(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE servicio_n1 SET codigo = $2, nombre = $3, actualizado_en = now() WHERE id = $1`,
			id, codigo, nombre); err != nil {
			return traducirN1(err, codigo)
		}
		n, err = leerN1(ctx, tx, id)
		return err
	})
	return n, err
}

// DesactivarN1 hace la baja logica. Se rechaza mientras tenga servicios de nivel 2 con
// activo S o DESCONOCIDO (D22), y se dicen cuales.
func (s *Servicio) DesactivarN1(ctx context.Context, id int64) (*ServicioN1, error) {
	var n *ServicioN1
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		activo, codigo, err := bloquearN1(ctx, tx, id)
		if err != nil {
			return err
		}
		if activo {
			refs, total, err := listarRefs(ctx, tx, "servicio_n2",
				`SELECT id, codigo, nombre FROM servicio_n2 WHERE servicio_n1_id = $1 AND activo <> 'N'`, id)
			if err != nil {
				return err
			}
			if total > 0 {
				return dependenciasActivas(fmt.Sprintf(
					"No se puede desactivar el servicio de nivel 1 %s: tiene %s (S o DESCONOCIDO). Desactivelos primero.",
					codigo, cantidad(total, "servicio de nivel 2 activo", "servicios de nivel 2 activos")), refs, total)
			}
			if _, err := tx.Exec(ctx, `UPDATE servicio_n1 SET activo = false, actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("desactivar servicio_n1 %d: %w", id, err)
			}
		}
		n, err = leerN1(ctx, tx, id)
		return err
	})
	return n, err
}

// ActivarN1 reactiva el nivel 1. No reactiva a sus hijos.
func (s *Servicio) ActivarN1(ctx context.Context, id int64) (*ServicioN1, error) {
	var n *ServicioN1
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		activo, _, err := bloquearN1(ctx, tx, id)
		if err != nil {
			return err
		}
		if !activo {
			if _, err := tx.Exec(ctx, `UPDATE servicio_n1 SET activo = true, actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("activar servicio_n1 %d: %w", id, err)
			}
		}
		n, err = leerN1(ctx, tx, id)
		return err
	})
	return n, err
}

// verificarN1 comprueba, con FOR SHARE, que el nivel 1 exista. Devuelve si esta activo.
func verificarN1(ctx context.Context, tx pgx.Tx, id int64) (db.Ref, bool, error) {
	ref := db.Ref{Entidad: "servicio_n1", ID: id}
	var activo bool
	err := tx.QueryRow(ctx, `SELECT codigo, nombre, activo FROM servicio_n1 WHERE id = $1 FOR SHARE`, id).
		Scan(&ref.Codigo, &ref.Nombre, &activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, false, db.NuevoError(db.ErrReferenciaInexistente,
			fmt.Sprintf("No existe un servicio de nivel 1 con id %d (campo servicio_n1_id).", id), "servicio_n1_id")
	}
	if err != nil {
		return ref, false, fmt.Errorf("leer servicio_n1 %d: %w", id, err)
	}
	return ref, activo, nil
}
