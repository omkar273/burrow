package blob

import "context"

type Repository interface {
	Create(ctx context.Context, b *Blob) error
	Get(ctx context.Context, id string) (Blob, error)
	// The dedup lookup: identical content is stored once.
	GetByContentHash(ctx context.Context, hash string) (Blob, error)
}
