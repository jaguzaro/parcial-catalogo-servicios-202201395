// Package importador lleva data/CatalogoServicios.xlsx a la base. Las reglas estan en
// docs/diseno/mapeo-excel.md y las decisiones en docs/contexto/decisiones.md (D08 a D13,
// D20, D21 y D25).
//
// El archivo se abre solo para lectura y nunca se escribe. Todo su contenido es dato.
package importador

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrEnCurso indica que otra corrida tiene el candado (IMPORTACION_EN_CURSO).
var ErrEnCurso = errors.New("IMPORTACION_EN_CURSO: hay otra importacion en curso")

// llaveCandado es la llave fija de pg_try_advisory_xact_lock: una corrida a la vez.
const llaveCandado int64 = 20220139507

// Conexion es lo que el importador necesita de la base: un *pgxpool.Pool, o una pgx.Tx en
// las pruebas, para revertir todo al final.
type Conexion interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Contador es el resultado por entidad.
type Contador struct {
	Creados, Actualizados, SinCambios, Rechazados int
}

// Resumen es lo que queda en la fila de importacion y lo que se imprime al terminar.
type Resumen struct {
	ImportacionID int64
	Archivo       string
	SHA256        string
	Creados       int
	Actualizados  int
	Omitidos      int
	Observados    int
	Entidades     map[string]*Contador
	FilasNoImport int
	Incidencias   map[string]int
	Controles     map[string]int
}

// entidades en el orden en que se escriben y se informan.
var entidades = []string{"clase_servicio", "criticidad", "tipo_servicio", "servicio_n1", "servicio_n2"}

// TiposIncidencia es la lista cerrada del CHECK de incidencia.tipo.
var TiposIncidencia = []string{
	"N1_HEREDADO_DE_FILA_ANTERIOR", "N1_NOMBRE_CONFLICTO", "CODIGO_FORMATO_NO_ESTANDAR",
	"ATRIBUTOS_AUSENTES", "FILA_SIN_CODIGO", "VALOR_FUERA_DE_DOMINIO", "VALOR_FUERA_DE_CATALOGO",
	"VALOR_NUMERICO_INVALIDO", "MINIMO_MAYOR_QUE_MAXIMO", "CONFLICTO_ATRIBUTO",
	"PREFIJO_INCOHERENTE", "CODIGO_N2_REPETIDO", "PADRE_INACTIVO",
	"CODIGO_EXISTENTE_NO_IMPORTADO", "EDICION_LOCAL_SOBRESCRITA", "AUSENTE_EN_ARCHIVO",
}

