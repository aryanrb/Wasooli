package db

import (
	"context"
	"embed"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
)

// New opens and verifies a pgx connection pool.
func New(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// RunMigrations applies all pending Tern migrations embedded under migrations/.
// Safe to run on every boot — Tern skips already-applied versions.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsFS embed.FS) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	m, err := migrate.NewMigrator(ctx, conn.Conn(), "schema_version")
	if err != nil {
		return err
	}
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	if err := m.LoadMigrations(sub); err != nil {
		return err
	}
	return m.Migrate(ctx)
}
