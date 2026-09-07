package object

import "context"

// Repository persists objects, their aliases, and their versions.
type Repository interface {
	Create(ctx context.Context, o *Object) error
	Get(ctx context.Context, id string) (Object, error)
	// GetByExternalID resolves a provider ID, following aliases. It
	// returns an error marked ErrNotFound when neither the object nor any
	// alias matches.
	GetByExternalID(ctx context.Context, sourceID, externalID string) (Object, error)
	AddAlias(ctx context.Context, objectID, sourceID, externalID string) error
	CreateVersion(ctx context.Context, v *Version) error
	CurrentVersion(ctx context.Context, objectID string) (Version, error)
	// OwnerOfBlob returns the object whose version references blobID.
	// Ingest uses it to attach restored content as an alias rather than
	// minting a second object for bytes it already holds.
	OwnerOfBlob(ctx context.Context, blobID string) (Object, error)
}
