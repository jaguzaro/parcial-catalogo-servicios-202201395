// Package usuarios mantiene los usuarios: alta, modificacion, contrasena, baja y
// reactivacion, con la proteccion del ultimo administrador. Las reglas estan en
// docs/diseno/reglas.md.
//
// Ninguna consulta de lectura de este paquete selecciona la columna de la contrasena: el
// tipo Usuario no tiene donde guardarla.
package usuarios

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"catalogo/internal/auth"
	"catalogo/internal/db"
)

// Limites de longitud, en caracteres. Los fija este paquete; el diseno no los da.
const (
	LargoNombre  = 200
	LargoUsuario = 100
	LargoCorreo  = 254
)

// ServicioResumen es un servicio del que el usuario es responsable.
type ServicioResumen struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
	Activo string `json:"activo"`
}

// Usuario es lo que se expone de un usuario.
type Usuario struct {
	ID            int64          `json:"id"`
	Nombre        string         `json:"nombre"`
	Usuario       string         `json:"usuario"`
	Correo        *string        `json:"correo"`
	Rol           string         `json:"rol"`
	Activo        bool           `json:"activo"`
	PuestoID      int64          `json:"puesto_id"`
	Jerarquia     auth.Jerarquia `json:"jerarquia"`
	CreadoEn      time.Time      `json:"creado_en"`
	ActualizadoEn time.Time      `json:"actualizado_en"`
	// Solo en la ficha. En el listado no aparece.
	ServiciosResponsable *[]ServicioResumen `json:"servicios_responsable,omitempty"`
}

// Advertencia acompana una operacion que se hizo pero deja algo que revisar.
type Advertencia struct {
	Codigo    string   `json:"codigo"`
	Mensaje   string   `json:"mensaje"`
	Servicios []db.Ref `json:"servicios"`
}

// Entrada es el cuerpo de alta y de modificacion. Contrasena solo se usa en el alta.
type Entrada struct {
	Nombre     string
	Usuario    string
	Correo     *string
	Contrasena string
	Rol        string
	PuestoID   int64
}

// Filtro de un listado.
type Filtro struct {
	Q         string
	Rol       string
	Activo    *bool
	PuestoID  *int64
	SeccionID *int64
	Pagina    db.Pagina
}

// Servicio aplica las reglas de usuarios.
type Servicio struct {
	pool *pgxpool.Pool
}

// Nuevo crea el servicio.
func Nuevo(pool *pgxpool.Pool) *Servicio { return &Servicio{pool: pool} }

type consultor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ---------- lectura ----------

const selectUsuario = `
SELECT u.id, u.nombre, u.usuario, u.correo, u.rol, u.activo, u.puesto_id, u.creado_en, u.actualizado_en,
       p.id, p.codigo, p.nombre, s.id, s.codigo, s.nombre, d.id, d.codigo, d.nombre,
       a.id, a.codigo, a.nombre, e.id, e.codigo, e.nombre
  FROM usuario u
  JOIN puesto p       ON p.id = u.puesto_id
  JOIN seccion s      ON s.id = p.seccion_id
  JOIN departamento d ON d.id = s.departamento_id
  JOIN area a         ON a.id = d.area_id
  JOIN empresa e      ON e.id = a.empresa_id`

