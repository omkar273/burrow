// Package object models the archive's logical contents. It imports
// nothing third-party and cannot reach the network or the disk.
package object

import "time"

// Kind is the family of thing an object represents.
type Kind string

const (
	KindMessage    Kind = "message"
	KindAttachment Kind = "attachment"
)

// Object is a logical thing in the archive. Its ID is ours and stable
// forever; ExternalID is a provider fact we record but never build
// identity on.
type Object struct {
	ID          string
	SourceID    string
	Kind        Kind
	ExternalID  string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	// DeletedAtSource records that the provider no longer has this object.
	// It is emphatically not a deletion from the archive.
	DeletedAtSource *time.Time
}

// Alias is an additional provider ID that resolves to the same Object.
//
// Restoring a message creates a new provider ID for content we already
// hold; recording it as an alias is what stops restore from forking the
// archive.
type Alias struct {
	ObjectID   string
	SourceID   string
	ExternalID string
	CreatedAt  time.Time
}

// Version is an Object's content at a point in time.
//
// Gmail RAW bytes are immutable, so every Gmail message has exactly one
// version forever. This type is therefore unexercised beyond N=1 until a
// mutable source arrives — treat the first such connector as validating
// it for the first time.
//
// Label changes do not create versions. Labels are mutable derived state;
// versions are created only when content bytes change.
type Version struct {
	ID       string
	ObjectID string
	BlobID   string
	// RestoredFromVersionID records provenance when this version's content
	// re-entered the archive through a restore.
	RestoredFromVersionID *string
	CapturedAt            time.Time
}
