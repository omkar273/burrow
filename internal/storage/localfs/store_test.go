package localfs_test

import (
	"bytes"
	stderrors "errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ierr "github.com/omkar273/burrow/internal/errors"
	"github.com/omkar273/burrow/internal/storage"
	"github.com/omkar273/burrow/internal/storage/localfs"
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

func TestKeyFansOutByHashPrefix(t *testing.T) {
	got := storage.KeyForHash("sha256:abcd1234")
	want := "objects/ab/cd/abcd1234"
	if got != want {
		t.Fatalf("KeyForHash = %q, want %q", got, want)
	}
}

func TestPutIsIdempotentForIdenticalContent(t *testing.T) {
	s, _ := localfs.New(t.TempDir())
	key := storage.KeyForHash("sha256:ffff0000")
	body := []byte("same bytes")

	for i := 0; i < 2; i++ {
		if err := s.Put(t.Context(), key, bytes.NewReader(body), int64(len(body))); err != nil {
			t.Fatalf("Put #%d: %v", i+1, err)
		}
	}
	st, err := s.Stat(t.Context(), key)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.Size != int64(len(body)) {
		t.Fatalf("Size = %d, want %d", st.Size, len(body))
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

// A crash mid-write must never leave a partial file at the real key.
// Writes go to a temp file and are renamed only after fsync, so a reader
// sees either nothing or the complete object.
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

// The temp file from a failed write must not linger and must never be
// mistaken for a stored object.
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
