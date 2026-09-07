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

type Client struct {
	db  *sql.DB
	ent *generated.Client
}

// WAL lets readers proceed during a write; busy_timeout makes concurrent
// writers wait instead of returning SQLITE_BUSY. foreign_keys is off by
// default in SQLite and must be asked for.
func FileDSN(path string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(ON)")
	return "file:" + path + "?" + q.Encode()
}

func DSN(p config.Profile) string { return FileDSN(p.StateDB) }

func Open(dsn string) (*Client, error) {
	db, err := sql.Open(sqlited.DriverName, dsn)
	if err != nil {
		return nil, ierr.Wrap(err, "opening state database").Mark(ierr.ErrInternal)
	}

	// SQLite tolerates one writer, and a single conn keeps the
	// connection-scoped pragmas above applying to every query.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, ierr.Wrap(err, "connecting to state database").Mark(ierr.ErrInternal)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	return &Client{db: db, ent: generated.NewClient(generated.Driver(drv))}, nil
}

// Ent is exported for this package's repositories only. Nothing outside
// this package may use it: generated types must not escape.
func (c *Client) Ent() *generated.Client { return c.ent }

func (c *Client) DB() *sql.DB { return c.db }

func (c *Client) Close() error { return c.db.Close() }

func (c *Client) Migrate(ctx context.Context) error {
	return applyVersioned(ctx, c.db)
}
