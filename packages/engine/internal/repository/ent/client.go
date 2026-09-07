// Package ent adapts the generated ent client to the engine. Generated
// ent.* types must not escape this package: domain packages own their own
// hand-written structs and FromEnt converters.
package ent

import (
	"context"
	"database/sql"
	"net/url"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/sqlited"
)

// Client owns the database handle and the generated ent client.
type Client struct {
	db  *sql.DB
	ent *generated.Client
}

// FileDSN builds a DSN for a SQLite file.
//
// WAL allows readers to proceed during a write, and busy_timeout makes
// concurrent writers wait rather than immediately returning SQLITE_BUSY.
// Both matter once background workers share one file. foreign_keys is off
// by default in SQLite and must be asked for.
func FileDSN(path string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	return "file:" + path + "?" + q.Encode()
}

// DSN builds the DSN for a profile's state database.
func DSN(p config.Profile) string { return FileDSN(p.StateDB) }

// Open connects to SQLite and wraps the generated client.
func Open(dsn string) (*Client, error) {
	db, err := sql.Open(sqlited.DriverName, dsn)
	if err != nil {
		return nil, ierr.Wrap(err, "opening state database").Mark(ierr.ErrInternal)
	}

	// SQLite tolerates one writer. Serialising here is simpler and more
	// predictable than relying on busy_timeout alone under contention, and
	// it keeps the connection-scoped pragmas above applying to every query.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, ierr.Wrap(err, "connecting to state database").Mark(ierr.ErrInternal)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	return &Client{db: db, ent: generated.NewClient(generated.Driver(drv))}, nil
}

// Ent exposes the generated client. Callers outside this package must not
// use it; repository implementations in this package may.
func (c *Client) Ent() *generated.Client { return c.ent }

// DB exposes the raw handle for pragma checks and the jobs table.
func (c *Client) DB() *sql.DB { return c.db }

// Close releases the database handle.
func (c *Client) Close() error { return c.db.Close() }

// Migrate applies pending versioned migrations.
func (c *Client) Migrate(ctx context.Context) error {
	return applyVersioned(ctx, c.db)
}
