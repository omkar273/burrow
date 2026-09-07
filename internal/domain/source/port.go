package source

import (
	"context"
	"io"
	"time"
)

// Cursor resumes a full listing. Opaque to callers.
type Cursor string

// Checkpoint resumes an incremental sync. Opaque to callers.
type Checkpoint string

// ObjectRef identifies one object at the provider.
type ObjectRef struct {
	ExternalID string
	ThreadID   string
}

// ObjectMeta is provider metadata that does not live in the content
// itself — Gmail labels and internalDate have no RFC822 representation.
type ObjectMeta struct {
	ExternalID   string
	ThreadID     string
	Labels       []string
	InternalDate time.Time
	SizeEstimate int64
}

// ChangeKind distinguishes what happened to an object at the provider.
type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeDeleted ChangeKind = "deleted"
	ChangeLabeled ChangeKind = "labeled"
)

// Change is one provider-side event.
type Change struct {
	Kind ChangeKind
	Ref  ObjectRef
}

// Connector reads from a provider.
type Connector interface {
	SourceID() string
	FullList(ctx context.Context, cur Cursor) (refs []ObjectRef, next Cursor, err error)
	// Changes returns provider events since cp.
	//
	// It returns an error marked ErrCheckpointExpired when the provider
	// rejects cp and a full resync is required. Gmail returns 404 on a
	// stale historyId, typically after about a week, so every caller must
	// have a full-resync path. This is why it is a typed error rather
	// than a special return value.
	Changes(ctx context.Context, cp Checkpoint) (changes []Change, next Checkpoint, err error)
	Fetch(ctx context.Context, ref ObjectRef) (io.ReadCloser, ObjectMeta, error)
	Health(ctx context.Context) error
}

// RestoreOpts controls how content is written back.
type RestoreOpts struct {
	// Labels to apply to the restored object, where the provider supports it.
	Labels []string
}

// Restorer writes content back to a provider.
//
// It is a separate interface from Connector because reading and writing
// back are different capabilities: a future connector may be read-only,
// and Burrow requests the write scope only when a restore is first
// attempted.
type Restorer interface {
	Restore(ctx context.Context, r io.Reader, opts RestoreOpts) (externalID string, err error)
}
