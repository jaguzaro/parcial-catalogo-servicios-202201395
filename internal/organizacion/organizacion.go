package organizacion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"catalogo/internal/db"
)

// Limites de longitud de los campos de texto, en caracteres. Los fija este paquete: el
// diseno no los da y sin limite un nombre podria ocupar el megabyte entero del cuerpo.
const (
	LargoCodigo = 50
	LargoNombre = 200
)

// Unidad resume un registro: el padre de un item, por ejemplo.
type Unidad struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

// Ancestro es un nivel de la ruta de ancestros de una ficha.
type Ancestro struct {
	Entidad string `json:"entidad"`
	ID      int64  `json:"id"`
	Codigo  string `json:"codigo"`
	Nombre  string `json:"nombre"`
	Activo  bool   `json:"activo"`
}

// Registro es una unidad organizacional. En JSON el padre sale con su campo real
// (empresa_id y empresa, area_id y area, ...).
type Registro struct {
	Nivel         *Nivel
	ID            int64
	Codigo        string
	Nombre        string
	Activo        bool
	PadreID       int64
	Padre         *Unidad
	Ancestros     []Ancestro // solo en la ficha
	CreadoEn      time.Time
	ActualizadoEn time.Time
}

// MarshalJSON escribe el registro con el nombre real del campo padre.
func (r Registro) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"id":             r.ID,
		"codigo":         r.Codigo,
		"nombre":         r.Nombre,
		"activo":         r.Activo,
		"creado_en":      r.CreadoEn,
		"actualizado_en": r.ActualizadoEn,
	}
	if p := r.Nivel.Padre; p != nil {
		m[r.Nivel.CampoPadre] = r.PadreID
		m[p.Tabla] = r.Padre
	}
	if r.Ancestros != nil {
		m["ancestros"] = r.Ancestros
	}
	return json.Marshal(m)
}

// Entrada es el cuerpo de alta y de modificacion. PadreID es 0 en la empresa.
type Entrada struct {
	Codigo  string
	Nombre  string
	PadreID int64
}

// Filtro de un listado.
type Filtro struct {
	PadreID *int64
	Q       string
	Activo  *bool
	Pagina  db.Pagina
}

// Servicio aplica las reglas de la estructura organizacional.
type Servicio struct {
	pool *pgxpool.Pool
}

// Nuevo crea el servicio.
func Nuevo(pool *pgxpool.Pool) *Servicio { return &Servicio{pool: pool} }

// consultor es lo comun entre el pool y una transaccion.
type consultor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ---------- lectura ----------

// selectRegistro devuelve las columnas y el FROM de un registro con su padre.
func (n *Nivel) selectRegistro() string {
	if n.Padre == nil {
		return fmt.Sprintf(`SELECT t.id, t.codigo, t.nombre, t.activo, t.creado_en, t.actualizado_en,
		        0::bigint, ''::text, ''::text FROM %s t`, n.Tabla)
	}
	return fmt.Sprintf(`SELECT t.id, t.codigo, t.nombre, t.activo, t.creado_en, t.actualizado_en,
	        p.id, p.codigo, p.nombre FROM %s t JOIN %s p ON p.id = t.%s`, n.Tabla, n.Padre.Tabla, n.CampoPadre)
}

func escanear(n *Nivel, fila pgx.Row) (*Registro, error) {
	r := &Registro{Nivel: n}
	var padre Unidad
	if err := fila.Scan(&r.ID, &r.Codigo, &r.Nombre, &r.Activo, &r.CreadoEn, &r.ActualizadoEn,
		&padre.ID, &padre.Codigo, &padre.Nombre); err != nil {
		return nil, err
	}
	if n.Padre != nil {
		r.PadreID = padre.ID
		r.Padre = &padre
	}
	return r, nil
}