// Importar corre una importacion completa desde ruta. La fila de importacion se crea como
// en_curso en su propia transaccion; todo lo demas va en una sola: o queda completa o
// queda fallida, nunca a medias.
func Importar(ctx context.Context, conn Conexion, ruta string) (*Resumen, error) {
	datos, err := leerArchivo(ruta)
	if err != nil {
		return nil, err
	}
	suma := sha256.Sum256(datos)
	r := &Resumen{Archivo: ruta, SHA256: hex.EncodeToString(suma[:])}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO importacion (origen, estado, archivo_ruta, archivo_sha256, hoja)
		 VALUES ('cli', 'en_curso', $1, $2, $3) RETURNING id`, ruta, r.SHA256, Hoja).Scan(&r.ImportacionID)
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("registrar la importacion: %w", err)
	}

	if err := ejecutar(ctx, conn, datos, r); err != nil {
		if _, e := conn.Exec(ctx,
			`UPDATE importacion SET estado = 'fallida', mensaje_error = $2, finalizada_en = now() WHERE id = $1`,
			r.ImportacionID, err.Error()); e != nil {
			return nil, fmt.Errorf("%w (y no se pudo marcar la corrida como fallida: %v)", err, e)
		}
		return nil, err
	}
	return r, nil
}

// leerArchivo abre el Excel en solo lectura.
func leerArchivo(ruta string) ([]byte, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, fmt.Errorf("abrir el Excel: %w", err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

func ejecutar(ctx context.Context, conn Conexion, datos []byte, r *Resumen) error {
	plan, err := Analizar(bytes.NewReader(datos))
	if err != nil {
		return err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tengo bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, llaveCandado).Scan(&tengo); err != nil {
		return err
	}
	if !tengo {
		return ErrEnCurso
	}

	a := &aplicador{
		ctx: ctx, tx: tx, id: r.ImportacionID, r: r,
		incidencias: plan.Incidencias,
		n1:          map[string]registro{}, n2: map[string]registro{},
		catalogos: map[string]map[string]int64{},
	}
	r.Entidades = map[string]*Contador{}
	for _, e := range entidades {
		r.Entidades[e] = &Contador{}
	}
	r.Controles = plan.Controles
	r.FilasNoImport = plan.FilasNoImportables
	r.Entidades["servicio_n2"].Rechazados += plan.RepetidosN2

	for _, c := range []struct {
		tabla string
		ops   []Opcion
	}{{"clase_servicio", plan.Clases}, {"criticidad", plan.Criticidades}, {"tipo_servicio", plan.Tipos}} {
		if err := a.catalogo(c.tabla, c.ops); err != nil {
			return fmt.Errorf("%s: %w", c.tabla, err)
		}
	}
	for _, s := range plan.N1 {
		if err := a.servicioN1(s); err != nil {
			return fmt.Errorf("servicio_n1 %s: %w", s.Codigo, err)
		}
	}
	for _, s := range plan.N2 {
		if err := a.servicioN2(s); err != nil {
			return fmt.Errorf("servicio_n2 %s: %w", s.Codigo, err)
		}
	}
	if err := a.ausentes(plan); err != nil {
		return err
	}
	if err := a.registrarIncidencias(); err != nil {
		return fmt.Errorf("incidencias: %w", err)
	}

	for _, c := range r.Entidades {
		r.Creados += c.Creados
		r.Actualizados += c.Actualizados
		r.Omitidos += c.SinCambios + c.Rechazados
	}
	r.Omitidos += r.FilasNoImport

	detalle, err := json.Marshal(r.detalle())
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE importacion SET estado = 'completada', creados = $2, actualizados = $3, omitidos = $4,
		        observados = $5, detalle = $6::jsonb, finalizada_en = now()
		  WHERE id = $1`,
		r.ImportacionID, r.Creados, r.Actualizados, r.Omitidos, r.Observados, string(detalle)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Resumen) detalle() map[string]any {
	d := map[string]any{}
	for _, e := range entidades {
		c := r.Entidades[e]
		d[e] = map[string]any{
			"creados": c.Creados, "actualizados": c.Actualizados,
			"omitidos": map[string]int{"sin_cambios": c.SinCambios, "rechazado": c.Rechazados},
		}
	}
	d["filas"] = map[string]int{"no_importables": r.FilasNoImport}
	d["incidencias"] = r.Incidencias
	d["controles"] = r.Controles
	return d
}

// Escribir imprime el resumen de la corrida.
func (r *Resumen) Escribir(w io.Writer) {
	fmt.Fprintf(w, "importacion %d completada\n", r.ImportacionID)
	fmt.Fprintf(w, "  archivo: %s\n  sha256:  %s\n  hoja:    %s\n", r.Archivo, r.SHA256, Hoja)
	fmt.Fprintf(w, "  creados: %d  actualizados: %d  omitidos: %d  observados: %d\n",
		r.Creados, r.Actualizados, r.Omitidos, r.Observados)
	for _, e := range entidades {
		c := r.Entidades[e]
		fmt.Fprintf(w, "  %-15s creados %3d  actualizados %3d  sin cambios %3d  rechazados %3d\n",
			e, c.Creados, c.Actualizados, c.SinCambios, c.Rechazados)
	}
	fmt.Fprintf(w, "  filas no importables: %d\n", r.FilasNoImport)
	fmt.Fprintf(w, "  controles: nivel 1 distintos %d (en %d celdas), nivel 2 distintos %d (en %d celdas)\n",
		r.Controles["n1_distintos"], r.Controles["celdas_n1"], r.Controles["n2_distintos"], r.Controles["celdas_n2"])
	fmt.Fprintln(w, "  incidencias:")
	total := 0
	for _, t := range TiposIncidencia {
		if n := r.Incidencias[t]; n > 0 {
			fmt.Fprintf(w, "    %-30s %d\n", t, n)
			total += n
		}
	}
	fmt.Fprintf(w, "    %-30s %d\n", "total", total)
}

