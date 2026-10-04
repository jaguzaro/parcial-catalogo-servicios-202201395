package catalogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/db"
)

// Estados de servicio_n2.activo (D22). DESCONOCIDO bloquea las bajas igual que S.
const (
	ActivoS           = "S"
	ActivoN           = "N"
	ActivoDesconocido = "DESCONOCIDO"
)

func activoValido(v string) bool { return v == ActivoS || v == ActivoN || v == ActivoDesconocido }

// OpcionRef es la opcion de un catalogo dentro de un servicio. Activo dice si sigue
// disponible en los formularios.
type OpcionRef struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

// SeccionResponsable es la seccion a cargo del servicio, con su jerarquia.
type SeccionResponsable struct {
	ID           int64  `json:"id"`
	Codigo       string `json:"codigo"`
	Nombre       string `json:"nombre"`
	Activo       bool   `json:"activo"`
	Departamento Ref    `json:"departamento"`
	Area         Ref    `json:"area"`
	Empresa      Ref    `json:"empresa"`
}

// UsuarioResponsable es el usuario a cargo. Se muestra aunque este inactivo (D23).
type UsuarioResponsable struct {
	ID      int64  `json:"id"`
	Nombre  string `json:"nombre"`
	Usuario string `json:"usuario"`
	Activo  bool   `json:"activo"`
}

// ServicioN2 es un servicio de nivel 2 con todos sus campos. Minimo y maximo salen como
// numero o null: null es "sin dato", nunca cero.
type ServicioN2 struct {
	ID                   int64               `json:"id"`
	Codigo               string              `json:"codigo"`
	Nombre               string              `json:"nombre"`
	Activo               string              `json:"activo"`
	ServicioN1ID         int64               `json:"servicio_n1_id"`
	ServicioN1           Ref                 `json:"servicio_n1"`
	ClaseID              *int64              `json:"clase_id"`
	Clase                *OpcionRef          `json:"clase"`
	CriticidadID         *int64              `json:"criticidad_id"`
	Criticidad           *OpcionRef          `json:"criticidad"`
	TipoID               *int64              `json:"tipo_id"`
	Tipo                 *OpcionRef          `json:"tipo"`
	Descripcion          *string             `json:"descripcion"`
	Metrica              *string             `json:"metrica"`
	Minimo               *json.Number        `json:"minimo"`
	Maximo               *json.Number        `json:"maximo"`
	RequiereRevision     bool                `json:"requiere_revision"`
	SeccionResponsableID *int64              `json:"seccion_responsable_id"`
	SeccionResponsable   *SeccionResponsable `json:"seccion_responsable"`
	UsuarioResponsableID *int64              `json:"usuario_responsable_id"`
	UsuarioResponsable   *UsuarioResponsable `json:"usuario_responsable"`
	Origen               *Origen             `json:"origen"`
	ImportacionID        *int64              `json:"importacion_id"`
	CreadoEn             time.Time           `json:"creado_en"`
	ActualizadoEn        time.Time           `json:"actualizado_en"`
	// Solo en la ficha.
	Incidencias *[]Incidencia `json:"incidencias,omitempty"`
}

// EntradaN2 es el cuerpo de alta y de modificacion. En la modificacion, un campo ausente
// conserva lo guardado y null lo borra.
type EntradaN2 struct {
	ServicioN1ID     Valor
	Codigo           Valor
	Nombre           Valor
	Activo           Valor
	ClaseID          Valor
	CriticidadID     Valor
	TipoID           Valor
	Descripcion      Valor
	Metrica          Valor
	Minimo           Valor
	Maximo           Valor
	RequiereRevision Valor
}

// FiltroN2 es el filtro del listado. Los filtros se combinan con Y.
type FiltroN2 struct {
	Q                string
	N1ID             *int64
	Activo           string
	ClaseID          *int64
	CriticidadID     *int64
	TipoID           *int64
	SeccionID        *int64
	RequiereRevision *bool
	Pagina           db.Pagina
}

