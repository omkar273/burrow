package ent_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/omkar273/burrow/internal/domain/object"
	"github.com/omkar273/burrow/internal/domain/source"
	ierr "github.com/omkar273/burrow/internal/errors"
	entrepo "github.com/omkar273/burrow/internal/repository/ent"
	"github.com/omkar273/burrow/internal/types"
)

func newSourceForTest(t *testing.T, c *entrepo.Client) string {
	t.Helper()
	id := types.NewID(types.PrefixSource)
	err := entrepo.NewSourceRepository(c).Create(context.Background(), source.Source{
		ID: id, Kind: source.KindGmail,
		AccountEmail: "test@example.com", Status: source.StatusActive,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	return id
}

func mustCreateBlob(t *testing.T, c *entrepo.Client, hash string, size int64) string {
	t.Helper()
	id := types.NewID(types.PrefixBlob)
	if _, err := c.Ent().Blob.Create().
		SetID(id).SetContentHash(hash).SetSizeBytes(size).
		Save(context.Background()); err != nil {
		t.Fatalf("create blob: %v", err)
	}
	return id
}

func TestObjectRoundTrip(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	obj := &object.Object{
		ID:         types.NewID(types.PrefixObject),
		SourceID:   srcID,
		Kind:       object.KindMessage,
		ExternalID: "18f0a1b2c3d4e5f6",
	}
	if err := repo.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, obj.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ExternalID != obj.ExternalID {
		t.Fatalf("ExternalID = %q, want %q", got.ExternalID, obj.ExternalID)
	}
	if got.Kind != object.KindMessage {
		t.Fatalf("Kind = %q, want %q", got.Kind, object.KindMessage)
	}
}

func TestGetMissingObjectIsErrNotFound(t *testing.T) {
	c := openTestClient(t)
	repo := entrepo.NewObjectRepository(c)

	_, err := repo.Get(context.Background(), "obj_01J00000000000000000000X")
	if !stderrors.Is(err, ierr.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// A restored message arrives back from Gmail under a new provider ID.
// Resolving it must return the ORIGINAL object, not mint a second one —
// otherwise every restore forks the archive.
func TestAliasResolvesToTheOriginalObject(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	obj := &object.Object{
		ID:         types.NewID(types.PrefixObject),
		SourceID:   srcID,
		Kind:       object.KindMessage,
		ExternalID: "original-provider-id",
	}
	if err := repo.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.AddAlias(ctx, obj.ID, srcID, "restored-provider-id"); err != nil {
		t.Fatalf("add alias: %v", err)
	}

	viaOriginal, err := repo.GetByExternalID(ctx, srcID, "original-provider-id")
	if err != nil {
		t.Fatalf("lookup by original id: %v", err)
	}
	viaAlias, err := repo.GetByExternalID(ctx, srcID, "restored-provider-id")
	if err != nil {
		t.Fatalf("lookup by alias: %v", err)
	}
	if viaAlias.ID != viaOriginal.ID {
		t.Fatalf("alias resolved to %q, want %q", viaAlias.ID, viaOriginal.ID)
	}
}

func TestCurrentVersionReturnsTheLatest(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	obj := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: srcID,
		Kind: object.KindMessage, ExternalID: "m1",
	}
	if err := repo.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}

	blobID := mustCreateBlob(t, c, "sha256:cccc", 10)
	v := &object.Version{
		ID: types.NewID(types.PrefixVersion), ObjectID: obj.ID, BlobID: blobID,
	}
	if err := repo.CreateVersion(ctx, v); err != nil {
		t.Fatalf("create version: %v", err)
	}

	got, err := repo.CurrentVersion(ctx, obj.ID)
	if err != nil {
		t.Fatalf("current version: %v", err)
	}
	if got.ID != v.ID {
		t.Fatalf("CurrentVersion = %q, want %q", got.ID, v.ID)
	}
	if got.BlobID != blobID {
		t.Fatalf("BlobID = %q, want %q", got.BlobID, blobID)
	}
}

// Ingest resolves a content hash back to the object that owns it. This is
// what lets restored content attach as an alias instead of forking.
func TestOwnerOfBlobFindsTheObject(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	obj := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: srcID,
		Kind: object.KindMessage, ExternalID: "m1",
	}
	if err := repo.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}
	blobID := mustCreateBlob(t, c, "sha256:dddd", 3)
	if err := repo.CreateVersion(ctx, &object.Version{
		ID: types.NewID(types.PrefixVersion), ObjectID: obj.ID, BlobID: blobID,
	}); err != nil {
		t.Fatalf("create version: %v", err)
	}

	owner, err := repo.OwnerOfBlob(ctx, blobID)
	if err != nil {
		t.Fatalf("OwnerOfBlob: %v", err)
	}
	if owner.ID != obj.ID {
		t.Fatalf("OwnerOfBlob = %q, want %q", owner.ID, obj.ID)
	}
}
