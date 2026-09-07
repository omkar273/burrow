// Package source models a connected provider account and defines the
// contracts a connector must satisfy. It imports nothing third-party.
package source

import "time"

// Kind identifies the connector family.
type Kind string

// KindGmail is the only connector in V1.
const KindGmail Kind = "gmail"

// Status is the operational state of a connection.
type Status string

const (
	StatusActive       Status = "active"
	StatusNeedsAuth    Status = "needs_auth"
	StatusDisconnected Status = "disconnected"
)

// Source is one connected account.
//
// Multiple accounts within one profile are simply multiple Sources: they
// share a blob store, so a message sent to two of your addresses is
// stored once.
type Source struct {
	ID           string
	Kind         Kind
	AccountEmail string
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
