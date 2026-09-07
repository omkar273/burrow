package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	"github.com/omkar273/burrow/packages/engine/internal/validator"
)

// Migration is one versioned schema change.
type Migration struct {
	Name    string
	SQL     string
	Applied bool
}

// MigrationPlan reports what Migrate would apply, without applying it.
//
// The schema is a portability contract — another Burrow runtime reconstructs
// an archive from it — so an operator can read the statements before they
// touch a database holding their mail.
func MigrationPlan(ctx context.Context, cfg Config) ([]Migration, error) {
	return withMigrationDB(ctx, cfg, func(c *entrepo.Client) ([]Migration, error) {
		pending, err := c.ListPending(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]Migration, 0, len(pending))
		for _, p := range pending {
			out = append(out, Migration{Name: p.Name, SQL: p.SQL})
		}
		return out, nil
	})
}

// Migrate applies every pending migration and returns the names applied.
func Migrate(ctx context.Context, cfg Config) ([]string, error) {
	return withMigrationDB(ctx, cfg, func(c *entrepo.Client) ([]string, error) {
		return c.ApplyPending(ctx)
	})
}

// withMigrationDB resolves the profile, takes the same lock Open takes, and
// hands the caller a database handle.
//
// Migrating shares the profile lock deliberately: a running burrowd against a
// half-migrated schema is the failure this prevents.
func withMigrationDB[T any](ctx context.Context, cfg Config, fn func(*entrepo.Client) (T, error)) (T, error) {
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
			WithHint("stop the running burrowd before migrating").
			Mark(ierr.ErrProfileLocked)
	}
	defer lock.Unlock() //nolint:errcheck

	client, err := entrepo.Open(entrepo.DSN(profile))
	if err != nil {
		return zero, err
	}
	defer client.Close() //nolint:errcheck

	return fn(client)
}

// TableNames lists the archive's tables. Exists so a caller can confirm a dry
// run changed nothing without reaching into internal packages.
func TableNames(ctx context.Context, cfg Config) ([]string, error) {
	return withMigrationDB(ctx, cfg, func(c *entrepo.Client) ([]string, error) {
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
