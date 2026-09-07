package sqlite

import (
	"context"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

type txKey struct{}

// WithTx runs fn inside a transaction, committing when it returns nil and
// rolling back otherwise.
//
// Nesting reuses the outer transaction rather than opening a new one. That is
// not merely a convenience here: the pool is capped at one connection, so a
// nested BeginTx would block forever waiting for the connection the outer
// transaction already holds. Reuse also gives the semantics a caller wants —
// an inner failure rolls the whole unit back, not just its own part.
func (c *Client) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	tx, err := c.ent.Tx(ctx)
	if err != nil {
		return ierr.Wrap(err, "beginning transaction").Mark(ierr.ErrInternal)
	}

	// A panic must not leave the transaction holding the only connection;
	// every later query would block. Roll back, then let it propagate.
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return ierr.Wrap(err, "rollback also failed: "+rbErr.Error()).Mark(ierr.ErrInternal)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ierr.Wrap(err, "committing transaction").Mark(ierr.ErrInternal)
	}
	return nil
}

// TxFromContext returns the transaction carried by ctx, or nil.
func TxFromContext(ctx context.Context) *generated.Tx {
	tx, _ := ctx.Value(txKey{}).(*generated.Tx)
	return tx
}

// Querier returns the transaction's client when ctx carries one, else the
// plain client. Repositories use this so the same method works inside and
// outside a transaction.
func (c *Client) Querier(ctx context.Context) *generated.Client {
	if tx := TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return c.ent
}
