// Package sqlite owns the database handle, transactions, and migrations.
// Package doc: the generated ent client to the engine. Generated
// ent.* types must not escape this package: domain packages own their own
// hand-written structs and FromEnt converters.
package sqlite

import (
	"context"
	"database/sql"
	"net/url"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

// Client is the state database: the handle, transactions, and migrations.
type Client interface {
	// Ent is exported for this package's own tests only. Nothing outside
	// this package may use it: generated types must not escape.
	Ent() *generated.Client
	// Querier returns the transaction's client when ctx carries one, else
	// the plain client. Repositories use this so the same method works
	// inside and outside a transaction.
	Querier(ctx context.Context) *generated.Client
	WithTx(ctx context.Context, fn func(context.Context) error) error
	DB() *sql.DB
	Close() error
	Migrate(ctx context.Context) error
	PlanSQL(ctx context.Context) (string, error)
	Version(ctx context.Context) (int, error)
}

type client struct {
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

// NewClient opens the state database for a profile.
func NewClient(p config.Profile) (Client, error) { return Open(DSN(p)) }

func Open(dsn string) (Client, error) {
	db, err := sql.Open(DriverName, dsn)
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
	return &client{db: db, ent: generated.NewClient(generated.Driver(drv))}, nil
}

func (c *client) Ent() *generated.Client { return c.ent }

func (c *client) DB() *sql.DB { return c.db }

func (c *client) Close() error { return c.db.Close() }

// MemoryDSN is a throwaway in-memory database, used to compute a migration
// plan without creating a real archive.
func MemoryDSN() string {
	return "file:plan?mode=memory&cache=shared&_pragma=foreign_keys(ON)"
}
