package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

type RestoreResult struct {
	NewExternalID string
	ContentHash   string
	BytesRestored int64
}

type Restore struct{ deps Deps }

// RestoreObject writes an object's current version back to the provider.
//
// The stored bytes are re-hashed and checked before anything is sent. Silently
// restoring corrupted content would be worse than failing: the product's claim
// is that what comes back is what went in.
//
// The provider mints a new id for restored content — IMAP APPEND assigns a new
// UID, as messages.insert assigned a new message id. That id is recorded as an
// alias before returning, so the next sync recognises the restored copy as
// content already held rather than ingesting it as a second object.
//
// This is not idempotent and cannot be made so from here: APPEND creates a
// message on every call. Restore is an explicit operator-initiated act, never
// retried automatically.
func (s *Restore) RestoreObject(
	ctx context.Context,
	r source.Restorer,
	sourceID, objectID string,
	opts source.RestoreOpts,
) (RestoreResult, error) {
	version, err := s.deps.Objects.CurrentVersion(ctx, objectID)
	if err != nil {
		return RestoreResult{}, err
	}

	stored, err := s.deps.Blobs.Get(ctx, version.BlobID)
	if err != nil {
		return RestoreResult{}, err
	}

	rc, err := s.deps.Store.Get(ctx, storage.KeyForHash(stored.ContentHash))
	if err != nil {
		return RestoreResult{}, err
	}
	defer rc.Close()

	raw, err := io.ReadAll(rc)
	if err != nil {
		return RestoreResult{}, ierr.Wrap(err, "reading stored blob").Mark(ierr.ErrInternal)
	}

	sum := sha256.Sum256(raw)
	if actual := "sha256:" + hex.EncodeToString(sum[:]); actual != stored.ContentHash {
		return RestoreResult{}, ierr.New("stored blob does not match its recorded hash").
			WithHint("this replica is corrupt; refetch the object from the source before restoring").
			Mark(ierr.ErrChecksumMismatch)
	}

	newExternalID, err := r.Restore(ctx, bytesReader(raw), opts)
	if err != nil {
		return RestoreResult{}, ierr.Wrap(err, "writing object back to the provider").
			Mark(ierr.ErrSourceUnavailable)
	}

	if err := s.deps.Objects.AddAlias(ctx, objectID, sourceID, newExternalID); err != nil {
		return RestoreResult{}, err
	}

	previous := version.ID
	provenance := &object.Version{
		ID:                    types.NewID(types.PrefixVersion),
		ObjectID:              objectID,
		BlobID:                version.BlobID,
		RestoredFromVersionID: &previous,
		CapturedAt:            time.Now().UTC(),
	}
	if err := s.deps.Objects.CreateVersion(ctx, provenance); err != nil {
		return RestoreResult{}, err
	}

	return RestoreResult{
		NewExternalID: newExternalID,
		ContentHash:   stored.ContentHash,
		BytesRestored: int64(len(raw)),
	}, nil
}
