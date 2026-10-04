// Package db abre el pool de PostgreSQL y aplica las migraciones embebidas.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx" para goose
	"github.com/pressly/goose/v3"
)

//go:embed migraciones/*.sql
var migraciones embed.FS

// Pool envuelve pgxpool y expone la comprobacion que usa /healthz.
type Pool struct{ *pgxpool.Pool }

// Abrir crea el pool y comprueba que la base responde.
func Abrir(ctx context.Context, url string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("abrir pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("la base no responde: %w", err)
	}
	return &Pool{pool}, nil
}

// Salud ejecuta SELECT 1.
func (p *Pool) Salud(ctx context.Context) error {
	var uno int
	return p.QueryRow(ctx, "SELECT 1").Scan(&uno)
}

// Migrar aplica las migraciones pendientes y devuelve la version antes y despues.
func Migrar(ctx context.Context, url string) (antes, despues int64, err error) {
	conn, err := sql.Open("pgx", url)
	if err != nil {
		return 0, 0, fmt.Errorf("abrir conexion de migracion: %w", err)
	}
	defer conn.Close()

	goose.SetBaseFS(migraciones)
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, 0, err
	}
	if antes, err = goose.GetDBVersionContext(ctx, conn); err != nil {
		return 0, 0, fmt.Errorf("leer version: %w", err)
	}
	if err := goose.UpContext(ctx, conn, "migraciones"); err != nil {
		return antes, 0, fmt.Errorf("aplicar migraciones: %w", err)
	}
	if despues, err = goose.GetDBVersionContext(ctx, conn); err != nil {
		return antes, 0, fmt.Errorf("leer version: %w", err)
	}
	return antes, despues, nil
}
