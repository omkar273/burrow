// Package localfs stores blobs as files on a local disk.
package localfs

import (
	"context"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

// tempPrefix marks in-flight writes. Files with this prefix are never
// reported by List: they are not objects until renamed.
const tempPrefix = ".tmp-"

type store struct{ root string }

func New(root string) (storage.Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, ierr.Wrap(err, "creating object root").Mark(ierr.ErrInternal)
	}
	return &store{root: root}, nil
}

// NewFromProfile stores blobs under a profile's object directory.
func NewFromProfile(p config.Profile) (storage.Store, error) { return New(p.ObjectDir) }

func (s *store) path(key string) string {
	return filepath.Join(s.root, filepath.FromSlash(key))
}

// A crash mid-write leaves a temp file, which is inert. The
// alternative — writing directly to the final key — can leave a truncated
// object that later reads as real. Because keys are content-addressed, an
// orphaned temp file is harmless and collectable.
func (s *store) Put(_ context.Context, key string, r io.Reader, _ int64) error {
	final := s.path(key)
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		return ierr.Wrap(err, "creating object directory").Mark(ierr.ErrInternal)
	}

	tmp, err := os.CreateTemp(filepath.Dir(final), tempPrefix+"*")
	if err != nil {
		return ierr.Wrap(err, "creating temp object").Mark(ierr.ErrInternal)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return ierr.Wrap(err, "writing object body").Mark(ierr.ErrInternal)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return ierr.Wrap(err, "syncing object").Mark(ierr.ErrInternal)
	}
	if err := tmp.Close(); err != nil {
		return ierr.Wrap(err, "closing object").Mark(ierr.ErrInternal)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return ierr.Wrap(err, "setting object mode").Mark(ierr.ErrInternal)
	}
	if err := os.Rename(tmpName, final); err != nil {
		return ierr.Wrap(err, "publishing object").Mark(ierr.ErrInternal)
	}

	// File fsync persists bytes, not the rename. Skip this and a crash
	// can leave a catalog row with no blob.
	return syncDir(filepath.Dir(final))
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return ierr.Wrap(err, "opening object directory to sync").Mark(ierr.ErrInternal)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return ierr.Wrap(err, "syncing object directory").Mark(ierr.ErrInternal)
	}
	return nil
}

func (s *store) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ierr.New("no object at key " + key).Mark(ierr.ErrBlobMissing)
		}
		return nil, ierr.Wrap(err, "opening object").Mark(ierr.ErrInternal)
	}
	return f, nil
}

func (s *store) Stat(_ context.Context, key string) (storage.Stat, error) {
	info, err := os.Stat(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return storage.Stat{}, ierr.New("no object at key " + key).Mark(ierr.ErrBlobMissing)
		}
		return storage.Stat{}, ierr.Wrap(err, "stat object").Mark(ierr.ErrInternal)
	}
	return storage.Stat{Size: info.Size(), ModifiedAt: info.ModTime()}, nil
}

func (s *store) Exists(_ context.Context, key string) (bool, error) {
	if _, err := os.Stat(s.path(key)); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ierr.Wrap(err, "stat object").Mark(ierr.ErrInternal)
	}
	return true, nil
}

func (s *store) Delete(_ context.Context, key string) error {
	if err := os.Remove(s.path(key)); err != nil && !os.IsNotExist(err) {
		return ierr.Wrap(err, "removing object").Mark(ierr.ErrInternal)
	}
	return nil
}

func (s *store) List(_ context.Context, prefix string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		root := s.path(prefix)
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				yield("", ierr.Wrap(err, "walking objects").Mark(ierr.ErrInternal))
				return filepath.SkipAll
			}
			if info.IsDir() || strings.HasPrefix(info.Name(), tempPrefix) {
				return nil
			}
			rel, relErr := filepath.Rel(s.root, p)
			if relErr != nil {
				return nil
			}
			if !yield(filepath.ToSlash(rel), nil) {
				return filepath.SkipAll
			}
			return nil
		})
	}
}

func (s *store) Capabilities() storage.Caps {
	return storage.Caps{Multipart: false, RangeReads: true, AtomicRename: true}
}
