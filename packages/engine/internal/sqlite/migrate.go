package sqlite

import (
	"bytes"
	"context"
	"strconv"

	entmigrate "entgo.io/ent/dialect/sql/schema"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

// SchemaVersion identifies the shape of the archive. Bump it whenever
// ent/schema changes in a way another Burrow would need to understand.
//
// It is written to SQLite's user_version header, so any tool can read what
// schema an archive holds — the version the portable archive bundle carries.
const SchemaVersion = 1

// migrateOptions keeps the two behaviours that matter explicit rather than
// inherited from ent's defaults.
//
// Foreign keys are on because dangling references are the corruption class
// this product exists to avoid. Dropping is off because losing a column
// silently loses archived data; a destructive change must be a deliberate,
// hand-written act.
func migrateOptions() []entmigrate.MigrateOption {
	return []entmigrate.MigrateOption{
		entmigrate.WithForeignKeys(true),
		entmigrate.WithDropColumn(false),
		entmigrate.WithDropIndex(false),
	}
}

// schemaVersion reports the schema version recorded in the database.
func schemaVersion(ctx context.Context, c *client) (int, error) {
	var v int
	if err := c.db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&v); err != nil {
		return 0, ierr.Wrap(err, "reading schema version").Mark(ierr.ErrInternal)
	}
	return v, nil
}

// PlanSQL returns the statements Migrate would run, without running them.
// An empty string means the database already matches ent/schema.
func (c *client) PlanSQL(ctx context.Context) (string, error) {
	var buf bytes.Buffer
	if err := c.ent.Schema.WriteTo(ctx, &buf, migrateOptions()...); err != nil {
		return "", ierr.Wrap(err, "planning schema migration").Mark(ierr.ErrInternal)
	}
	return buf.String(), nil
}

// Migrate brings the database up to ent/schema and records SchemaVersion.
func (c *client) Migrate(ctx context.Context) error {
	recorded, err := schemaVersion(ctx, c)
	if err != nil {
		return err
	}
	if recorded > SchemaVersion {
		return ierr.New("archive is at schema version " + strconv.Itoa(recorded) +
			" but this build understands " + strconv.Itoa(SchemaVersion)).
			WithHint("this archive was written by a newer Burrow; upgrade before opening it").
			Mark(ierr.ErrValidation)
	}

	if err := c.ent.Schema.Create(ctx, migrateOptions()...); err != nil {
		return ierr.Wrap(err, "applying schema migration").Mark(ierr.ErrInternal)
	}
	if _, err := c.db.ExecContext(ctx,
		"PRAGMA user_version = "+strconv.Itoa(SchemaVersion)+";"); err != nil {
		return ierr.Wrap(err, "recording schema version").Mark(ierr.ErrInternal)
	}
	return nil
}

// Version reports the schema version recorded in the database.
func (c *client) Version(ctx context.Context) (int, error) { return schemaVersion(ctx, c) }
