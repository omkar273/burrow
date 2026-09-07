package ent_test

import (
	"context"
	"path/filepath"
	"testing"

	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
)

// A real SQLite file, not a mock: these tests exist to prove the schema
// and the driver actually work.
func openTestClient(t *testing.T) *entrepo.Client {
	t.Helper()
	dsn := entrepo.FileDSN(filepath.Join(t.TempDir(), "state.db"))
	c, err := entrepo.Open(dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return c
}

func TestOpenAppliesMigrationsAndRoundTripsABlob(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)

	const hash = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	created, err := c.Ent().Blob.Create().
		SetID("blob_01J000000000000000000000").
		SetContentHash(hash).
		SetSizeBytes(1024).
		Save(ctx)
	if err != nil {
		t.Fatalf("create blob: %v", err)
	}

	got, err := c.Ent().Blob.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get blob: %v", err)
	}
	if got.ContentHash != hash {
		t.Fatalf("ContentHash = %q, want %q", got.ContentHash, hash)
	}
	if got.SizeBytes != 1024 {
		t.Fatalf("SizeBytes = %d, want 1024", got.SizeBytes)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt was not defaulted")
	}
}

func TestContentHashIsUnique(t *testing.T) {
	ctx := context.Background()
	c := openTestClient(t)

	const hash = "sha256:aaaa"

	if _, err := c.Ent().Blob.Create().
		SetID("blob_01J000000000000000000001").
		SetContentHash(hash).SetSizeBytes(1).Save(ctx); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Content addressing is what keeps restore from duplicating objects.
	// A duplicate hash row would defeat it.
	if _, err := c.Ent().Blob.Create().
		SetID("blob_01J000000000000000000002").
		SetContentHash(hash).SetSizeBytes(1).Save(ctx); err == nil {
		t.Fatal("duplicate content_hash was accepted")
	}
}

func TestMigrateIsIdempotentAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.db")

	first, err := entrepo.Open(entrepo.FileDSN(path))
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := first.Migrate(ctx); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if _, err := first.Ent().Blob.Create().
		SetID("blob_01J000000000000000000003").
		SetContentHash("sha256:bbbb").SetSizeBytes(7).Save(ctx); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	second, err := entrepo.Open(entrepo.FileDSN(path))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer second.Close()
	if err := second.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	got, err := second.Ent().Blob.Get(ctx, "blob_01J000000000000000000003")
	if err != nil {
		t.Fatalf("row did not survive reopen and re-migrate: %v", err)
	}
	if got.SizeBytes != 7 {
		t.Fatalf("SizeBytes = %d, want 7", got.SizeBytes)
	}
}