func leer(ctx context.Context, c consultor, n *Nivel, id int64) (*Registro, error) {
	r, err := escanear(n, c.QueryRow(ctx, n.selectRegistro()+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noEncontrado(n, id)
	}
	if err != nil {
		return nil, fmt.Errorf("leer %s %d: %w", n.Tabla, id, err)
	}
	return r, nil
}

// Listar devuelve una pagina, ordenada por codigo.
func (s *Servicio) Listar(ctx context.Context, n *Nivel, f Filtro) (db.Listado[*Registro], error) {
	var (
		conds []string
		args  []any
	)
	if f.PadreID != nil {
		if n.Padre == nil {
			return db.Listado[*Registro]{}, fmt.Errorf("filtro por padre en %s", n.Tabla)
		}
		args = append(args, *f.PadreID)
		conds = append(conds, fmt.Sprintf("t.%s = $%d", n.CampoPadre, len(args)))
	}
	if f.Q != "" {
		args = append(args, db.PatronContiene(f.Q))
		conds = append(conds, fmt.Sprintf("(t.codigo ILIKE $%d OR t.nombre ILIKE $%d)", len(args), len(args)))
	}
	if f.Activo != nil {
		args = append(args, *f.Activo)
		conds = append(conds, fmt.Sprintf("t.activo = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s t%s`, n.Tabla, where), args...).Scan(&total); err != nil {
		return db.Listado[*Registro]{}, fmt.Errorf("contar %s: %w", n.Plural, err)
	}
	args = append(args, f.Pagina.Tamano, f.Pagina.Desplazamiento())
	filas, err := s.pool.Query(ctx, fmt.Sprintf(`%s%s ORDER BY t.codigo, t.id LIMIT $%d OFFSET $%d`,
		n.selectRegistro(), where, len(args)-1, len(args)), args...)
	if err != nil {
		return db.Listado[*Registro]{}, fmt.Errorf("listar %s: %w", n.Plural, err)
	}
	defer filas.Close()
	var items []*Registro
	for filas.Next() {
		r, err := escanear(n, filas)
		if err != nil {
			return db.Listado[*Registro]{}, fmt.Errorf("leer %s: %w", n.Plural, err)
		}
		items = append(items, r)
	}
	if err := filas.Err(); err != nil {
		return db.Listado[*Registro]{}, fmt.Errorf("listar %s: %w", n.Plural, err)
	}
	return db.NuevoListado(items, total, f.Pagina), nil
}

// Obtener devuelve la ficha: el registro y la ruta de ancestros, de la empresa al padre.
func (s *Servicio) Obtener(ctx context.Context, n *Nivel, id int64) (*Registro, error) {
	r, err := leer(ctx, s.pool, n, id)
	if err != nil {
		return nil, err
	}
	if r.Ancestros, err = ancestros(ctx, s.pool, n, id); err != nil {
		return nil, err
	}
	return r, nil
}

// ancestros lee toda la cadena de ancestros con una sola consulta.
func ancestros(ctx context.Context, c consultor, n *Nivel, id int64) ([]Ancestro, error) {
	niveles := n.ancestros()
	if len(niveles) == 0 {
		return []Ancestro{}, nil
	}
	var (
		cols  []string
		joins []string
	)
	previo, alias := n, "t"
	for i, a := range niveles {
		al := fmt.Sprintf("a%d", i)
		joins = append(joins, fmt.Sprintf("JOIN %s %s ON %s.id = %s.%s", a.Tabla, al, al, alias, previo.CampoPadre))
		cols = append(cols, fmt.Sprintf("%s.id, %s.codigo, %s.nombre, %s.activo", al, al, al, al))
		previo, alias = a, al
	}
	out := make([]Ancestro, len(niveles))
	destinos := make([]any, 0, 4*len(niveles))
	for i, a := range niveles {
		// Se guardan de la empresa hacia abajo.
		x := &out[len(niveles)-1-i]
		x.Entidad = a.Tabla
		destinos = append(destinos, &x.ID, &x.Codigo, &x.Nombre, &x.Activo)
	}
	sql := fmt.Sprintf("SELECT %s FROM %s t %s WHERE t.id = $1", strings.Join(cols, ", "), n.Tabla, strings.Join(joins, " "))
	if err := c.QueryRow(ctx, sql, id).Scan(destinos...); err != nil {
		return nil, fmt.Errorf("leer ancestros de %s %d: %w", n.Tabla, id, err)
	}
	return out, nil
}

// ---------- escritura ----------

func validar(n *Nivel, e Entrada) error {
	var (
		campos   []string
		mensajes []string
	)
	if e.Codigo == "" || utf8.RuneCountInString(e.Codigo) > LargoCodigo {
		campos = append(campos, "codigo")
		mensajes = append(mensajes, fmt.Sprintf("El codigo es obligatorio y admite hasta %d caracteres.", LargoCodigo))
	} else if strings.TrimSpace(e.Codigo) != e.Codigo {
		campos = append(campos, "codigo")
		mensajes = append(mensajes, fmt.Sprintf("El codigo %q tiene espacios al inicio o al final.", e.Codigo))
	} else if strings.ContainsFunc(e.Codigo, unicode.IsControl) {
		campos = append(campos, "codigo")
		mensajes = append(mensajes, "El codigo tiene caracteres de control.")
	}
	if strings.TrimSpace(e.Nombre) == "" || utf8.RuneCountInString(e.Nombre) > LargoNombre {
		campos = append(campos, "nombre")
		mensajes = append(mensajes, fmt.Sprintf("El nombre es obligatorio y admite hasta %d caracteres.", LargoNombre))
	}
	if n.Padre != nil && e.PadreID <= 0 {
		campos = append(campos, n.CampoPadre)
		mensajes = append(mensajes, fmt.Sprintf("Indique %s en %s.", n.Padre.el(), n.CampoPadre))
	}
	if len(campos) > 0 {
		return db.Validacion(strings.Join(mensajes, " "), campos...)
	}
	return nil
}

// bloquearPadre lee el padre con FOR SHARE, para que no lo desactiven mientras se le
// agrega o se le mueve un hijo. Si no existe responde REFERENCIA_INEXISTENTE.
func bloquearPadre(ctx context.Context, tx pgx.Tx, n *Nivel, padreID int64) (ref db.Ref, activo bool, err error) {
	p := n.Padre
	ref.Entidad = p.Tabla
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT id, codigo, nombre, activo FROM %s WHERE id = $1 FOR SHARE`, p.Tabla), padreID).
		Scan(&ref.ID, &ref.Codigo, &ref.Nombre, &activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, false, referenciaInexistente(n, padreID)
	}
	if err != nil {
		return ref, false, fmt.Errorf("leer %s %d: %w", p.Tabla, padreID, err)
	}
	return ref, activo, nil
}

func referenciaInexistente(n *Nivel, padreID int64) error {
	return db.NuevoError(db.ErrReferenciaInexistente,
		fmt.Sprintf("No existe %s con id %d (campo %s).", n.Padre.un(), padreID, n.CampoPadre), n.CampoPadre)
}

func padreInactivo(n *Nivel, padre db.Ref, accion string) error {
	e := db.NuevoError(db.ErrPadreInactivo,
		fmt.Sprintf("%s %s esta %s: no se puede %s %s bajo un padre inactivo.",
			mayuscula(n.Padre.el()), padre.Codigo, n.Padre.inactivo(), accion, n.un()),
		n.CampoPadre)
	e.Detalle = padre
	return e
}

func noEncontrado(n *Nivel, id int64) error {
	return db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe %s con id %d.", n.un(), id))
}

// traducir convierte las violaciones de restricciones en errores con el dato que fallo.
// padre es el padre en el que se busco la unicidad (vacio en la empresa).
func traducir(n *Nivel, err error, codigo string, padre db.Ref) error {
	estado, _ := db.RestriccionViolada(err)
	switch estado {
	case "23505":
		msg := fmt.Sprintf("Ya existe %s con codigo %s", n.un(), codigo)
		if n.Padre != nil {
			msg += fmt.Sprintf(" en %s %s", n.Padre.el(), padre.Codigo)
		}
		return db.NuevoError(db.ErrCodigoDuplicado, msg+". Los codigos dados de baja tampoco se reutilizan.", "codigo")
	case "23503":
		return referenciaInexistente(n, padre.ID)
	}
	return err
}

// Crear da de alta un registro activo.
func (s *Servicio) Crear(ctx context.Context, n *Nivel, e Entrada) (*Registro, error) {
	if err := validar(n, e); err != nil {
		return nil, err
	}
	var r *Registro
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		var (
			padre db.Ref
			id    int64
			err   error
		)
		if n.Padre == nil {
			err = tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO %s (codigo, nombre) VALUES ($1, $2) RETURNING id`, n.Tabla),
				e.Codigo, e.Nombre).Scan(&id)
		} else {
			var activo bool
			if padre, activo, err = bloquearPadre(ctx, tx, n, e.PadreID); err != nil {
				return err
			}
			if !activo {
				return padreInactivo(n, padre, "agregar")
			}
			err = tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO %s (%s, codigo, nombre) VALUES ($1, $2, $3) RETURNING id`,
				n.Tabla, n.CampoPadre), e.PadreID, e.Codigo, e.Nombre).Scan(&id)
		}
		if err != nil {
			return traducir(n, err, e.Codigo, padre)
		}
		r, err = leer(ctx, tx, n, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// bloquear lee el registro con FOR UPDATE. Devuelve el padre actual y si esta activo.
func bloquear(ctx context.Context, tx pgx.Tx, n *Nivel, id int64) (activo bool, padreID int64, codigo string, err error) {
	padreCol := "0::bigint"
	if n.Padre != nil {
		padreCol = n.CampoPadre
	}
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT activo, %s, codigo FROM %s WHERE id = $1 FOR UPDATE`, padreCol, n.Tabla), id).
		Scan(&activo, &padreID, &codigo)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, "", noEncontrado(n, id)
	}
	if err != nil {
		return false, 0, "", fmt.Errorf("bloquear %s %d: %w", n.Tabla, id, err)
	}
	return activo, padreID, codigo, nil
}

