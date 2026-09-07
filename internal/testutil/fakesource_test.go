package testutil_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/omkar273/burrow/internal/domain/source"
	"github.com/omkar273/burrow/internal/storage"
	"github.com/omkar273/burrow/internal/testutil"
)

// Compile-time proof that the fakes satisfy the real ports.
var (
	_ source.Connector = (*testutil.FakeSource)(nil)
	_ source.Restorer  = (*testutil.FakeSource)(nil)
	_ storage.Store    = (*testutil.FakeStore)(nil)
)

func TestFakeSourceRestoreMintsANewExternalID(t *testing.T) {
	ctx := context.Background()
	f := testutil.NewFakeSource("src_test")
	body := []byte("From: a@example.com\r\n\r\nhi\r\n")
	f.AddMessage("original", body)

	newID, err := f.Restore(ctx, bytes.NewReader(body), source.RestoreOpts{})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if newID == "original" {
		t.Fatal("Restore reused the original external id; the duplicate hazard is not modelled")
	}

	rc, _, err := f.Fetch(ctx, source.ObjectRef{ExternalID: newID})
	if err != nil {
		t.Fatalf("Fetch restored: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if !bytes.Equal(got, body) {
		t.Fatal("restored bytes differ from the bytes handed to Restore")
	}
}
