package blob

import "context"

// Repository persists blob records.
type Repository interface {
	Create(ctx context.Context, b *Blob) error
	// Get returns a blob by its own ID.
	Get(ctx context.Context, id string) (Blob, error)
	// GetByContentHash returns a blob by the hash of its bytes. This is
	// the dedup lookup: identical content is stored once.
	GetByContentHash(ctx context.Context, hash string) (Blob, error)
}
