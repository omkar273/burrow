package localfs_test

import (
	"bytes"
	stderrors "errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/storage/localfs"
)

func TestPutThenGetReturnsTheSameBytes(t *testing.T) {
	s, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want := []byte("From: a@example.com\r\n\r\nhello\r\n")
	key := storage.KeyForHash("sha256:abcd1234")

	if err := s.Put(t.Context(), key, bytes.NewReader(want), int64(len(want))); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, err := s.Get(t.Context(), key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("round-trip changed the bytes:\n got %q\nwant %q", got, want)
	}
}

func TestGetMissingKeyIsErrBlobMissing(t *testing.T) {
	s, _ := localfs.New(t.TempDir())
	_, err := s.Get(t.Context(), storage.KeyForHash("sha256:0000dead"))
	if !stderrors.Is(err, ierr.ErrBlobMissing) {
		t.Fatalf("err = %v, want ErrBlobMissing", err)
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// Temp file, fsync, then rename: a reader sees nothing or the whole object.
func TestPartialWritesAreNeverVisibleAtTheFinalKey(t *testing.T) {
	root := t.TempDir()
	s, _ := localfs.New(root)
	key := storage.KeyForHash("sha256:beefcafe")

	failing := io.MultiReader(
		strings.NewReader("first half"),
		errReader{stderrors.New("network died mid-stream")},
	)
	if err := s.Put(t.Context(), key, failing, 100); err == nil {
		t.Fatal("Put succeeded despite a failing reader")
	}

	if _, err := os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Fatal("a partial object is visible at the final key")
	}
}

func TestFailedWritesLeaveNoTempFiles(t *testing.T) {
	root := t.TempDir()
	s, _ := localfs.New(root)
	key := storage.KeyForHash("sha256:beefcafe")

	_ = s.Put(t.Context(), key, errReader{stderrors.New("boom")}, 100)

	var found []string
	for k, err := range s.List(t.Context(), "objects") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		found = append(found, k)
	}
	if len(found) != 0 {
		t.Fatalf("List returned %v after a failed write, want nothing", found)
	}
}
