package service_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"io"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/dto"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

// The acceptance test for the whole engine, in miniature: bytes out equal
// bytes in. M0 proved this against a real mailbox; this holds it in place.
func TestRestoredBytesHashMatchStoredBytes(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	ingested, err := h.ingest.IngestOne(t.Context(), h.src, &dto.IngestRequest{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	res, err := h.restore.RestoreObject(t.Context(), h.src, &dto.RestoreRequest{SourceID: h.src.SourceID(), ObjectID: ingested.ObjectID})
	if err != nil {
		t.Fatalf("RestoreObject: %v", err)
	}

	rc, _, err := h.src.Fetch(t.Context(), source.ObjectRef{ExternalID: res.NewExternalID})
	if err != nil {
		t.Fatalf("fetch restored: %v", err)
	}
	defer rc.Close()
	back, _ := io.ReadAll(rc)

	sum := sha256.Sum256(back)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != ingested.ContentHash {
		t.Fatalf("restored hash %s != stored hash %s", got, ingested.ContentHash)
	}
	if !bytes.Equal(back, []byte(rawMessage)) {
		t.Fatal("restored bytes differ from the original")
	}
}

// Restoring corrupted bytes silently is worse than failing: the product's
// whole claim is that what comes back is what went in.
func TestRestoreRefusesCorruptedBlobs(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	ingested, err := h.ingest.IngestOne(t.Context(), h.src, &dto.IngestRequest{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	h.store.Corrupt(storage.KeyForHash(ingested.ContentHash), []byte("tampered"))

	_, err = h.restore.RestoreObject(t.Context(), h.src, &dto.RestoreRequest{SourceID: h.src.SourceID(), ObjectID: ingested.ObjectID})
	if !stderrors.Is(err, ierr.ErrChecksumMismatch) {
		t.Fatalf("err = %v, want ErrChecksumMismatch", err)
	}
}

func TestRestoreOfUnknownObjectIsErrNotFound(t *testing.T) {
	h := newHarness(t)
	_, err := h.restore.RestoreObject(t.Context(), h.src, &dto.RestoreRequest{SourceID: h.src.SourceID(), ObjectID: "obj_01J0000000000000000000ZZ"})
	if !stderrors.Is(err, ierr.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// Restore mints a new provider id. Recording it as an alias immediately is
// what stops the next sync ingesting the restored copy as a second object.
func TestRestoreRecordsTheNewIDAsAnAlias(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	ingested, err := h.ingest.IngestOne(t.Context(), h.src, &dto.IngestRequest{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	res, err := h.restore.RestoreObject(t.Context(), h.src, &dto.RestoreRequest{SourceID: h.src.SourceID(), ObjectID: ingested.ObjectID})
	if err != nil {
		t.Fatalf("RestoreObject: %v", err)
	}

	resolved, err := h.objects.GetByExternalID(t.Context(), h.src.SourceID(), res.NewExternalID)
	if err != nil {
		t.Fatalf("the restored id was not recorded as an alias: %v", err)
	}
	if resolved.ID != ingested.ObjectID {
		t.Fatalf("alias resolves to %q, want %q", resolved.ID, ingested.ObjectID)
	}

	// And a later sync of that id must not fork the archive.
	after, err := h.ingest.IngestOne(t.Context(), h.src, &dto.IngestRequest{ExternalID: res.NewExternalID})
	if err != nil {
		t.Fatalf("ingest restored: %v", err)
	}
	if after.ObjectID != ingested.ObjectID {
		t.Fatal("re-syncing a restored message forked the archive")
	}
}

// Provenance: the new version records which version it came from.
func TestRestoreRecordsProvenance(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	ingested, err := h.ingest.IngestOne(t.Context(), h.src, &dto.IngestRequest{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if _, err := h.restore.RestoreObject(t.Context(), h.src, &dto.RestoreRequest{SourceID: h.src.SourceID(), ObjectID: ingested.ObjectID}); err != nil {
		t.Fatalf("RestoreObject: %v", err)
	}

	current, err := h.objects.CurrentVersion(t.Context(), ingested.ObjectID)
	if err != nil {
		t.Fatalf("CurrentVersion: %v", err)
	}
	if current.RestoredFromVersionID == nil {
		t.Fatal("the restore left no provenance")
	}
	if *current.RestoredFromVersionID != ingested.VersionID {
		t.Fatalf("provenance points at %q, want %q", *current.RestoredFromVersionID, ingested.VersionID)
	}
}