// Actualizar cambia codigo, nombre y padre. No toca el estado.
func (s *Servicio) Actualizar(ctx context.Context, n *Nivel, id int64, e Entrada) (*Registro, error) {
	if err := validar(n, e); err != nil {
		return nil, err
	}
	var r *Registro
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		_, padreActual, _, err := bloquear(ctx, tx, n, id)
		if err != nil {
			return err
		}
		var padre db.Ref
		if n.Padre == nil {
			_, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET codigo = $2, nombre = $3, actualizado_en = now() WHERE id = $1`, n.Tabla),
				id, e.Codigo, e.Nombre)
		} else {
			var activo bool
			if padre, activo, err = bloquearPadre(ctx, tx, n, e.PadreID); err != nil {
				return err
			}
			if e.PadreID != padreActual {
				// Mover equivale a colgar de un padre nuevo: tiene que estar activo.
				if !activo {
					return padreInactivo(n, padre, "mover")
				}
				if n == Puesto {
					if err := asignacionesAfectadasPorPuesto(ctx, tx, id); err != nil {
						return err
					}
				}
			}
			_, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET %s = $2, codigo = $3, nombre = $4, actualizado_en = now() WHERE id = $1`,
				n.Tabla, n.CampoPadre), id, e.PadreID, e.Codigo, e.Nombre)
		}
		if err != nil {
			return traducir(n, err, e.Codigo, padre)
		}
		r, err = leer(ctx, tx, n, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// asignacionesAfectadasPorPuesto rechaza mover un puesto a otra seccion si alguno de sus
// usuarios es responsable de servicios: la asignacion quedaria fuera de su seccion.
func asignacionesAfectadasPorPuesto(ctx context.Context, tx pgx.Tx, puestoID int64) error {
	deps, total, err := listarRefs(ctx, tx, "servicio_n2",
		`SELECT sv.id, sv.codigo, sv.nombre FROM servicio_n2 sv JOIN usuario u ON u.id = sv.usuario_responsable_id
		  WHERE u.puesto_id = $1`, puestoID)
	if err != nil {
		return err
	}
	if total == 0 {
		return nil
	}
	e := db.NuevoError(db.ErrAsignacionesAfectadas,
		fmt.Sprintf("No se puede mover el puesto a otra seccion: sus usuarios son responsables de %s. Reasigne o quite el responsable primero.",
			cantidad(total, "servicio", "servicios")), "seccion_id")
	e.Dependencias, e.TotalDependencias = deps, total
	return e
}

// Desactivar hace la baja logica. Si hay dependientes activos no cambia nada y responde
// DEPENDENCIAS_ACTIVAS con la lista. Si ya estaba inactivo, no hace nada.
func (s *Servicio) Desactivar(ctx context.Context, n *Nivel, id int64) (*Registro, error) {
	var r *Registro
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		activo, _, codigo, err := bloquear(ctx, tx, n, id)
		if err != nil {
			return err
		}
		if activo {
			if err := verificarDependencias(ctx, tx, n, id, codigo); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET activo = false, actualizado_en = now() WHERE id = $1`, n.Tabla), id); err != nil {
				return fmt.Errorf("desactivar %s %d: %w", n.Tabla, id, err)
			}
		}
		r, err = leer(ctx, tx, n, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func verificarDependencias(ctx context.Context, tx pgx.Tx, n *Nivel, id int64, codigo string) error {
	var (
		todas  []db.Ref
		total  int
		partes []string
	)
	for _, d := range n.dependientes {
		refs, cuantos, err := listarRefs(ctx, tx, d.entidad, d.consulta, id)
		if err != nil {
			return err
		}
		if cuantos == 0 {
			continue
		}
		total += cuantos
		todas = append(todas, refs...)
		partes = append(partes, cantidad(cuantos, d.singular, d.plural))
	}
	if total == 0 {
		return nil
	}
	if len(todas) > db.MaxDependencias {
		todas = todas[:db.MaxDependencias]
	}
	e := db.NuevoError(db.ErrDependenciasActivas,
		fmt.Sprintf("No se puede desactivar %s %s: tiene %s. Desactivelos primero, de abajo hacia arriba.",
			n.el(), codigo, strings.Join(partes, " y ")))
	e.Dependencias, e.TotalDependencias = todas, total
	return e
}

// listarRefs corre una consulta que devuelve id, codigo y nombre, y trae como mucho
// MaxDependencias filas junto con el total.
func listarRefs(ctx context.Context, c consultor, entidad, consulta string, id int64) ([]db.Ref, int, error) {
	filas, err := c.Query(ctx, fmt.Sprintf(`SELECT id, codigo, nombre, count(*) OVER () FROM (%s) d ORDER BY codigo, id LIMIT %d`,
		consulta, db.MaxDependencias), id)
	if err != nil {
		return nil, 0, fmt.Errorf("listar dependientes (%s): %w", entidad, err)
	}
	defer filas.Close()
	var (
		refs  []db.Ref
		total int
	)
	for filas.Next() {
		ref := db.Ref{Entidad: entidad}
		if err := filas.Scan(&ref.ID, &ref.Codigo, &ref.Nombre, &total); err != nil {
			return nil, 0, fmt.Errorf("leer dependientes (%s): %w", entidad, err)
		}
		refs = append(refs, ref)
	}
	if err := filas.Err(); err != nil {
		return nil, 0, fmt.Errorf("listar dependientes (%s): %w", entidad, err)
	}
	return refs, total, nil
}

// Activar reactiva un registro. Exige que el padre este activo y no reactiva hijos.
func (s *Servicio) Activar(ctx context.Context, n *Nivel, id int64) (*Registro, error) {
	var r *Registro
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		activo, padreID, _, err := bloquear(ctx, tx, n, id)
		if err != nil {
			return err
		}
		if !activo {
			if n.Padre != nil {
				padre, padreActivo, err := bloquearPadre(ctx, tx, n, padreID)
				if err != nil {
					return err
				}
				if !padreActivo {
					return padreInactivo(n, padre, "reactivar")
				}
			}
			if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET activo = true, actualizado_en = now() WHERE id = $1`, n.Tabla), id); err != nil {
				return fmt.Errorf("activar %s %d: %w", n.Tabla, id, err)
			}
		}
		r, err = leer(ctx, tx, n, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// cantidad escribe "1 area activa" o "3 areas activas".
func cantidad(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func mayuscula(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
