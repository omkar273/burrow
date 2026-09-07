package ent

import (
	"context"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	entblob "github.com/omkar273/burrow/packages/engine/ent/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

type blobRepository struct{ c *Client }

func NewBlobRepository(c *Client) blob.Repository { return &blobRepository{c: c} }

func blobFromEnt(b *generated.Blob) blob.Blob {
	return blob.Blob{
		ID:          b.ID,
		ContentHash: b.ContentHash,
		SizeBytes:   b.SizeBytes,
		CreatedAt:   b.CreatedAt,
	}
}

func (r *blobRepository) Create(ctx context.Context, b *blob.Blob) error {
	err := r.c.ent.Blob.Create().
		SetID(b.ID).
		SetContentHash(b.ContentHash).
		SetSizeBytes(b.SizeBytes).
		Exec(ctx)
	if err != nil {
		return ierr.Wrap(err, "creating blob").Mark(ierr.ErrInternal)
	}
	return nil
}

func (r *blobRepository) Get(ctx context.Context, id string) (blob.Blob, error) {
	row, err := r.c.ent.Blob.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return blob.Blob{}, ierr.New("no blob with id " + id).Mark(ierr.ErrNotFound)
		}
		return blob.Blob{}, ierr.Wrap(err, "getting blob").Mark(ierr.ErrInternal)
	}
	return blobFromEnt(row), nil
}

func (r *blobRepository) GetByContentHash(ctx context.Context, hash string) (blob.Blob, error) {
	row, err := r.c.ent.Blob.Query().Where(entblob.ContentHash(hash)).Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return blob.Blob{}, ierr.New("no blob with content hash " + hash).Mark(ierr.ErrNotFound)
		}
		return blob.Blob{}, ierr.Wrap(err, "getting blob by content hash").Mark(ierr.ErrInternal)
	}
	return blobFromEnt(row), nil
}
