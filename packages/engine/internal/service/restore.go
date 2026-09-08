package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/dto"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

// RestoreService writes an object's current version back to its source.
type RestoreService interface {
	RestoreObject(ctx context.Context, r source.Restorer, req *dto.RestoreRequest) (*dto.RestoreResponse, error)
}

type restoreService struct {
	params *ServiceParams
}

// p is taken by value because ServiceParams embeds fx.In: fx must populate
// this exact struct type, not a pointer to it. The address is taken once
// here so every call site after that shares one copy instead of holding its
// own.
func NewRestoreService(p ServiceParams) RestoreService {
	return &restoreService{params: &p}
}

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
func (s *restoreService) RestoreObject(ctx context.Context, r source.Restorer, req *dto.RestoreRequest) (*dto.RestoreResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	version, err := s.params.Objects.CurrentVersion(ctx, req.ObjectID)
	if err != nil {
		return nil, err
	}

	stored, err := s.params.Blobs.Get(ctx, version.BlobID)
	if err != nil {
		return nil, err
	}

	rc, err := s.params.Store.Get(ctx, storage.KeyForHash(stored.ContentHash))
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, ierr.Wrap(err, "reading stored blob").Mark(ierr.ErrInternal)
	}

	sum := sha256.Sum256(raw)
	if actual := "sha256:" + hex.EncodeToString(sum[:]); actual != stored.ContentHash {
		return nil, ierr.New("stored blob does not match its recorded hash").
			WithHint("this replica is corrupt; refetch the object from the source before restoring").
			Mark(ierr.ErrChecksumMismatch)
	}

	newExternalID, err := r.Restore(ctx, bytesReader(raw), source.RestoreOpts{Labels: req.Labels})
	if err != nil {
		return nil, ierr.Wrap(err, "writing object back to the provider").
			Mark(ierr.ErrSourceUnavailable)
	}

	if err := s.params.Objects.AddAlias(ctx, req.ObjectID, req.SourceID, newExternalID); err != nil {
		return nil, err
	}

	previous := version.ID
	provenance := &object.Version{
		ID:                    types.NewID(types.PrefixVersion),
		ObjectID:              req.ObjectID,
		BlobID:                version.BlobID,
		RestoredFromVersionID: &previous,
		CapturedAt:            time.Now().UTC(),
	}
	if err := s.params.Objects.CreateVersion(ctx, provenance); err != nil {
		return nil, err
	}

	return &dto.RestoreResponse{
		NewExternalID: newExternalID,
		ContentHash:   stored.ContentHash,
		BytesRestored: int64(len(raw)),
	}, nil
}