const selectN2 = `
SELECT s.id, s.codigo, s.nombre, s.activo, s.servicio_n1_id, n1.codigo, n1.nombre, n1.activo,
       s.clase_id, c.nombre, c.activo, s.criticidad_id, k.nombre, k.activo, s.tipo_id, t.nombre, t.activo,
       s.descripcion, s.metrica, s.minimo::text, s.maximo::text, s.requiere_revision,
       s.seccion_responsable_id, sc.codigo, sc.nombre, sc.activo,
       d.id, d.codigo, d.nombre, a.id, a.codigo, a.nombre, e.id, e.codigo, e.nombre,
       s.usuario_responsable_id, u.nombre, u.usuario, u.activo,
       s.origen_hoja, s.origen_rango, s.origen_valores::text, s.importacion_id, s.creado_en, s.actualizado_en
  FROM servicio_n2 s
  JOIN servicio_n1 n1             ON n1.id = s.servicio_n1_id
  LEFT JOIN clase_servicio c      ON c.id = s.clase_id
  LEFT JOIN criticidad k          ON k.id = s.criticidad_id
  LEFT JOIN tipo_servicio t       ON t.id = s.tipo_id
  LEFT JOIN seccion sc            ON sc.id = s.seccion_responsable_id
  LEFT JOIN departamento d        ON d.id = sc.departamento_id
  LEFT JOIN area a                ON a.id = d.area_id
  LEFT JOIN empresa e             ON e.id = a.empresa_id
  LEFT JOIN usuario u             ON u.id = s.usuario_responsable_id`

func escanearN2(fila pgx.Row) (*ServicioN2, error) {
	var (
		s                       ServicioN2
		n1Activo                bool
		cNom, kNom, tNom        *string
		cAct, kAct, tAct        *bool
		minimo, maximo          *string
		scCod, scNom            *string
		scAct                   *bool
		dID, aID, eID           *int64
		dCod, dNom, aCod, aNom  *string
		eCod, eNom              *string
		uNom, uUsu              *string
		uAct                    *bool
		hoja, rango, valoresTxt *string
	)
	if err := fila.Scan(&s.ID, &s.Codigo, &s.Nombre, &s.Activo, &s.ServicioN1ID, &s.ServicioN1.Codigo, &s.ServicioN1.Nombre, &n1Activo,
		&s.ClaseID, &cNom, &cAct, &s.CriticidadID, &kNom, &kAct, &s.TipoID, &tNom, &tAct,
		&s.Descripcion, &s.Metrica, &minimo, &maximo, &s.RequiereRevision,
		&s.SeccionResponsableID, &scCod, &scNom, &scAct,
		&dID, &dCod, &dNom, &aID, &aCod, &aNom, &eID, &eCod, &eNom,
		&s.UsuarioResponsableID, &uNom, &uUsu, &uAct,
		&hoja, &rango, &valoresTxt, &s.ImportacionID, &s.CreadoEn, &s.ActualizadoEn); err != nil {
		return nil, err
	}
	s.ServicioN1.ID, s.ServicioN1.Activo = s.ServicioN1ID, &n1Activo
	opcion := func(id *int64, nombre *string, activo *bool) *OpcionRef {
		if id == nil {
			return nil
		}
		return &OpcionRef{ID: *id, Nombre: *nombre, Activo: *activo}
	}
	s.Clase, s.Criticidad, s.Tipo = opcion(s.ClaseID, cNom, cAct), opcion(s.CriticidadID, kNom, kAct), opcion(s.TipoID, tNom, tAct)
	numero := func(v *string) *json.Number {
		if v == nil {
			return nil
		}
		n := json.Number(*v)
		return &n
	}
	s.Minimo, s.Maximo = numero(minimo), numero(maximo)
	if s.SeccionResponsableID != nil {
		s.SeccionResponsable = &SeccionResponsable{
			ID: *s.SeccionResponsableID, Codigo: *scCod, Nombre: *scNom, Activo: *scAct,
			Departamento: Ref{ID: *dID, Codigo: *dCod, Nombre: *dNom},
			Area:         Ref{ID: *aID, Codigo: *aCod, Nombre: *aNom},
			Empresa:      Ref{ID: *eID, Codigo: *eCod, Nombre: *eNom},
		}
	}
	if s.UsuarioResponsableID != nil {
		s.UsuarioResponsable = &UsuarioResponsable{ID: *s.UsuarioResponsableID, Nombre: *uNom, Usuario: *uUsu, Activo: *uAct}
	}
	s.Origen = origen(hoja, rango, valoresTxt)
	return &s, nil
}