// registro es lo que se sabe de un servicio en la base durante la corrida.
type registro struct {
	id     int64
	activo bool
}

type aplicador struct {
	ctx         context.Context
	tx          pgx.Tx
	id          int64
	r           *Resumen
	incidencias []Incidencia
	n1          map[string]registro
	n2          map[string]registro
	catalogos   map[string]map[string]int64
}

func (a *aplicador) incidencia(i Incidencia) { a.incidencias = append(a.incidencias, i) }

// catalogo carga una lista de opciones por valor_origen (D12). El nombre no se toca si ya
// existe: lo puede haber cambiado la aplicacion.
func (a *aplicador) catalogo(tabla string, ops []Opcion) error {
	c := a.r.Entidades[tabla]
	ids := map[string]int64{}
	valores := make([]string, 0, len(ops))
	for _, o := range ops {
		valores = append(valores, o.Valor)
		var id int64
		var celda *string
		var orden int
		err := a.tx.QueryRow(a.ctx, `SELECT id, origen_celda, orden FROM `+tabla+` WHERE valor_origen = $1`, o.Valor).
			Scan(&id, &celda, &orden)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			var ocupado bool
			if err := a.tx.QueryRow(a.ctx, `SELECT EXISTS (SELECT 1 FROM `+tabla+` WHERE nombre = $1)`, o.Valor).
				Scan(&ocupado); err != nil {
				return err
			}
			if ocupado {
				c.Rechazados++
				a.incidencia(Incidencia{Tipo: "CODIGO_EXISTENTE_NO_IMPORTADO", Regla: "D13", Celdas: o.Celda, Codigo: o.Valor,
					Original: o.Valor, Mensaje: fmt.Sprintf("%s ya tiene una opcion %q creada desde la aplicacion. No se sobrescribe.", tabla, o.Valor)})
				continue
			}
			if err := a.tx.QueryRow(a.ctx,
				`INSERT INTO `+tabla+` (nombre, valor_origen, origen_celda, orden, importacion_id)
				 VALUES ($1, $1, $2, $3, $4) RETURNING id`, o.Valor, o.Celda, o.Orden, a.id).Scan(&id); err != nil {
				return err
			}
			c.Creados++
		case err != nil:
			return err
		case celda != nil && *celda == o.Celda && orden == o.Orden:
			c.SinCambios++
		default:
			if _, err := a.tx.Exec(a.ctx,
				`UPDATE `+tabla+` SET origen_celda = $2, orden = $3, importacion_id = $4, actualizado_en = now() WHERE id = $1`,
				id, o.Celda, o.Orden, a.id); err != nil {
				return err
			}
			c.Actualizados++
		}
		ids[o.Valor] = id
	}
	a.catalogos[tabla] = ids

	filas, err := a.tx.Query(a.ctx,
		`SELECT valor_origen FROM `+tabla+` WHERE valor_origen IS NOT NULL AND NOT (valor_origen = ANY($1)) ORDER BY id`, valores)
	if err != nil {
		return err
	}
	faltan, err := pgx.CollectRows(filas, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, v := range faltan {
		a.incidencia(Incidencia{Tipo: "AUSENTE_EN_ARCHIVO", Regla: "D13", Codigo: v, Original: v,
			Mensaje: fmt.Sprintf("La opcion %q de %s se importo antes y ya no esta en el archivo. No se borra ni se desactiva.", v, tabla)})
	}
	return nil
}

// campo es una columna de servicio que viene del Excel. Los valores viajan como texto y
// se convierten en el servidor, asi los nulos siguen siendo nulos.
type campo struct {
	clave, columna, tipo string
}

