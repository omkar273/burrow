// Package source models a connected provider account and defines the
// contracts a connector must satisfy. It imports nothing third-party.
package source

import "time"

type Kind string

const KindGmail Kind = "gmail"

type Status string

const (
	StatusActive       Status = "active"
	StatusNeedsAuth    Status = "needs_auth"
	StatusDisconnected Status = "disconnected"
)

// Multiple accounts in one profile are just multiple Sources sharing a
// blob store, so a message sent to two of your addresses is stored once.
type Source struct {
	ID           string
	Kind         Kind
	AccountEmail string
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
