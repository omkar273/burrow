package ent

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"sort"
	"strings"

	ierr "github.com/omkar273/burrow/internal/errors"
)

//go:embed all:migrations
var migrationFS embed.FS

// applyVersioned runs every .sql migration not yet recorded, in filename
// order, each in its own transaction.
//
// Versioned migrations rather than ent's auto-migrate because the schema
// is a portability contract: the archive bundle carries a schema version
// and another Burrow runtime must reconstruct state from it. Auto-migrate
// would make the schema an unversioned side effect of Go structs.
func applyVersioned(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_version (
			filename TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`); err != nil {
		return ierr.Wrap(err, "creating schema_version table").Mark(ierr.ErrInternal)
	}

	entries, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return ierr.Wrap(err, "listing migrations").Mark(ierr.ErrInternal)
	}
	sort.Strings(entries)

	for _, name := range entries {
		var seen int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(1) FROM schema_version WHERE filename = ?`, name,
		).Scan(&seen); err != nil {
			return ierr.Wrap(err, "checking migration "+name).Mark(ierr.ErrInternal)
		}
		if seen > 0 {
			continue
		}

		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return ierr.Wrap(err, "reading migration "+name).Mark(ierr.ErrInternal)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return ierr.Wrap(err, "beginning migration "+name).Mark(ierr.ErrInternal)
		}
		for _, stmt := range strings.Split(string(body), ";") {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				_ = tx.Rollback()
				return ierr.Wrap(err, "applying migration "+name).Mark(ierr.ErrInternal)
			}
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_version (filename, applied_at) VALUES (?, datetime('now'))`,
			name,
		); err != nil {
			_ = tx.Rollback()
			return ierr.Wrap(err, "recording migration "+name).Mark(ierr.ErrInternal)
		}
		if err := tx.Commit(); err != nil {
			return ierr.Wrap(err, "committing migration "+name).Mark(ierr.ErrInternal)
		}
	}
	return nil
}
