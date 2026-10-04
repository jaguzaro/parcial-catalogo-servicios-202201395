package importador

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/db"
)

// Lector consulta las corridas ya hechas: GET /api/importaciones y GET /api/importaciones/{id}.
// Solo lee; quien escribe es Importar.
type Lector struct {
	conn lectorConn
}

type lectorConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// NuevoLector crea el lector sobre un *pgxpool.Pool o una pgx.Tx.
func NuevoLector(conn lectorConn) *Lector { return &Lector{conn: conn} }

// Corrida es una fila de importacion tal como la expone la API.
type Corrida struct {
	ID            int64           `json:"id"`
	UsuarioID     *int64          `json:"usuario_id"`
	Origen        string          `json:"origen"`
	Estado        string          `json:"estado"`
	ArchivoRuta   string          `json:"archivo_ruta"`
	ArchivoSHA256 string          `json:"archivo_sha256"`
	Hoja          string          `json:"hoja"`
	Creados       int             `json:"creados"`
	Actualizados  int             `json:"actualizados"`
	Omitidos      int             `json:"omitidos"`
	Observados    int             `json:"observados"`
	Detalle       json.RawMessage `json:"detalle"`
	MensajeError  *string         `json:"mensaje_error"`
	IniciadaEn    time.Time       `json:"iniciada_en"`
	FinalizadaEn  *time.Time      `json:"finalizada_en"`
	// Solo en la ficha.
	Incidencias *[]IncidenciaCorrida `json:"incidencias,omitempty"`
}

// IncidenciaCorrida es una incidencia de una corrida, con los valores leidos y aplicados.
type IncidenciaCorrida struct {
	ID            int64           `json:"id"`
	Tipo          string          `json:"tipo"`
	Regla         *string         `json:"regla"`
	Hoja          string          `json:"hoja"`
	Fila          *int            `json:"fila"`
	Celdas        *string         `json:"celdas"`
	Codigo        *string         `json:"codigo"`
	ServicioN1ID  *int64          `json:"servicio_n1_id"`
	ServicioN2ID  *int64          `json:"servicio_n2_id"`
	ValorOriginal json.RawMessage `json:"valor_original"`
	ValorAplicado json.RawMessage `json:"valor_aplicado"`
	Mensaje       string          `json:"mensaje"`
	CreadaEn      time.Time       `json:"creada_en"`
}

const selectCorrida = `
SELECT id, usuario_id, origen, estado, archivo_ruta, archivo_sha256, hoja, creados, actualizados, omitidos,
       observados, detalle::text, mensaje_error, iniciada_en, finalizada_en
  FROM importacion`

func escanearCorrida(fila pgx.Row) (*Corrida, error) {
	var (
		c       Corrida
		detalle string
	)
	if err := fila.Scan(&c.ID, &c.UsuarioID, &c.Origen, &c.Estado, &c.ArchivoRuta, &c.ArchivoSHA256, &c.Hoja,
		&c.Creados, &c.Actualizados, &c.Omitidos, &c.Observados, &detalle, &c.MensajeError,
		&c.IniciadaEn, &c.FinalizadaEn); err != nil {
		return nil, err
	}
	c.Detalle = json.RawMessage(detalle)
	return &c, nil
}

// Listar devuelve las corridas, la mas reciente primero.
func (l *Lector) Listar(ctx context.Context, p db.Pagina) (db.Listado[*Corrida], error) {
	var total int
	if err := l.conn.QueryRow(ctx, `SELECT count(*) FROM importacion`).Scan(&total); err != nil {
		return db.Listado[*Corrida]{}, fmt.Errorf("contar importaciones: %w", err)
	}
	filas, err := l.conn.Query(ctx, selectCorrida+` ORDER BY iniciada_en DESC, id DESC LIMIT $1 OFFSET $2`,
		p.Tamano, p.Desplazamiento())
	if err != nil {
		return db.Listado[*Corrida]{}, fmt.Errorf("listar importaciones: %w", err)
	}
	defer filas.Close()
	var items []*Corrida
	for filas.Next() {
		c, err := escanearCorrida(filas)
		if err != nil {
			return db.Listado[*Corrida]{}, fmt.Errorf("leer importaciones: %w", err)
		}
		items = append(items, c)
	}
	if err := filas.Err(); err != nil {
		return db.Listado[*Corrida]{}, fmt.Errorf("listar importaciones: %w", err)
	}
	return db.NuevoListado(items, total, p), nil
}

// Obtener devuelve una corrida con todas sus incidencias, por fila.
func (l *Lector) Obtener(ctx context.Context, id int64) (*Corrida, error) {
	c, err := escanearCorrida(l.conn.QueryRow(ctx, selectCorrida+` WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, db.NuevoError(db.ErrNoEncontrado, fmt.Sprintf("La importacion %d no existe.", id))
	}
	if err != nil {
		return nil, fmt.Errorf("leer importacion %d: %w", id, err)
	}
	filas, err := l.conn.Query(ctx, `
		SELECT id, tipo, regla, hoja, fila, celdas, codigo, servicio_n1_id, servicio_n2_id,
		       valor_original::text, valor_aplicado::text, mensaje, creada_en
		  FROM incidencia
		 WHERE importacion_id = $1
		 ORDER BY fila NULLS LAST, id`, id)
	if err != nil {
		return nil, fmt.Errorf("incidencias de la importacion %d: %w", id, err)
	}
	defer filas.Close()
	inc := []IncidenciaCorrida{}
	for filas.Next() {
		var (
			i              IncidenciaCorrida
			original, apli *string
		)
		if err := filas.Scan(&i.ID, &i.Tipo, &i.Regla, &i.Hoja, &i.Fila, &i.Celdas, &i.Codigo, &i.ServicioN1ID,
			&i.ServicioN2ID, &original, &apli, &i.Mensaje, &i.CreadaEn); err != nil {
			return nil, fmt.Errorf("incidencias de la importacion %d: %w", id, err)
		}
		i.ValorOriginal, i.ValorAplicado = jsonCrudo(original), jsonCrudo(apli)
		inc = append(inc, i)
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("incidencias de la importacion %d: %w", id, err)
	}
	c.Incidencias = &inc
	return c, nil
}

// jsonCrudo convierte un jsonb leido como texto en JSON crudo; null si no hay.
func jsonCrudo(s *string) json.RawMessage {
	if s == nil {
		return nil
	}
	return json.RawMessage(*s)
}