var camposN1 = []campo{{"nombre", "nombre", "text"}}

var camposN2 = []campo{
	{"n1", "servicio_n1_id", "bigint"},
	{"nombre", "nombre", "text"},
	{"activo", "activo", "text"},
	{"clase", "clase_id", "smallint"},
	{"criticidad", "criticidad_id", "smallint"},
	{"tipo", "tipo_id", "smallint"},
	{"descripcion", "descripcion", "text"},
	{"metrica", "metrica", "text"},
	{"minimo", "minimo", "numeric"},
	{"maximo", "maximo", "numeric"},
	{"requiere_revision", "requiere_revision", "boolean"},
}

// origen es lo que se guarda en origen_valores: las celdas leidas, las filas y los valores
// que el Excel aporto despues de aplicar las reglas. De "valores" sale el valor anterior de
// cada campo en la fusion de D25.
type origen struct {
	Celdas  map[string]string `json:"celdas"`
	Filas   []int             `json:"filas"`
	Valores map[string]any    `json:"valores"`
}

// canonico serializa con claves ordenadas y sin espacios, y devuelve tambien su SHA-256.
func canonico(o origen) (string, string, error) {
	b, err := json.Marshal(o) // encoding/json ordena las claves de los mapas
	if err != nil {
		return "", "", err
	}
	s := sha256.Sum256(b)
	return string(b), hex.EncodeToString(s[:]), nil
}

func (s *ServicioN1) origen() origen {
	return origen{Celdas: s.Celdas, Filas: s.Filas, Valores: map[string]any{"nombre": s.Nombre}}
}

func (s *ServicioN2) origen() origen {
	return origen{Celdas: s.Celdas, Filas: s.Filas, Valores: map[string]any{
		"n1": s.N1, "nombre": s.Nombre, "activo": s.Activo,
		"clase": s.Clase, "criticidad": s.Criticidad, "tipo": s.Tipo,
		"descripcion": s.Descripcion, "metrica": s.Metrica,
		"minimo": s.Minimo, "maximo": s.Maximo, "requiere_revision": s.RequiereRevision,
	}}
}

func rango(filas []int, hasta byte) string {
	return fmt.Sprintf("A%d:%c%d", filas[0], hasta, filas[len(filas)-1])
}

func comoTexto(v any) *string {
	var s string
	switch x := v.(type) {
	case nil:
		return nil
	case *string:
		return x
	case string:
		s = x
	case bool:
		s = strconv.FormatBool(x)
	case float64:
		s = strconv.FormatFloat(x, 'f', -1, 64)
	default:
		s = fmt.Sprint(x)
	}
	return &s
}

func idTexto(id int64, ok bool) *string {
	if !ok {
		return nil
	}
	s := strconv.FormatInt(id, 10)
	return &s
}

