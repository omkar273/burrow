package service

import "context"

// Txer runs a function inside a database transaction.
//
// Declared here rather than imported from the sqlite package so the use cases
// do not depend on a concrete database: ingest needs blob, object and version
// to commit together, not on SQLite specifically. Nested calls reuse the outer
// transaction, so a service calling another service does not deadlock.
type Txer interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
