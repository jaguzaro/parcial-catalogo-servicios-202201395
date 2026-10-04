package catalogo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"catalogo/internal/db"
)

// AsignarResponsable fija la seccion responsable y, opcionalmente, un usuario que
// pertenezca a esa seccion. Los dos en nil quitan la asignacion. Ver docs/diseno/reglas.md,
// "Responsable de un servicio".
func (s *Servicio) AsignarResponsable(ctx context.Context, id int64, seccionID, usuarioID *int64) (*ServicioN2, error) {
	if usuarioID != nil && seccionID == nil {
		return nil, db.Validacion("Indique la seccion responsable: el usuario responsable se asigna siempre dentro de una seccion.",
			"seccion_id")
	}
	var sv *ServicioN2
	err := s.enTx(ctx, func(tx pgx.Tx) error {
		d, err := bloquearN2(ctx, tx, id)
		if err != nil {
			return err
		}
		// Quitar la asignacion de un servicio dado de baja si se permite: no asigna nada.
		if d.activo == ActivoN && seccionID != nil {
			e := db.NuevoError(db.ErrServicioInactivo,
				fmt.Sprintf("El servicio %s esta dado de baja (activo = N): no se le asigna responsable. Reactivelo primero.", d.codigo))
			e.Detalle = db.Ref{Entidad: "servicio_n2", ID: id, Codigo: d.codigo, Nombre: d.nombre}
			return e
		}
		if seccionID != nil {
			seccion, err := verificarSeccion(ctx, tx, *seccionID)
			if err != nil {
				return err
			}
			if usuarioID != nil {
				if err := verificarUsuario(ctx, tx, *usuarioID, seccion); err != nil {
					return err
				}
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE servicio_n2 SET seccion_responsable_id = $2, usuario_responsable_id = $3, actualizado_en = now() WHERE id = $1`,
			id, seccionID, usuarioID); err != nil {
			return fmt.Errorf("asignar responsable de servicio_n2 %d: %w", id, err)
		}
		sv, err = ficha(ctx, tx, id)
		return err
	})
	return sv, err
}

// verificarSeccion lee la seccion con FOR SHARE: existe y esta activa.
func verificarSeccion(ctx context.Context, tx pgx.Tx, id int64) (db.Ref, error) {
	ref := db.Ref{Entidad: "seccion", ID: id}
	var activo bool
	err := tx.QueryRow(ctx, `SELECT codigo, nombre, activo FROM seccion WHERE id = $1 FOR SHARE`, id).
		Scan(&ref.Codigo, &ref.Nombre, &activo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, db.NuevoError(db.ErrReferenciaInexistente, fmt.Sprintf("No existe una seccion con id %d (campo seccion_id).", id), "seccion_id")
	}
	if err != nil {
		return ref, fmt.Errorf("leer seccion %d: %w", id, err)
	}
	if !activo {
		e := db.NuevoError(db.ErrPadreInactivo,
			fmt.Sprintf("La seccion %s (id %d) esta inactiva: no se le asignan servicios.", ref.Codigo, id), "seccion_id")
		e.Detalle = ref
		return ref, e
	}
	return ref, nil
}

// verificarUsuario exige un usuario existente, activo y cuyo puesto sea de la seccion
// pedida. Bloquea el usuario y su puesto, para que no los muevan de seccion a la vez.
func verificarUsuario(ctx context.Context, tx pgx.Tx, id int64, seccion db.Ref) error {
	var (
		usuario string
		activo  bool
		propia  db.Ref
	)
	err := tx.QueryRow(ctx,
		`SELECT u.usuario, u.activo, s.id, s.codigo, s.nombre
		   FROM usuario u
		   JOIN puesto p  ON p.id = u.puesto_id
		   JOIN seccion s ON s.id = p.seccion_id
		  WHERE u.id = $1
		    FOR SHARE OF u, p`, id).
		Scan(&usuario, &activo, &propia.ID, &propia.Codigo, &propia.Nombre)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.NuevoError(db.ErrReferenciaInexistente, fmt.Sprintf("No existe un usuario con id %d (campo usuario_id).", id), "usuario_id")
	}
	if err != nil {
		return fmt.Errorf("leer usuario %d: %w", id, err)
	}
	if !activo {
		e := db.NuevoError(db.ErrPadreInactivo,
			fmt.Sprintf("El usuario %s esta inactivo: no se le asignan servicios.", usuario), "usuario_id")
		e.Detalle = db.Ref{Entidad: "usuario", ID: id, Codigo: usuario}
		return e
	}
	if propia.ID != seccion.ID {
		propia.Entidad = "seccion"
		e := db.NuevoError(db.ErrFueraDeSeccion,
			fmt.Sprintf("El usuario %s pertenece a la seccion %s (id %d) y la seccion responsable pedida es %s (id %d): "+
				"el usuario responsable tiene que pertenecer a la seccion responsable.",
				usuario, propia.Codigo, propia.ID, seccion.Codigo, seccion.ID), "usuario_id")
		e.Detalle = map[string]any{
			"usuario":         db.Ref{Entidad: "usuario", ID: id, Codigo: usuario},
			"seccion_usuario": propia,
			"seccion_pedida":  seccion,
		}
		return e
	}
	return nil
}
