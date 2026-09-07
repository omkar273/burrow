package service_test

import (
	stderrors "errors"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

const rawMessage = "From: a@example.com\r\nSubject: hello\r\n\r\nbody\r\n"

func TestIngestStoresBlobAndCreatesObject(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	got, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("IngestOne: %v", err)
	}
	if got.ObjectID == "" || got.VersionID == "" || got.ContentHash == "" {
		t.Fatalf("incomplete result: %+v", got)
	}
	if got.Deduplicated {
		t.Fatal("first ingest reported as deduplicated")
	}

	ok, err := h.store.Exists(t.Context(), storage.KeyForHash(got.ContentHash))
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("blob was not written to storage")
	}
}

// At-least-once ingestion makes re-fetching the same message routine, not
// exceptional.
func TestIngestingTheSameMessageTwiceIsIdempotent(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	first, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}

	if second.ObjectID != first.ObjectID || second.VersionID != first.VersionID {
		t.Fatalf("second ingest minted new identity: %+v vs %+v", second, first)
	}
	if !second.Deduplicated {
		t.Fatal("second ingest not reported as deduplicated")
	}
}

// Identical bytes under a different provider id are a DIFFERENT object. Only
// restore knows lineage, and only restore records an alias — ingest must not
// infer one, or two distinct messages get merged and one is lost.
func TestIdenticalBytesUnderANewIDBecomeTheirOwnObject(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))
	h.src.AddMessage("m2", []byte(rawMessage))

	first, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest m1: %v", err)
	}
	second, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m2"})
	if err != nil {
		t.Fatalf("ingest m2: %v", err)
	}

	if second.ObjectID == first.ObjectID {
		t.Fatal("two distinct messages were merged into one object")
	}
	if second.ContentHash != first.ContentHash {
		t.Fatal("identical bytes produced different hashes")
	}
	if !second.BlobReused {
		t.Fatal("the existing blob row should have been reused")
	}
}

// Blob before row. A row pointing at bytes that do not exist is corruption;
// bytes with no row is an inert orphan, collectable because keys are
// content-addressed.
func TestNoMetadataIsCommittedWhenTheBlobWriteFails(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))
	h.store.PutErr = stderrors.New("disk full")

	if _, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"}); err == nil {
		t.Fatal("IngestOne succeeded despite a failed blob write")
	}

	_, err := h.objects.GetByExternalID(t.Context(), h.src.SourceID(), "m1")
	if !stderrors.Is(err, ierr.ErrNotFound) {
		t.Fatalf("an object row survived a failed blob write: err = %v", err)
	}
}

func TestIngestSurfacesFetchFailures(t *testing.T) {
	h := newHarness(t)
	h.src.FetchErr = ierr.New("provider is down").Mark(ierr.ErrSourceUnavailable)

	_, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if !stderrors.Is(err, ierr.ErrSourceUnavailable) {
		t.Fatalf("err = %v, want ErrSourceUnavailable", err)
	}
}

// A restored message returns under a new provider id. Restore records the
// alias, so a later ingest of that id must resolve to the original object.
func TestIngestFollowsAnAliasRecordedByRestore(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	original, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if err := h.objects.AddAlias(t.Context(), original.ObjectID, h.src.SourceID(), "restored-1"); err != nil {
		t.Fatalf("AddAlias: %v", err)
	}
	h.src.AddMessage("restored-1", []byte(rawMessage))

	after, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "restored-1"})
	if err != nil {
		t.Fatalf("ingest restored: %v", err)
	}
	if after.ObjectID != original.ObjectID {
		t.Fatalf("aliased id resolved to %q, want %q", after.ObjectID, original.ObjectID)
	}
	if !after.Deduplicated {
		t.Fatal("an aliased id should report as already held")
	}
}

// The object row and its version must commit together. A crash between them
// would leave an object with no content — a row pointing at nothing, which is
// the corruption the write ordering exists to prevent.
func TestObjectAndVersionCommitTogether(t *testing.T) {
	h := newHarness(t)
	h.src.AddMessage("m1", []byte(rawMessage))

	res, err := h.ingest.IngestOne(t.Context(), h.src, source.ObjectRef{ExternalID: "m1"})
	if err != nil {
		t.Fatalf("IngestOne: %v", err)
	}

	// Every ingested object resolves to a current version.
	v, err := h.objects.CurrentVersion(t.Context(), res.ObjectID)
	if err != nil {
		t.Fatalf("ingested object has no version: %v", err)
	}
	if v.ID != res.VersionID {
		t.Fatalf("CurrentVersion = %q, want %q", v.ID, res.VersionID)
	}

	// And the version's blob is really in storage.
	b, err := h.blobs.Get(t.Context(), v.BlobID)
	if err != nil {
		t.Fatalf("version references a missing blob row: %v", err)
	}
	ok, err := h.store.Exists(t.Context(), storage.KeyForHash(b.ContentHash))
	if err != nil || !ok {
		t.Fatal("version references bytes that are not in storage")
	}
}