func noEncontradoN2(id int64) error {
	return db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe un servicio de nivel 2 con id %d.", id))
}

func leerN2(ctx context.Context, c consultor, id int64) (*ServicioN2, error) {
	s, err := escanearN2(c.QueryRow(ctx, selectN2+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noEncontradoN2(id)
	}
	if err != nil {
		return nil, fmt.Errorf("leer servicio_n2 %d: %w", id, err)
	}
	return s, nil
}

// ListarN2 busca por codigo o nombre y filtra. Ordena por codigo.
func (s *Servicio) ListarN2(ctx context.Context, f FiltroN2) (db.Listado[*ServicioN2], error) {
	if f.Activo != "" && !activoValido(f.Activo) {
		return db.Listado[*ServicioN2]{}, db.Validacion(
			fmt.Sprintf("activo debe ser S, N o DESCONOCIDO; llego %q.", f.Activo), "activo")
	}
	var (
		conds []string
		args  []any
	)
	agregar := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "$?", fmt.Sprintf("$%d", len(args))))
	}
	if f.Q != "" {
		agregar(`(s.codigo ILIKE $? OR s.nombre ILIKE $?)`, db.PatronContiene(f.Q))
	}
	for _, c := range []struct {
		col string
		v   *int64
	}{
		{"servicio_n1_id", f.N1ID}, {"clase_id", f.ClaseID}, {"criticidad_id", f.CriticidadID},
		{"tipo_id", f.TipoID}, {"seccion_responsable_id", f.SeccionID},
	} {
		if c.v != nil {
			// Las columnas de catalogo son smallint: un id fuera de rango no encuentra nada.
			agregar("s."+c.col+" = $?::bigint", *c.v)
		}
	}
	if f.Activo != "" {
		agregar(`s.activo = $?`, f.Activo)
	}
	if f.RequiereRevision != nil {
		agregar(`s.requiere_revision = $?`, *f.RequiereRevision)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := s.conn.QueryRow(ctx, `SELECT count(*) FROM servicio_n2 s`+where, args...).Scan(&total); err != nil {
		return db.Listado[*ServicioN2]{}, fmt.Errorf("contar servicios n2: %w", err)
	}
	args = append(args, f.Pagina.Tamano, f.Pagina.Desplazamiento())
	filas, err := s.conn.Query(ctx, fmt.Sprintf(`%s%s ORDER BY s.codigo, s.id LIMIT $%d OFFSET $%d`,
		selectN2, where, len(args)-1, len(args)), args...)
	if err != nil {
		return db.Listado[*ServicioN2]{}, fmt.Errorf("listar servicios n2: %w", err)
	}
	defer filas.Close()
	var items []*ServicioN2
	for filas.Next() {
		sv, err := escanearN2(filas)
		if err != nil {
			return db.Listado[*ServicioN2]{}, fmt.Errorf("leer servicios n2: %w", err)
		}
		items = append(items, sv)
	}
	if err := filas.Err(); err != nil {
		return db.Listado[*ServicioN2]{}, fmt.Errorf("listar servicios n2: %w", err)
	}
	return db.NuevoListado(items, total, f.Pagina), nil
}

