package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

//go:embed all:migrations
var migrationFS embed.FS

// Pending is a migration the database has not applied.
type Pending struct {
	Name string
	SQL  string
}

// Version is how many migrations a database has applied. It lives in
// SQLite's user_version header field, so any SQLite tool can read it and no
// bookkeeping table can drift from the schema it describes.
//
// This is the schema version the portable archive bundle carries: an archive
// is readable by any Burrow whose migration set is at least this long.
func Version(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&v); err != nil {
		return 0, ierr.Wrap(err, "reading schema version").Mark(ierr.ErrInternal)
	}
	return v, nil
}

func migrationFiles() ([]string, error) {
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return nil, ierr.Wrap(err, "listing migrations").Mark(ierr.ErrInternal)
	}
	sort.Strings(names)
	return names, nil
}

// ListPending reports what ApplyPending would run, without running it or
// writing anything.
func ListPending(ctx context.Context, db *sql.DB) ([]Pending, error) {
	if err := adoptLegacyBookkeeping(ctx, db); err != nil {
		return nil, err
	}
	applied, err := Version(ctx, db)
	if err != nil {
		return nil, err
	}
	names, err := migrationFiles()
	if err != nil {
		return nil, err
	}
	if applied > len(names) {
		return nil, ierr.New("database is at schema version " + strconv.Itoa(applied) +
			" but this build only has " + strconv.Itoa(len(names)) + " migrations").
			WithHint("this archive was written by a newer Burrow; upgrade before opening it").
			Mark(ierr.ErrValidation)
	}

	pending := make([]Pending, 0, len(names)-applied)
	for _, name := range names[applied:] {
		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return nil, ierr.Wrap(err, "reading migration "+name).Mark(ierr.ErrInternal)
		}
		pending = append(pending, Pending{Name: name, SQL: string(body)})
	}
	return pending, nil
}

// ApplyPending applies every unapplied migration in one transaction and
// returns their names.
//
// user_version is transactional in SQLite, so the DDL and the version bump
// commit or roll back together. A crash mid-migration therefore leaves the
// database exactly as it was, and two processes cannot both claim the same
// migration.
func ApplyPending(ctx context.Context, db *sql.DB) ([]string, error) {
	pending, err := ListPending(ctx, db)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, nil
	}
	names, err := migrationFiles()
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ierr.Wrap(err, "beginning migration").Mark(ierr.ErrInternal)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	applied := make([]string, 0, len(pending))
	for _, m := range pending {
		for _, stmt := range strings.Split(m.SQL, ";") {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return nil, ierr.Wrap(err, "applying migration "+m.Name).Mark(ierr.ErrInternal)
			}
		}
		applied = append(applied, m.Name)
	}

	// Pragmas take no bound parameters; the value is a computed count.
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(len(names))+";"); err != nil {
		return nil, ierr.Wrap(err, "recording schema version").Mark(ierr.ErrInternal)
	}
	if err := tx.Commit(); err != nil {
		return nil, ierr.Wrap(err, "committing migrations").Mark(ierr.ErrInternal)
	}
	return applied, nil
}

// adoptLegacyBookkeeping converts a database written before user_version
// replaced the schema_version table, then drops the table.
//
// Burrow has no released version, so this exists only for archives created
// during development. It can go once none are left.
func adoptLegacyBookkeeping(ctx context.Context, db *sql.DB) error {
	var present int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'schema_version'`,
	).Scan(&present); err != nil {
		return ierr.Wrap(err, "checking for legacy bookkeeping").Mark(ierr.ErrInternal)
	}
	if present == 0 {
		return nil
	}

	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_version`).Scan(&rows); err != nil {
		return ierr.Wrap(err, "reading legacy bookkeeping").Mark(ierr.ErrInternal)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(rows)+";"); err != nil {
		return ierr.Wrap(err, "adopting legacy schema version").Mark(ierr.ErrInternal)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE schema_version`); err != nil {
		return ierr.Wrap(err, "dropping legacy bookkeeping").Mark(ierr.ErrInternal)
	}
	return nil
}

func applyVersioned(ctx context.Context, db *sql.DB) error {
	_, err := ApplyPending(ctx, db)
	return err
}
