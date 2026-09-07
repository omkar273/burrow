package ent

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"sort"
	"strings"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
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
		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return ierr.Wrap(err, "reading migration "+name).Mark(ierr.ErrInternal)
		}
		if err := applyOne(ctx, db, name, string(body)); err != nil {
			return err
		}
	}
	return nil
}

// applyOne claims a migration by inserting its bookkeeping row first, inside
// the same transaction that runs it.
//
// Checking schema_version before opening the transaction would let two
// processes both observe the migration as unapplied; the loser then runs a
// non-idempotent CREATE TABLE and fails. INSERT OR IGNORE makes the claim and
// the check one atomic step, and a rollback releases it.
func applyOne(ctx context.Context, db *sql.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ierr.Wrap(err, "beginning migration "+name).Mark(ierr.ErrInternal)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version (filename, applied_at) VALUES (?, datetime('now'))`,
		name)
	if err != nil {
		return ierr.Wrap(err, "claiming migration "+name).Mark(ierr.ErrInternal)
	}
	claimed, err := res.RowsAffected()
	if err != nil {
		return ierr.Wrap(err, "claiming migration "+name).Mark(ierr.ErrInternal)
	}
	if claimed == 0 {
		return nil // already applied
	}

	for _, stmt := range strings.Split(body, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return ierr.Wrap(err, "applying migration "+name).Mark(ierr.ErrInternal)
		}
	}
	if err := tx.Commit(); err != nil {
		return ierr.Wrap(err, "committing migration "+name).Mark(ierr.ErrInternal)
	}
	return nil
}
