package object

import "context"

type Repository interface {
	Create(ctx context.Context, o *Object) error
	Get(ctx context.Context, id string) (Object, error)
	// Follows aliases. ErrNotFound when neither object nor alias matches.
	GetByExternalID(ctx context.Context, sourceID, externalID string) (Object, error)
	AddAlias(ctx context.Context, objectID, sourceID, externalID string) error
	CreateVersion(ctx context.Context, v *Version) error
	CurrentVersion(ctx context.Context, objectID string) (Version, error)
}