// ObtenerN2 devuelve la ficha con sus incidencias.
func (s *Servicio) ObtenerN2(ctx context.Context, id int64) (*ServicioN2, error) {
	return ficha(ctx, s.conn, id)
}

func ficha(ctx context.Context, c consultor, id int64) (*ServicioN2, error) {
	sv, err := leerN2(ctx, c, id)
	if err != nil {
		return nil, err
	}
	inc, err := incidencias(ctx, c, "servicio_n2_id", id)
	if err != nil {
		return nil, err
	}
	sv.Incidencias = &inc
	return sv, nil
}

// ---------- alta y modificacion ----------

// datosN2 es el estado final de un servicio que se va a guardar.
type datosN2 struct {
	n1                    int64
	codigo, nombre        string
	activo                string
	clase, critic, tipo   *int64
	descripcion, metrica  *string
	minimo, maximo        *string
	requiereRevision      bool
	minimoNuevo, maxNuevo bool // si llegaron en esta peticion, para el mensaje
}

// aplicar valida los campos que llegaron y los pone sobre d. En el alta, d trae los
// valores por defecto y los obligatorios tienen que venir.
func (d *datosN2) aplicar(e EntradaN2, alta bool) error {
	var errs errores
	obligatorio := func(v Valor, campo string) bool {
		if alta && !v.Presente {
			errs.falla(campo, fmt.Sprintf("%s es obligatorio.", campo))
			return false
		}
		return v.Presente
	}
	if obligatorio(e.ServicioN1ID, "servicio_n1_id") {
		if id, ok := errs.id(e.ServicioN1ID, "servicio_n1_id"); ok {
			if id == nil {
				errs.falla("servicio_n1_id", "servicio_n1_id es obligatorio: todo servicio de nivel 2 pertenece a un nivel 1.")
			} else {
				d.n1 = *id
			}
		}
	}
	if obligatorio(e.Codigo, "codigo") {
		if v, ok := errs.codigo(e.Codigo); ok {
			d.codigo = v
		}
	}
	if obligatorio(e.Nombre, "nombre") {
		if v, ok := errs.nombre(e.Nombre, "nombre", LargoNombre); ok {
			d.nombre = v
		}
	}
	if e.Activo.Presente {
		var v string
		if e.Activo.nulo() || json.Unmarshal(e.Activo.JSON, &v) != nil || !activoValido(v) {
			errs.falla("activo", fmt.Sprintf("activo debe ser S, N o DESCONOCIDO; llego %s.", e.Activo.JSON))
		} else {
			d.activo = v
		}
	}
	for _, c := range []struct {
		v     Valor
		campo string
		dest  **int64
	}{{e.ClaseID, "clase_id", &d.clase}, {e.CriticidadID, "criticidad_id", &d.critic}, {e.TipoID, "tipo_id", &d.tipo}} {
		if c.v.Presente {
			if id, ok := errs.id(c.v, c.campo); ok {
				*c.dest = id
			}
		}
	}
	for _, c := range []struct {
		v     Valor
		campo string
		dest  **string
	}{{e.Descripcion, "descripcion", &d.descripcion}, {e.Metrica, "metrica", &d.metrica}} {
		if c.v.Presente {
			if s, ok := errs.opcional(c.v, c.campo); ok {
				*c.dest = s
			}
		}
	}
	if e.Minimo.Presente {
		if v, ok := errs.numero(e.Minimo, "minimo"); ok {
			d.minimo, d.minimoNuevo = v, true
		}
	}
	if e.Maximo.Presente {
		if v, ok := errs.numero(e.Maximo, "maximo"); ok {
			d.maximo, d.maxNuevo = v, true
		}
	}
	if e.RequiereRevision.Presente {
		if b, ok := errs.booleano(e.RequiereRevision, "requiere_revision"); ok {
			d.requiereRevision = b
		}
	}
	if err := errs.err(); err != nil {
		return err
	}
	return d.validarRango()
}

