// Package testutil holds in-memory fakes so tests run with no
// credentials and no external services.
package testutil

import (
	"bytes"
	"context"
	"io"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

type FakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte

	// PutErr, when set, makes Put fail. Use it to exercise the
	// blob-before-row write ordering.
	PutErr error
}

func NewFakeStore() *FakeStore {
	return &FakeStore{objects: map[string][]byte{}}
}

// Corrupt replaces an object's bytes without changing its key, simulating
// silent storage corruption.
func (f *FakeStore) Corrupt(key string, replacement []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = replacement
}

func (f *FakeStore) Put(_ context.Context, key string, r io.Reader, _ int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PutErr != nil {
		return f.PutErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.objects[key] = b
	return nil
}

func (f *FakeStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objects[key]
	if !ok {
		return nil, ierr.New("no object at key " + key).Mark(ierr.ErrBlobMissing)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (f *FakeStore) Stat(_ context.Context, key string) (storage.Stat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objects[key]
	if !ok {
		return storage.Stat{}, ierr.New("no object at key " + key).Mark(ierr.ErrBlobMissing)
	}
	return storage.Stat{Size: int64(len(b)), ModifiedAt: time.Now().UTC()}, nil
}

func (f *FakeStore) Exists(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok, nil
}

func (f *FakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	return nil
}

func (f *FakeStore) List(_ context.Context, prefix string) iter.Seq2[string, error] {
	f.mu.Lock()
	keys := slices.Sorted(maps.Keys(f.objects))
	f.mu.Unlock()
	return func(yield func(string, error) bool) {
		for _, k := range keys {
			if strings.HasPrefix(k, prefix) && !yield(k, nil) {
				return
			}
		}
	}
}

func (f *FakeStore) Capabilities() storage.Caps {
	return storage.Caps{RangeReads: true, AtomicRename: true}
}
