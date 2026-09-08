// Package service holds the engine's use cases. It orchestrates repositories
// and adapters; adapters never call upward into it.
package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"io"
	"time"

	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

type IngestResult struct {
	ObjectID    string
	VersionID   string
	ContentHash string
	// Deduplicated: this provider id was already held, directly or via an
	// alias recorded by a restore.
	Deduplicated bool
	BlobReused   bool
}

type Ingest struct{ params ServiceParams }

func NewIngest(p ServiceParams) *Ingest { return &Ingest{params: p} }

// IngestOne copies a single object into the archive.
//
// The order is not an implementation detail. Bytes are fetched and hashed,
// the blob is written and made durable, and only then is metadata committed.
// There is no transaction spanning storage and the database: a crash between
// the write and the commit leaves an orphan blob, which is inert and
// collectable because keys are content-addressed. The reverse order would
// leave a row pointing at bytes that do not exist.
//
// Identity comes from the provider, never from content. Two distinct messages
// can share bytes, so a hash match reuses the blob row and nothing else — only
// restore knows lineage, and only restore records an alias.
func (s *Ingest) IngestOne(ctx context.Context, conn source.Connector, ref source.ObjectRef) (IngestResult, error) {
	sourceID := conn.SourceID()

	if existing, err := s.params.Objects.GetByExternalID(ctx, sourceID, ref.ExternalID); err == nil {
		version, err := s.params.Objects.CurrentVersion(ctx, existing.ID)
		if err != nil {
			return IngestResult{}, err
		}
		held, err := s.params.Blobs.Get(ctx, version.BlobID)
		if err != nil {
			return IngestResult{}, err
		}
		return IngestResult{
			ObjectID: existing.ID, VersionID: version.ID,
			ContentHash: held.ContentHash, Deduplicated: true,
		}, nil
	} else if !isNotFound(err) {
		return IngestResult{}, err
	}

	body, _, err := conn.Fetch(ctx, ref)
	if err != nil {
		return IngestResult{}, err
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return IngestResult{}, ierr.Wrap(err, "reading object body").Mark(ierr.ErrSourceUnavailable)
	}

	sum := sha256.Sum256(raw)
	contentHash := "sha256:" + hex.EncodeToString(sum[:])

	blobID, reused, err := s.ensureBlob(ctx, contentHash, raw)
	if err != nil {
		return IngestResult{}, err
	}

	now := time.Now().UTC()
	obj := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: sourceID,
		Kind: object.KindMessage, ExternalID: ref.ExternalID,
		FirstSeenAt: now, LastSeenAt: now,
	}
	version := &object.Version{
		ID: types.NewID(types.PrefixVersion), ObjectID: obj.ID,
		BlobID: blobID, CapturedAt: now,
	}

	// One transaction: an object with no version is a row pointing at nothing.
	if err := s.params.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.params.Objects.Create(ctx, obj); err != nil {
			return err
		}
		return s.params.Objects.CreateVersion(ctx, version)
	}); err != nil {
		return IngestResult{}, err
	}

	return IngestResult{
		ObjectID: obj.ID, VersionID: version.ID,
		ContentHash: contentHash, BlobReused: reused,
	}, nil
}

// ensureBlob reuses the existing row for contentHash, or writes raw and
// creates one.
func (s *Ingest) ensureBlob(ctx context.Context, contentHash string, raw []byte) (id string, reused bool, err error) {
	existing, err := s.params.Blobs.GetByContentHash(ctx, contentHash)
	switch {
	case err == nil:
		return existing.ID, true, nil
	case !isNotFound(err):
		return "", false, err
	}

	if err := s.params.Store.Put(ctx, storage.KeyForHash(contentHash), bytes.NewReader(raw), int64(len(raw))); err != nil {
		return "", false, ierr.Wrap(err, "writing blob").Mark(ierr.ErrInternal)
	}

	b := &blob.Blob{
		ID: types.NewID(types.PrefixBlob), ContentHash: contentHash,
		SizeBytes: int64(len(raw)),
	}
	if err := s.params.Blobs.Create(ctx, b); err != nil {
		return "", false, err
	}
	return b.ID, false, nil
}

func isNotFound(err error) bool { return stderrors.Is(err, ierr.ErrNotFound) }

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
