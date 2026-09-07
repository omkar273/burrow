package source

import (
	"context"
	"io"
	"time"
)

type Cursor string

type Checkpoint string

type ObjectRef struct {
	ExternalID string
	ThreadID   string
}

// Provider metadata absent from the content itself: Gmail labels and
// internalDate have no RFC822 representation.
type ObjectMeta struct {
	ExternalID   string
	ThreadID     string
	Labels       []string
	InternalDate time.Time
	SizeEstimate int64
}

type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeDeleted ChangeKind = "deleted"
	ChangeLabeled ChangeKind = "labeled"
)

type Change struct {
	Kind ChangeKind
	Ref  ObjectRef
}

type Connector interface {
	SourceID() string
	FullList(ctx context.Context, cur Cursor) (refs []ObjectRef, next Cursor, err error)
	// Returns ErrCheckpointExpired when the provider rejects cp and a full
	// resync is required. Gmail 404s a stale historyId after about a week,
	// so every caller needs that path — hence a typed error.
	Changes(ctx context.Context, cp Checkpoint) (changes []Change, next Checkpoint, err error)
	Fetch(ctx context.Context, ref ObjectRef) (io.ReadCloser, ObjectMeta, error)
	Health(ctx context.Context) error
}

type RestoreOpts struct {
	Labels []string
}

// Separate from Connector because reading and writing back are different
// capabilities: a connector may be read-only, and Burrow requests the
// write scope only when a restore is first attempted.
type Restorer interface {
	Restore(ctx context.Context, r io.Reader, opts RestoreOpts) (externalID string, err error)
}
