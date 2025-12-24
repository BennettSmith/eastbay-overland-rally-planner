package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenMigratedPool opens a Postgres pool against PG_DSN and applies repo migrations.
//
// It is destructive: it resets the public schema.
func OpenMigratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		t.Skip("PG_DSN not set; skipping Postgres contract tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse PG_DSN: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	ac, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire conn: %v", err)
	}
	defer ac.Release()

	conn := ac.Conn()
	if err := resetPublicSchema(ctx, conn); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	migrationsDir := filepath.Join(repoRoot(t), "migrations")
	if err := applyMigrations(ctx, conn, migrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return pool
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Most test runs execute from package directory; use CWD to find repo root.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Walk up until we find go.mod.
	dir := cwd
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root from cwd=%s", cwd)
	return ""
}

func resetPublicSchema(ctx context.Context, conn *pgx.Conn) error {
	sql := `
		DROP SCHEMA IF EXISTS public CASCADE;
		CREATE SCHEMA public;
	`
	return execMulti(ctx, conn, sql)
}

func applyMigrations(ctx context.Context, conn *pgx.Conn, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var ups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			ups = append(ups, filepath.Join(dir, name))
		}
	}
	sort.Strings(ups)
	for _, path := range ups {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := execMulti(ctx, conn, string(b)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

func execMulti(ctx context.Context, conn *pgx.Conn, sql string) error {
	res, err := conn.PgConn().Exec(ctx, sql).ReadAll()
	if err != nil {
		return err
	}
	for _, r := range res {
		if r.Err != nil {
			// Surface server-side errors.
			if pe, ok := r.Err.(*pgconn.PgError); ok {
				return fmt.Errorf("postgres error: %s (%s)", pe.Message, pe.Code)
			}
			return r.Err
		}
	}
	return nil
}
