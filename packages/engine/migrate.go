package engine

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofrs/flock"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
	"github.com/omkar273/burrow/packages/engine/internal/validator"
)

// MigrationPlan returns the statements Migrate would run, without running
// them. Empty means the archive already matches this build's schema.
//
// The schema is a portability contract — another Burrow reconstructs an
// archive from it — so an operator can read what will change before it
// touches a database holding their mail.
func MigrationPlan(ctx context.Context, cfg Config) (string, error) {
	return withMigrationDB(ctx, cfg, func(c *sqlitedb.Client) (string, error) {
		return c.PlanSQL(ctx)
	})
}

// SchemaVersion reports how many migrations this archive has applied. It is
// stored in SQLite's user_version header, readable by any SQLite tool, and is
// the version the portable archive bundle carries.
func SchemaVersion(ctx context.Context, cfg Config) (int, error) {
	return withMigrationDB(ctx, cfg, func(c *sqlitedb.Client) (int, error) {
		return c.Version(ctx)
	})
}

// Migrate brings the archive up to this build's schema.
func Migrate(ctx context.Context, cfg Config) error {
	_, err := withMigrationDB(ctx, cfg, func(c *sqlitedb.Client) (struct{}, error) {
		return struct{}{}, c.Migrate(ctx)
	})
	return err
}

// withMigrationDB resolves the profile, takes the same lock Open takes, and
// hands the caller a database handle.
//
// Migrating shares the profile lock deliberately: a running burrow against a
// half-migrated schema is the failure this prevents.
func withMigrationDB[T any](ctx context.Context, cfg Config, fn func(*sqlitedb.Client) (T, error)) (T, error) {
	var zero T

	if err := validator.ValidateRequest(cfg); err != nil {
		return zero, err
	}

	home := cfg.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return zero, ierr.Wrap(err, "resolving home directory").Mark(ierr.ErrInternal)
		}
		home = h
	}

	profile, err := config.ResolveProfile(cfg.Profile, os.Getenv, home)
	if err != nil {
		return zero, err
	}
	if err := profile.Ensure(); err != nil {
		return zero, err
	}

	lock := flock.New(filepath.Join(profile.Root, ".lock"))
	held, err := lock.TryLock()
	if err != nil {
		return zero, ierr.Wrap(err, "locking profile "+profile.Name).Mark(ierr.ErrInternal)
	}
	if !held {
		return zero, ierr.New("profile " + profile.Name + " is already open").
			WithHint("stop the running burrow before migrating").
			Mark(ierr.ErrProfileLocked)
	}
	defer lock.Unlock() //nolint:errcheck

	client, err := sqlitedb.Open(sqlitedb.DSN(profile))
	if err != nil {
		return zero, err
	}
	defer client.Close() //nolint:errcheck

	return fn(client)
}

// TableNames lists the archive's tables. Exists so a caller can confirm a dry
// run changed nothing without reaching into internal packages.
func TableNames(ctx context.Context, cfg Config) ([]string, error) {
	return withMigrationDB(ctx, cfg, func(c *sqlitedb.Client) ([]string, error) {
		rows, err := c.DB().QueryContext(ctx,
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
		if err != nil {
			return nil, ierr.Wrap(err, "listing tables").Mark(ierr.ErrInternal)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				return nil, ierr.Wrap(err, "scanning table name").Mark(ierr.ErrInternal)
			}
			names = append(names, n)
		}
		return names, rows.Err()
	})
}

// SetSchemaVersionForTest forces the recorded schema version. It exists so
// tests can simulate an archive written by a newer build.
func SetSchemaVersionForTest(ctx context.Context, cfg Config, v int) error {
	_, err := withMigrationDB(ctx, cfg, func(c *sqlitedb.Client) (struct{}, error) {
		_, err := c.DB().ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(v)+";")
		return struct{}{}, err
	})
	return err
}