func iguales(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// valoresBase pasa los valores del Excel a lo que se guarda en la base: codigos de
// catalogo e ids. nuevo indica si son los de esta corrida, para reportar lo que no se
// encuentra.
func (a *aplicador) valoresBase(campos []campo, v map[string]any, codigo string, nuevo bool) (map[string]*string, error) {
	out := map[string]*string{}
	for _, c := range campos {
		t := comoTexto(v[c.clave])
		switch c.clave {
		case "n1":
			if t == nil {
				break
			}
			if reg, ok := a.n1[*t]; ok {
				out[c.clave] = idTexto(reg.id, true)
				continue
			}
			var id int64
			err := a.tx.QueryRow(a.ctx, `SELECT id FROM servicio_n1 WHERE codigo = $1`, *t).Scan(&id)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			out[c.clave] = idTexto(id, err == nil)
			continue
		case "clase", "criticidad", "tipo":
			if t == nil {
				break
			}
			tabla := map[string]string{"clase": "clase_servicio", "criticidad": "criticidad", "tipo": "tipo_servicio"}[c.clave]
			id, ok := a.catalogos[tabla][*t]
			if !ok && nuevo {
				a.incidencia(Incidencia{Tipo: "VALOR_FUERA_DE_CATALOGO", Regla: "D12", Codigo: codigo, N2: codigo, Original: *t,
					Mensaje: fmt.Sprintf("%q no tiene opcion importada en %s. Queda nulo.", *t, tabla)})
			}
			out[c.clave] = idTexto(id, ok)
			continue
		}
		out[c.clave] = t
	}
	return out, nil
}

// fusionar decide, campo por campo, que se escribe (D25): si el Excel no cambio el campo
// se deja lo que hay en la base; si cambio, gana el Excel y, si la base tenia una edicion
// propia, queda EDICION_LOCAL_SOBRESCRITA.
func (a *aplicador) fusionar(campos []campo, anterior, nuevo, actual map[string]*string, inc Incidencia) map[string]*string {
	final := map[string]*string{}
	for _, c := range campos {
		k := c.clave
		if iguales(anterior[k], nuevo[k]) {
			final[k] = actual[k]
			continue
		}
		final[k] = nuevo[k]
		if !iguales(actual[k], anterior[k]) {
			i := inc
			i.Tipo, i.Regla = "EDICION_LOCAL_SOBRESCRITA", "D25"
			i.Original = map[string]any{"campo": c.columna, "valor_base": actual[k], "valor_importacion_anterior": anterior[k]}
			i.Aplicado = map[string]any{c.columna: nuevo[k]}
			i.Mensaje = fmt.Sprintf("El Excel cambio %s de %s y la base tenia una edicion propia. Gana el Excel; "+
				"el valor perdido queda en esta incidencia.", c.columna, inc.Codigo)
			a.incidencia(i)
		}
	}
	return final
}

// escribir inserta (id == 0) o actualiza un servicio con los campos dados y su origen.
func (a *aplicador) escribir(tabla string, campos []campo, id int64, codigo string, valores map[string]*string,
	rangoOrigen, origenJSON, hash string) (int64, error) {
	args := []any{}
	marcador := func(v any) string { args = append(args, v); return "$" + strconv.Itoa(len(args)) }
	var cols, vals, sets []string
	for _, c := range campos {
		ph := fmt.Sprintf("CAST(%s::text AS %s)", marcador(valores[c.clave]), c.tipo)
		cols, vals = append(cols, c.columna), append(vals, ph)
		sets = append(sets, c.columna+" = "+ph)
	}
	for _, o := range []struct {
		col, ph string
	}{
		{"origen_hoja", marcador(Hoja)}, {"origen_rango", marcador(rangoOrigen)},
		{"origen_valores", marcador(origenJSON) + "::jsonb"}, {"origen_hash", marcador(hash)},
		{"importacion_id", marcador(a.id)},
	} {
		cols, vals = append(cols, o.col), append(vals, o.ph)
		sets = append(sets, o.col+" = "+o.ph)
	}
	if id == 0 {
		sql := fmt.Sprintf(`INSERT INTO %s (codigo, %s) VALUES (%s, %s) RETURNING id`,
			tabla, strings.Join(cols, ", "), marcador(codigo), strings.Join(vals, ", "))
		err := a.tx.QueryRow(a.ctx, sql, args...).Scan(&id)
		return id, err
	}
	sql := fmt.Sprintf(`UPDATE %s SET %s, actualizado_en = now() WHERE id = %s`, tabla, strings.Join(sets, ", "), marcador(id))
	_, err := a.tx.Exec(a.ctx, sql, args...)
	return id, err
}

// servicio aplica la llave natural, la huella y la fusion a un servicio de nivel 1 o 2.
func (a *aplicador) servicio(tabla string, campos []campo, codigo string, o origen, rangoOrigen string,
	inc Incidencia, padreInactivo func() bool) (registro, error) {
	c := a.r.Entidades[tabla]
	origenJSON, hash, err := canonico(o)
	if err != nil {
		return registro{}, err
	}
	nuevo, err := a.valoresBase(campos, o.Valores, codigo, true)
	if err != nil {
		return registro{}, err
	}

	columnas := make([]string, len(campos))
	for i, cp := range campos {
		columnas[i] = cp.columna + "::text"
	}
	actualTxt := make([]*string, len(campos))
	destinos := []any{new(int64), new(*string), new([]byte), new(bool)}
	for i := range actualTxt {
		destinos = append(destinos, &actualTxt[i])
	}
	activoCol := "activo"
	if tabla == "servicio_n2" {
		activoCol = "activo <> 'N'"
	}
	err = a.tx.QueryRow(a.ctx, fmt.Sprintf(`SELECT id, origen_hash, origen_valores, %s, %s FROM %s WHERE codigo = $1`,
		activoCol, strings.Join(columnas, ", "), tabla), codigo).Scan(destinos...)
	id := *destinos[0].(*int64)
	hashActual := *destinos[1].(**string)
	origenActual := *destinos[2].(*[]byte)
	activo := *destinos[3].(*bool)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if padreInactivo != nil && padreInactivo() {
			c.Rechazados++
			i := inc
			i.Tipo, i.Regla = "PADRE_INACTIVO", "D13"
			i.Original = map[string]any{"servicio_n1": o.Valores["n1"]}
			i.Mensaje = fmt.Sprintf("%s es nuevo y su nivel 1 %v esta dado de baja. No se importa.", codigo, o.Valores["n1"])
			a.incidencia(i)
			return registro{}, nil
		}
		id, err = a.escribir(tabla, campos, 0, codigo, nuevo, rangoOrigen, origenJSON, hash)
		if err != nil {
			return registro{}, err
		}
		c.Creados++
		return registro{id: id, activo: true}, nil
	case err != nil:
		return registro{}, err
	case hashActual == nil:
		c.Rechazados++
		i := inc
		i.Tipo, i.Regla = "CODIGO_EXISTENTE_NO_IMPORTADO", "D13"
		i.Original = codigo
		i.Mensaje = fmt.Sprintf("%s ya existe y se creo desde la aplicacion. No se sobrescribe.", codigo)
		a.incidencia(i)
		return registro{id: id, activo: activo}, nil
	case *hashActual == hash:
		c.SinCambios++
		return registro{id: id, activo: activo}, nil
	}

	var previo origen
	if err := json.Unmarshal(origenActual, &previo); err != nil {
		return registro{}, fmt.Errorf("origen_valores ilegible: %w", err)
	}
	anterior, err := a.valoresBase(campos, previo.Valores, codigo, false)
	if err != nil {
		return registro{}, err
	}
	actual := map[string]*string{}
	for i, cp := range campos {
		actual[cp.clave] = actualTxt[i]
	}
	final := a.fusionar(campos, anterior, nuevo, actual, inc)
	if _, err := a.escribir(tabla, campos, id, codigo, final, rangoOrigen, origenJSON, hash); err != nil {
		return registro{}, err
	}
	c.Actualizados++
	return registro{id: id, activo: activo}, nil
}

