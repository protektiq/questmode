package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"questmode/backend/migrations"
)

const maxSQLBytes = 4 << 20 // 4 MiB per migration file

// Run applies embedded SQL migrations in lexicographic filename order. Each file runs in a
// transaction; applied versions are recorded in schema_migrations.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	names, err := listMigrationNames()
	if err != nil {
		return err
	}

	for _, name := range names {
		var applied string
		err := pool.QueryRow(ctx,
			`SELECT version FROM schema_migrations WHERE version = $1`, name,
		).Scan(&applied)
		if err == nil {
			continue
		}
		if err != pgx.ErrNoRows {
			return fmt.Errorf("check %s: %w", name, err)
		}

		body, err := readMigrationFile(name)
		if err != nil {
			return err
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("exec %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}

	return nil
}

func listMigrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrations.SQL, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".sql") {
			continue
		}
		if n == "" {
			continue
		}
		names = append(names, n)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no embedded .sql migrations found")
	}
	sort.Strings(names)
	return names, nil
}

func readMigrationFile(name string) ([]byte, error) {
	p := path.Clean(name)
	if p != name || strings.Contains(p, "..") || !strings.HasSuffix(p, ".sql") {
		return nil, fmt.Errorf("invalid migration name %q", name)
	}
	b, err := migrations.SQL.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if len(b) > maxSQLBytes {
		return nil, fmt.Errorf("migration %s exceeds max size", name)
	}
	return b, nil
}