// validarRango aplica minimo <= maximo sobre el estado final. Con uno solo, o ninguno, no
// hay nada que comparar.
func (d *datosN2) validarRango() error {
	if d.minimo == nil || d.maximo == nil {
		return nil
	}
	mn, _ := new(big.Rat).SetString(*d.minimo)
	mx, _ := new(big.Rat).SetString(*d.maximo)
	if mn == nil || mx == nil || mn.Cmp(mx) <= 0 {
		return nil
	}
	describir := func(v string, nuevo bool) string {
		if nuevo {
			return v
		}
		return v + ", el valor guardado"
	}
	return db.NuevoError(db.ErrMinimoMayorQueMaximo,
		fmt.Sprintf("El minimo (%s) no puede ser mayor que el maximo (%s). No se guardo nada.",
			describir(*d.minimo, d.minimoNuevo), describir(*d.maximo, d.maxNuevo)), "minimo", "maximo")
}

func traducirN2(err error, codigo string) error {
	estado, restriccion := db.RestriccionViolada(err)
	switch {
	case estado == "23505":
		return codigoDuplicado("nivel 2", codigo)
	case estado == "23514" && restriccion == "servicio_n2_minmax_ck":
		return db.NuevoError(db.ErrMinimoMayorQueMaximo, "El minimo no puede ser mayor que el maximo. No se guardo nada.", "minimo", "maximo")
	}
	return err
}

// verificarReferencias revisa el nivel 1 y las opciones que se fijan o cambian. antes es
// nil en el alta.
func verificarReferencias(ctx context.Context, tx pgx.Tx, d *datosN2, antes *datosN2) error {
	ref, activo, err := verificarN1(ctx, tx, d.n1)
	if err != nil {
		return err
	}
	// Crear, mover o reactivar bajo un nivel 1 inactivo rompe el invariante.
	if !activo {
		cambia := antes == nil || antes.n1 != d.n1 || (antes.activo == ActivoN && d.activo != ActivoN)
		if cambia {
			e := db.NuevoError(db.ErrPadreInactivo,
				fmt.Sprintf("El servicio de nivel 1 %s esta inactivo: no se puede agregar, mover ni reactivar un servicio de nivel 2 bajo el.", ref.Codigo),
				"servicio_n1_id")
			e.Detalle = ref
			return e
		}
	}
	nuevos := []*int64{d.clase, d.critic, d.tipo}
	viejos := make([]*int64, len(nuevos))
	if antes != nil {
		viejos = []*int64{antes.clase, antes.critic, antes.tipo}
	}
	for i, cat := range Catalogos {
		nuevo, viejo := nuevos[i], viejos[i]
		// Si el servicio ya tenia ese valor y no cambia, se conserva aunque este inactivo.
		if nuevo == nil || (viejo != nil && *viejo == *nuevo) {
			continue
		}
		if err := verificarOpcion(ctx, tx, cat, *nuevo); err != nil {
			return err
		}
	}
	return nil
}