func escanear(fila pgx.Row) (*Usuario, error) {
	var u Usuario
	j := &u.Jerarquia
	err := fila.Scan(&u.ID, &u.Nombre, &u.Usuario, &u.Correo, &u.Rol, &u.Activo, &u.PuestoID, &u.CreadoEn, &u.ActualizadoEn,
		&j.Puesto.ID, &j.Puesto.Codigo, &j.Puesto.Nombre, &j.Seccion.ID, &j.Seccion.Codigo, &j.Seccion.Nombre,
		&j.Departamento.ID, &j.Departamento.Codigo, &j.Departamento.Nombre,
		&j.Area.ID, &j.Area.Codigo, &j.Area.Nombre, &j.Empresa.ID, &j.Empresa.Codigo, &j.Empresa.Nombre)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func leer(ctx context.Context, c consultor, id int64) (*Usuario, error) {
	u, err := escanear(c.QueryRow(ctx, selectUsuario+` WHERE u.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, noEncontrado(id)
	}
	if err != nil {
		return nil, fmt.Errorf("leer usuario %d: %w", id, err)
	}
	return u, nil
}

func noEncontrado(id int64) error {
	return db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("No existe un usuario con id %d.", id))
}

// Listar devuelve una pagina ordenada por nombre de usuario.
func (s *Servicio) Listar(ctx context.Context, f Filtro) (db.Listado[*Usuario], error) {
	if f.Rol != "" && !rolValido(f.Rol) {
		return db.Listado[*Usuario]{}, db.Validacion(
			fmt.Sprintf("El rol %q no existe. Use administrador o consulta.", f.Rol), "rol")
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
		agregar(`(u.nombre ILIKE $? OR u.usuario ILIKE $? OR u.correo ILIKE $?)`, db.PatronContiene(f.Q))
	}
	if f.Rol != "" {
		agregar(`u.rol = $?`, f.Rol)
	}
	if f.Activo != nil {
		agregar(`u.activo = $?`, *f.Activo)
	}
	if f.PuestoID != nil {
		agregar(`u.puesto_id = $?`, *f.PuestoID)
	}
	if f.SeccionID != nil {
		agregar(`p.seccion_id = $?`, *f.SeccionID)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM usuario u JOIN puesto p ON p.id = u.puesto_id`+where, args...).
		Scan(&total); err != nil {
		return db.Listado[*Usuario]{}, fmt.Errorf("contar usuarios: %w", err)
	}
	args = append(args, f.Pagina.Tamano, f.Pagina.Desplazamiento())
	filas, err := s.pool.Query(ctx, fmt.Sprintf(`%s%s ORDER BY u.usuario, u.id LIMIT $%d OFFSET $%d`,
		selectUsuario, where, len(args)-1, len(args)), args...)
	if err != nil {
		return db.Listado[*Usuario]{}, fmt.Errorf("listar usuarios: %w", err)
	}
	defer filas.Close()
	var items []*Usuario
	for filas.Next() {
		u, err := escanear(filas)
		if err != nil {
			return db.Listado[*Usuario]{}, fmt.Errorf("leer usuarios: %w", err)
		}
		items = append(items, u)
	}
	if err := filas.Err(); err != nil {
		return db.Listado[*Usuario]{}, fmt.Errorf("listar usuarios: %w", err)
	}
	return db.NuevoListado(items, total, f.Pagina), nil
}

// Obtener devuelve la ficha: el usuario, su jerarquia y los servicios de los que es
// responsable, activos o no.
func (s *Servicio) Obtener(ctx context.Context, id int64) (*Usuario, error) {
	u, err := leer(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	filas, err := s.pool.Query(ctx,
		`SELECT id, codigo, nombre, activo FROM servicio_n2 WHERE usuario_responsable_id = $1 ORDER BY codigo, id`, id)
	if err != nil {
		return nil, fmt.Errorf("servicios del usuario %d: %w", id, err)
	}
	defer filas.Close()
	servicios := []ServicioResumen{}
	for filas.Next() {
		var sv ServicioResumen
		if err := filas.Scan(&sv.ID, &sv.Codigo, &sv.Nombre, &sv.Activo); err != nil {
			return nil, fmt.Errorf("servicios del usuario %d: %w", id, err)
		}
		servicios = append(servicios, sv)
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("servicios del usuario %d: %w", id, err)
	}
	u.ServiciosResponsable = &servicios
	return u, nil
}

// ---------- escritura ----------

func rolValido(rol string) bool { return rol == auth.RolAdministrador || rol == auth.RolConsulta }

func validar(e Entrada, conContrasena bool) error {
	var (
		campos   []string
		mensajes []string
	)
	falla := func(campo, msg string) {
		campos = append(campos, campo)
		mensajes = append(mensajes, msg)
	}
	if strings.TrimSpace(e.Nombre) == "" || utf8.RuneCountInString(e.Nombre) > LargoNombre {
		falla("nombre", fmt.Sprintf("El nombre es obligatorio y admite hasta %d caracteres.", LargoNombre))
	}
	switch {
	case e.Usuario == "" || utf8.RuneCountInString(e.Usuario) > LargoUsuario:
		falla("usuario", fmt.Sprintf("El usuario es obligatorio y admite hasta %d caracteres.", LargoUsuario))
	case strings.ContainsFunc(e.Usuario, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }):
		falla("usuario", fmt.Sprintf("El usuario %q no puede tener espacios.", e.Usuario))
	case strings.Contains(e.Usuario, "@"):
		falla("usuario", fmt.Sprintf("El usuario %q no puede llevar @: el inicio de sesion lo tomaria por un correo.", e.Usuario))
	}
	if e.Correo != nil {
		c := *e.Correo
		if utf8.RuneCountInString(c) > LargoCorreo || strings.Index(c, "@") < 1 || strings.Index(c, "@") == len(c)-1 ||
			strings.ContainsFunc(c, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
			falla("correo", fmt.Sprintf("El correo %q no es valido. Si no tiene, omita el campo o envie null.", c))
		}
	}
	if conContrasena && auth.ValidarLargo(e.Contrasena) != nil {
		falla("contrasena", fmt.Sprintf("La contrasena debe tener entre %d y %d caracteres.", auth.LargoMinimo, auth.LargoMaximo))
	}
	if !rolValido(e.Rol) {
		falla("rol", fmt.Sprintf("El rol %q no existe. Use administrador o consulta.", e.Rol))
	}
	if e.PuestoID <= 0 {
		falla("puesto_id", "Indique el puesto en puesto_id.")
	}
	if len(campos) > 0 {
		return db.Validacion(strings.Join(mensajes, " "), campos...)
	}
	return nil
}

type puestoInfo struct {
	ref       db.Ref
	activo    bool
	seccionID int64
}

// bloquearPuesto lee el puesto con FOR SHARE, para que no lo desactiven mientras se le
// asigna un usuario.
func bloquearPuesto(ctx context.Context, tx pgx.Tx, puestoID int64) (puestoInfo, error) {
	p := puestoInfo{ref: db.Ref{Entidad: "puesto"}}
	err := tx.QueryRow(ctx, `SELECT id, codigo, nombre, activo, seccion_id FROM puesto WHERE id = $1 FOR SHARE`, puestoID).
		Scan(&p.ref.ID, &p.ref.Codigo, &p.ref.Nombre, &p.activo, &p.seccionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, db.NuevoError(db.ErrReferenciaInexistente,
			fmt.Sprintf("No existe un puesto con id %d (campo puesto_id).", puestoID), "puesto_id")
	}
	if err != nil {
		return p, fmt.Errorf("leer puesto %d: %w", puestoID, err)
	}
	return p, nil
}

func puestoInactivo(p puestoInfo, accion string) error {
	e := db.NuevoError(db.ErrPadreInactivo,
		fmt.Sprintf("El puesto %s esta inactivo: no se puede %s un usuario en un puesto inactivo.", p.ref.Codigo, accion), "puesto_id")
	e.Detalle = p.ref
	return e
}

// traducir convierte una violacion de unicidad en un error que nombra el dato repetido.
func traducir(err error, e Entrada) error {
	estado, restriccion := db.RestriccionViolada(err)
	switch {
	case estado == "23505" && restriccion == "usuario_correo_uk" && e.Correo != nil:
		return db.NuevoError(db.ErrCodigoDuplicado,
			fmt.Sprintf("Ya existe un usuario con el correo %s (sin distinguir mayusculas).", *e.Correo), "correo")
	case estado == "23505":
		return db.NuevoError(db.ErrCodigoDuplicado,
			fmt.Sprintf("Ya existe un usuario %s (sin distinguir mayusculas).", e.Usuario), "usuario")
	case estado == "23503":
		return db.NuevoError(db.ErrReferenciaInexistente,
			fmt.Sprintf("No existe un puesto con id %d (campo puesto_id).", e.PuestoID), "puesto_id")
	}
	return err
}

// Crear da de alta un usuario activo. La contrasena se guarda con argon2id.
func (s *Servicio) Crear(ctx context.Context, e Entrada) (*Usuario, error) {
	if err := validar(e, true); err != nil {
		return nil, err
	}
	// El hash cuesta 64 MiB y algo de CPU: se calcula fuera de la transaccion.
	cifrada, err := auth.HashContrasena(e.Contrasena)
	if err != nil {
		return nil, err
	}
	var u *Usuario
	err = db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		p, err := bloquearPuesto(ctx, tx, e.PuestoID)
		if err != nil {
			return err
		}
		if !p.activo {
			return puestoInactivo(p, "crear")
		}
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO usuario (puesto_id, nombre, usuario, correo, hash_contrasena, rol)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			e.PuestoID, e.Nombre, e.Usuario, e.Correo, cifrada, e.Rol).Scan(&id); err != nil {
			return traducir(err, e)
		}
		u, err = leer(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

type estadoUsuario struct {
	rol       string
	activo    bool
	puestoID  int64
	seccionID int64
	usuario   string
}

// bloquear lee el usuario con FOR UPDATE.
func bloquear(ctx context.Context, tx pgx.Tx, id int64) (estadoUsuario, error) {
	var est estadoUsuario
	err := tx.QueryRow(ctx,
		`SELECT u.rol, u.activo, u.puesto_id, p.seccion_id, u.usuario
		   FROM usuario u JOIN puesto p ON p.id = u.puesto_id
		  WHERE u.id = $1 FOR UPDATE OF u`, id).
		Scan(&est.rol, &est.activo, &est.puestoID, &est.seccionID, &est.usuario)
	if errors.Is(err, pgx.ErrNoRows) {
		return est, noEncontrado(id)
	}
	if err != nil {
		return est, fmt.Errorf("bloquear usuario %d: %w", id, err)
	}
	return est, nil
}

// verificarNoEsUltimoAdministrador bloquea a todos los administradores activos, para que
// dos bajas simultaneas no dejen el sistema sin ninguno, y rechaza si solo queda uno.
func verificarNoEsUltimoAdministrador(ctx context.Context, tx pgx.Tx, usuario, accion string) error {
	filas, err := tx.Query(ctx, `SELECT id FROM usuario WHERE rol = 'administrador' AND activo ORDER BY id FOR UPDATE`)
	if err != nil {
		return fmt.Errorf("bloquear administradores: %w", err)
	}
	n := 0
	for filas.Next() {
		n++
	}
	filas.Close()
	if err := filas.Err(); err != nil {
		return fmt.Errorf("bloquear administradores: %w", err)
	}
	if n <= 1 {
		return db.NuevoError(db.ErrUltimoAdministrador,
			fmt.Sprintf("No se puede %s a %s: es el ultimo administrador activo y nadie podria mantener el sistema.", accion, usuario))
	}
	return nil
}

// Actualizar cambia nombre, usuario, correo, rol y puesto. No toca la contrasena ni el
// estado.
func (s *Servicio) Actualizar(ctx context.Context, id int64, e Entrada) (*Usuario, error) {
	if err := validar(e, false); err != nil {
		return nil, err
	}
	var u *Usuario
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		est, err := bloquear(ctx, tx, id)
		if err != nil {
			return err
		}
		if est.activo && est.rol == auth.RolAdministrador && e.Rol != auth.RolAdministrador {
			if err := verificarNoEsUltimoAdministrador(ctx, tx, est.usuario, "pasar a "+e.Rol); err != nil {
				return err
			}
		}
		if e.PuestoID != est.puestoID {
			p, err := bloquearPuesto(ctx, tx, e.PuestoID)
			if err != nil {
				return err
			}
			if !p.activo {
				return puestoInactivo(p, "mover")
			}
			if p.seccionID != est.seccionID {
				if err := asignacionesAfectadas(ctx, tx, id, est.usuario); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE usuario SET nombre = $2, usuario = $3, correo = $4, rol = $5, puesto_id = $6, actualizado_en = now()
			  WHERE id = $1`, id, e.Nombre, e.Usuario, e.Correo, e.Rol, e.PuestoID); err != nil {
			return traducir(err, e)
		}
		u, err = leer(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// asignacionesAfectadas rechaza cambiar de seccion a un usuario responsable de servicios:
// dejaria de pertenecer a la seccion responsable.
func asignacionesAfectadas(ctx context.Context, tx pgx.Tx, id int64, usuario string) error {
	refs, total, err := serviciosResponsable(ctx, tx, id, false)
	if err != nil {
		return err
	}
	if total == 0 {
		return nil
	}
	e := db.NuevoError(db.ErrAsignacionesAfectadas,
		fmt.Sprintf("No se puede mover a %s a un puesto de otra seccion: es responsable de %d servicio(s). Reasigne o quite el responsable primero.",
			usuario, total), "puesto_id")
	e.Dependencias, e.TotalDependencias = refs, total
	return e
}

// serviciosResponsable lista como mucho MaxDependencias servicios de los que el usuario
// es responsable, y el total. Con soloVigentes deja fuera los que tienen activo = 'N'.
func serviciosResponsable(ctx context.Context, c consultor, id int64, soloVigentes bool) ([]db.Ref, int, error) {
	filtro := ""
	if soloVigentes {
		filtro = ` AND activo <> 'N'`
	}
	filas, err := c.Query(ctx, fmt.Sprintf(
		`SELECT id, codigo, nombre, count(*) OVER () FROM servicio_n2 WHERE usuario_responsable_id = $1%s
		  ORDER BY codigo, id LIMIT %d`, filtro, db.MaxDependencias), id)
	if err != nil {
		return nil, 0, fmt.Errorf("servicios del usuario %d: %w", id, err)
	}
	defer filas.Close()
	var (
		refs  []db.Ref
		total int
	)
	for filas.Next() {
		ref := db.Ref{Entidad: "servicio_n2"}
		if err := filas.Scan(&ref.ID, &ref.Codigo, &ref.Nombre, &total); err != nil {
			return nil, 0, fmt.Errorf("servicios del usuario %d: %w", id, err)
		}
		refs = append(refs, ref)
	}
	if err := filas.Err(); err != nil {
		return nil, 0, fmt.Errorf("servicios del usuario %d: %w", id, err)
	}
	return refs, total, nil
}

// FijarContrasena reemplaza la contrasena y revoca todas las sesiones del usuario.
func (s *Servicio) FijarContrasena(ctx context.Context, id int64, contrasena string) error {
	if auth.ValidarLargo(contrasena) != nil {
		return db.Validacion(fmt.Sprintf("La contrasena debe tener entre %d y %d caracteres.", auth.LargoMinimo, auth.LargoMaximo),
			"contrasena")
	}
	cifrada, err := auth.HashContrasena(contrasena)
	if err != nil {
		return err
	}
	return db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := bloquear(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE usuario SET hash_contrasena = $2, actualizado_en = now() WHERE id = $1`, id, cifrada); err != nil {
			return fmt.Errorf("fijar contrasena del usuario %d: %w", id, err)
		}
		return revocarSesiones(ctx, tx, id)
	})
}

func revocarSesiones(ctx context.Context, tx pgx.Tx, id int64) error {
	if _, err := tx.Exec(ctx, `UPDATE sesion SET revocada_en = now() WHERE usuario_id = $1 AND revocada_en IS NULL`, id); err != nil {
		return fmt.Errorf("revocar sesiones del usuario %d: %w", id, err)
	}
	return nil
}

// Desactivar hace la baja logica y revoca las sesiones en la misma transaccion. Que sea
// responsable de servicios no la impide (D23): se informa en las advertencias y la
// asignacion se conserva.
func (s *Servicio) Desactivar(ctx context.Context, id int64) (*Usuario, []Advertencia, error) {
	var (
		u            *Usuario
		advertencias = []Advertencia{}
	)
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		est, err := bloquear(ctx, tx, id)
		if err != nil {
			return err
		}
		if est.activo {
			if est.rol == auth.RolAdministrador {
				if err := verificarNoEsUltimoAdministrador(ctx, tx, est.usuario, "desactivar"); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `UPDATE usuario SET activo = false, actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("desactivar usuario %d: %w", id, err)
			}
			if err := revocarSesiones(ctx, tx, id); err != nil {
				return err
			}
		}
		refs, total, err := serviciosResponsable(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if total > 0 {
			advertencias = append(advertencias, Advertencia{
				Codigo: "RESPONSABLE_DE_SERVICIOS",
				Mensaje: fmt.Sprintf("%s sigue figurando como responsable de %d servicio(s) vigente(s). La asignacion se conserva; conviene reasignarlos.",
					est.usuario, total),
				Servicios: refs,
			})
		}
		u, err = leer(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return u, advertencias, nil
}

// Activar reactiva al usuario si su puesto esta activo. Las sesiones anteriores siguen
// revocadas: tiene que iniciar sesion de nuevo.
func (s *Servicio) Activar(ctx context.Context, id int64) (*Usuario, error) {
	var u *Usuario
	err := db.EnTx(ctx, s.pool, func(tx pgx.Tx) error {
		est, err := bloquear(ctx, tx, id)
		if err != nil {
			return err
		}
		if !est.activo {
			p, err := bloquearPuesto(ctx, tx, est.puestoID)
			if err != nil {
				return err
			}
			if !p.activo {
				return puestoInactivo(p, "reactivar")
			}
			if _, err := tx.Exec(ctx, `UPDATE usuario SET activo = true, actualizado_en = now() WHERE id = $1`, id); err != nil {
				return fmt.Errorf("activar usuario %d: %w", id, err)
			}
		}
		u, err = leer(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}