func (a *aplicador) servicioN1(s *ServicioN1) error {
	reg, err := a.servicio("servicio_n1", camposN1, s.Codigo, s.origen(), rango(s.Filas, 'B'),
		Incidencia{Fila: s.Filas[0], Celdas: rango(s.Filas, 'B'), Codigo: s.Codigo, N1: s.Codigo}, nil)
	if err != nil {
		return err
	}
	a.n1[s.Codigo] = reg
	return nil
}

func (a *aplicador) servicioN2(s *ServicioN2) error {
	padreInactivo := func() bool {
		reg, ok := a.n1[s.N1]
		return ok && !reg.activo
	}
	reg, err := a.servicio("servicio_n2", camposN2, s.Codigo, s.origen(), rango(s.Filas, 'L'),
		Incidencia{Fila: s.Filas[0], Celdas: rango(s.Filas, 'L'), Codigo: s.Codigo, N1: s.N1, N2: s.Codigo}, padreInactivo)
	if err != nil {
		return err
	}
	if reg.id != 0 {
		a.n2[s.Codigo] = reg
	}
	return nil
}

// ausentes deja AUSENTE_EN_ARCHIVO por cada servicio importado antes que ya no esta en el
// archivo. No se borra ni se desactiva.
func (a *aplicador) ausentes(p *Plan) error {
	for _, t := range []struct {
		tabla   string
		codigos []string
		n1      bool
	}{{"servicio_n1", codigosN1(p), true}, {"servicio_n2", codigosN2(p), false}} {
		filas, err := a.tx.Query(a.ctx, `SELECT id, codigo FROM `+t.tabla+
			` WHERE origen_hash IS NOT NULL AND NOT (codigo = ANY($1)) ORDER BY codigo`, t.codigos)
		if err != nil {
			return err
		}
		type fila struct {
			ID     int64
			Codigo string
		}
		faltan, err := pgx.CollectRows(filas, pgx.RowToStructByPos[fila])
		if err != nil {
			return err
		}
		for _, f := range faltan {
			i := Incidencia{Tipo: "AUSENTE_EN_ARCHIVO", Regla: "D13", Codigo: f.Codigo, Original: f.Codigo,
				Mensaje: fmt.Sprintf("%s se importo antes y ya no esta en el archivo. No se borra ni se desactiva.", f.Codigo)}
			if t.n1 {
				i.N1 = f.Codigo
				a.n1[f.Codigo] = registro{id: f.ID}
			} else {
				i.N2 = f.Codigo
				a.n2[f.Codigo] = registro{id: f.ID}
			}
			a.incidencia(i)
		}
	}
	return nil
}