// CrearN2 da de alta un servicio de nivel 2. activo es S si no llega.
func (s *Servicio) CrearN2(ctx context.Context, e EntradaN2) (*ServicioN2, error) {
	d := &datosN2{activo: ActivoS}
	if err := d.aplicar(e, true); err != nil {
		return nil, err
	}
	var sv *ServicioN2
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		if err := verificarReferencias(ctx, tx, d, nil); err != nil {
			return err
		}
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO servicio_n2 (servicio_n1_id, codigo, nombre, activo, clase_id, criticidad_id, tipo_id,
			                          descripcion, metrica, minimo, maximo, requiere_revision)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::numeric, $11::numeric, $12) RETURNING id`,
			d.n1, d.codigo, d.nombre, d.activo, d.clase, d.critic, d.tipo, d.descripcion, d.metrica,
			d.minimo, d.maximo, d.requiereRevision).Scan(&id); err != nil {
			return traducirN2(err, d.codigo)
		}
		var err error
		sv, err = ficha(ctx, tx, id)
		return err
	})
	return sv, err
}

// bloquearN2 lee el estado guardado con FOR UPDATE.
func bloquearN2(ctx context.Context, tx pgx.Tx, id int64) (*datosN2, error) {
	var d datosN2
	err := tx.QueryRow(ctx,
		`SELECT servicio_n1_id, codigo, nombre, activo, clase_id, criticidad_id, tipo_id, descripcion, metrica,
		        minimo::text, maximo::text, requiere_revision
		   FROM servicio_n2 WHERE id = $1 FOR UPDATE`, id).
		Scan(&d.n1, &d.codigo, &d.nombre, &d.activo, &d.clase, &d.critic, &d.tipo, &d.descripcion, &d.metrica,
			&d.minimo, &d.maximo, &d.requiereRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noEncontradoN2(id)
	}
	if err != nil {
		return nil, fmt.Errorf("bloquear servicio_n2 %d: %w", id, err)
	}
	return &d, nil
}

// ActualizarN2 cambia los campos que llegan y valida el estado final: si solo llega el
// minimo, se compara con el maximo guardado. No toca el responsable.
func (s *Servicio) ActualizarN2(ctx context.Context, id int64, e EntradaN2) (*ServicioN2, error) {
	var sv *ServicioN2
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		antes, err := bloquearN2(ctx, tx, id)
		if err != nil {
			return err
		}
		d := *antes
		if err := d.aplicar(e, false); err != nil {
			return err
		}
		if err := verificarReferencias(ctx, tx, &d, antes); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE servicio_n2 SET servicio_n1_id = $2, codigo = $3, nombre = $4, activo = $5, clase_id = $6,
			        criticidad_id = $7, tipo_id = $8, descripcion = $9, metrica = $10, minimo = $11::numeric,
			        maximo = $12::numeric, requiere_revision = $13, actualizado_en = now()
			  WHERE id = $1`,
			id, d.n1, d.codigo, d.nombre, d.activo, d.clase, d.critic, d.tipo, d.descripcion, d.metrica,
			d.minimo, d.maximo, d.requiereRevision); err != nil {
			return traducirN2(err, d.codigo)
		}
		sv, err = ficha(ctx, tx, id)
		return err
	})
	return sv, err
}

// DesactivarN2 pone activo = N. Nada lo bloquea y conserva su responsable.
func (s *Servicio) DesactivarN2(ctx context.Context, id int64) (*ServicioN2, error) {
	var sv *ServicioN2
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		d, err := bloquearN2(ctx, tx, id)
		if err != nil {
			return err
		}
		if d.activo != ActivoN {
			if _, err := tx.Exec(ctx, `UPDATE servicio_n2 SET activo = 'N', actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("desactivar servicio_n2 %d: %w", id, err)
			}
		}
		sv, err = ficha(ctx, tx, id)
		return err
	})
	return sv, err
}

// ActivarN2 pone activo = S, que tambien resuelve un DESCONOCIDO. Exige el nivel 1 activo.
func (s *Servicio) ActivarN2(ctx context.Context, id int64) (*ServicioN2, error) {
	var sv *ServicioN2
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		d, err := bloquearN2(ctx, tx, id)
		if err != nil {
			return err
		}
		if d.activo != ActivoS {
			ref, activo, err := verificarN1(ctx, tx, d.n1)
			if err != nil {
				return err
			}
			if !activo {
				e := db.NuevoError(db.ErrPadreInactivo,
					fmt.Sprintf("El servicio de nivel 1 %s esta inactivo: reactivelo antes de reactivar el servicio %s.", ref.Codigo, d.codigo),
					"servicio_n1_id")
				e.Detalle = ref
				return e
			}
			if _, err := tx.Exec(ctx, `UPDATE servicio_n2 SET activo = 'S', actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("activar servicio_n2 %d: %w", id, err)
			}
		}
		sv, err = ficha(ctx, tx, id)
		return err
	})
	return sv, err
}
