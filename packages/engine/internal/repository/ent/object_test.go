package ent_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

func newSourceForTest(t *testing.T, c sqlitedb.Client) string {
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

func mustCreateBlob(t *testing.T, c sqlitedb.Client, hash string, size int64) string {
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
	// Timestamps are set by the repository, so compare everything else.
	if diff := cmp.Diff(*obj, got,
		cmpopts.IgnoreFields(object.Object{}, "FirstSeenAt", "LastSeenAt"),
	); diff != "" {
		t.Fatalf("round trip changed the object (-want +got):\n%s", diff)
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

// A restored message comes back under a new provider ID. If it did not
// resolve to the original object, every restore would fork the archive.
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

// objects and object_alias each have their own unique index on
// (source_id, external_id), so nothing at the schema level stops the same
// provider identity existing in both. GetByExternalID checks objects first,
// so an alias shadowed that way would silently resolve to the wrong object.
func TestAliasCannotShadowAnExistingObjectIdentity(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	first := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: srcID,
		Kind: object.KindMessage, ExternalID: "m1",
	}
	second := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: srcID,
		Kind: object.KindMessage, ExternalID: "m2",
	}
	for _, o := range []*object.Object{first, second} {
		if err := repo.Create(ctx, o); err != nil {
			t.Fatalf("create %s: %v", o.ExternalID, err)
		}
	}

	// Aliasing m2 onto the first object would make "m2" resolve to whichever
	// table is consulted first.
	if err := repo.AddAlias(ctx, first.ID, srcID, "m2"); err == nil {
		t.Fatal("alias was allowed to shadow an existing object identity")
	}
}

// The foreign_keys pragma is inert without REFERENCES clauses, so this
// asserts the constraints exist rather than that the pragma is set.
func TestDanglingReferencesAreRejected(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)
	srcID := newSourceForTest(t, c)
	repo := entrepo.NewObjectRepository(c)

	if err := repo.Create(ctx, &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: "src_does_not_exist",
		Kind: object.KindMessage, ExternalID: "m1",
	}); err == nil {
		t.Fatal("object accepted a source_id with no matching source")
	}

	obj := &object.Object{
		ID: types.NewID(types.PrefixObject), SourceID: srcID,
		Kind: object.KindMessage, ExternalID: "m1",
	}
	if err := repo.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.CreateVersion(ctx, &object.Version{
		ID: types.NewID(types.PrefixVersion), ObjectID: obj.ID,
		BlobID: "blob_does_not_exist",
	}); err == nil {
		t.Fatal("version accepted a blob_id with no matching blob")
	}

	if err := repo.AddAlias(ctx, "obj_does_not_exist", srcID, "m2"); err == nil {
		t.Fatal("alias accepted an object_id with no matching object")
	}
}