func codigosN1(p *Plan) []string {
	s := make([]string, len(p.N1))
	for i, x := range p.N1 {
		s[i] = x.Codigo
	}
	return s
}

func codigosN2(p *Plan) []string {
	s := make([]string, len(p.N2))
	for i, x := range p.N2 {
		s[i] = x.Codigo
	}
	return s
}

func jsonONulo(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// registrarIncidencias escribe todas las incidencias de la corrida y cuenta los registros
// o filas distintos observados.
func (a *aplicador) registrarIncidencias() error {
	a.r.Incidencias = map[string]int{}
	for _, t := range TiposIncidencia {
		a.r.Incidencias[t] = 0
	}
	observados := map[string]bool{}
	sort.SliceStable(a.incidencias, func(i, j int) bool { return a.incidencias[i].Fila < a.incidencias[j].Fila })
	for _, i := range a.incidencias {
		a.r.Incidencias[i.Tipo]++
		var n1, n2 any
		if reg, ok := a.n1[i.N1]; ok && i.N1 != "" && reg.id != 0 {
			n1 = reg.id
		}
		if reg, ok := a.n2[i.N2]; ok && i.N2 != "" && reg.id != 0 {
			n2 = reg.id
		}
		switch {
		case i.N2 != "":
			observados["n2:"+i.N2] = true
		case i.N1 != "":
			observados["n1:"+i.N1] = true
		case i.Fila > 0:
			observados["fila:"+strconv.Itoa(i.Fila)] = true
		default:
			observados["otro:"+i.Codigo] = true
		}
		orig, err := jsonONulo(i.Original)
		if err != nil {
			return err
		}
		apl, err := jsonONulo(i.Aplicado)
		if err != nil {
			return err
		}
		var fila, celdas, codigo any
		if i.Fila > 0 {
			fila = i.Fila
		}
		if i.Celdas != "" {
			celdas = i.Celdas
		}
		if i.Codigo != "" {
			codigo = i.Codigo
		}
		if _, err := a.tx.Exec(a.ctx,
			`INSERT INTO incidencia (importacion_id, tipo, regla, hoja, fila, celdas, codigo, servicio_n1_id, servicio_n2_id,
			                         valor_original, valor_aplicado, mensaje)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12)`,
			a.id, i.Tipo, i.Regla, Hoja, fila, celdas, codigo, n1, n2, orig, apl, i.Mensaje); err != nil {
			return fmt.Errorf("%s fila %d: %w", i.Tipo, i.Fila, err)
		}
	}
	a.r.Observados = len(observados)
	return nil
}
