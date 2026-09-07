// Package engine is Burrow's embeddable core: it copies objects from a
// source onto storage you control and restores them.
//
// This file is the entire public surface. Everything under internal/ is
// implementation and is unreachable from outside this package — which is
// what lets the internals change without breaking consumers. No internal
// type appears in a signature here; callers pass and receive plain data.
//
// apps/agent is the reference consumer.
package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/storage/localfs"
	"github.com/omkar273/burrow/packages/engine/internal/validator"

	"github.com/gofrs/flock"
)

// Plain strings, so nothing from internal/ crosses the boundary.
type Config struct {
	// Empty means the BURROW_PROFILE environment variable, then "default".
	Profile string `validate:"omitempty,max=64,profilename"`
	// Empty means os.UserHomeDir.
	Home string
}

type Paths struct {
	Profile   string
	Root      string
	StateDB   string
	ObjectDir string
}

// Engine is not safe for concurrent use until the job engine lands.
type Engine struct {
	profile config.Profile
	lock    *flock.Flock
	state   *entrepo.Client
	store   storage.Store

	objects object.Repository
	blobs   blob.Repository
	sources source.Repository
}

// The caller must Close the result.
func Open(ctx context.Context, cfg Config) (*Engine, error) {
	if err := validator.ValidateRequest(cfg); err != nil {
		return nil, err
	}

	home := cfg.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, ierr.Wrap(err, "resolving home directory").Mark(ierr.ErrInternal)
		}
		home = h
	}

	profile, err := config.ResolveProfile(cfg.Profile, os.Getenv, home)
	if err != nil {
		return nil, err
	}
	if err := profile.Ensure(); err != nil {
		return nil, err
	}

	// An OS advisory lock, not a pidfile: the kernel drops it when the
	// process dies, so a crashed burrow leaves nothing stale to clear.
	lock := flock.New(filepath.Join(profile.Root, ".lock"))
	held, err := lock.TryLock()
	if err != nil {
		return nil, ierr.Wrap(err, "locking profile "+profile.Name).Mark(ierr.ErrInternal)
	}
	if !held {
		return nil, ierr.New("profile " + profile.Name + " is already open").
			WithHint("another burrow is running against this profile; stop it or use --profile").
			Mark(ierr.ErrProfileLocked)
	}

	state, err := entrepo.Open(entrepo.DSN(profile))
	if err != nil {
		_ = lock.Unlock()
		return nil, err
	}
	if err := state.Migrate(ctx); err != nil {
		_ = state.Close()
		_ = lock.Unlock()
		return nil, err
	}

	store, err := localfs.New(profile.ObjectDir)
	if err != nil {
		_ = state.Close()
		_ = lock.Unlock()
		return nil, err
	}

	return &Engine{
		profile: profile,
		lock:    lock,
		state:   state,
		store:   store,
		objects: entrepo.NewObjectRepository(state),
		blobs:   entrepo.NewBlobRepository(state),
		sources: entrepo.NewSourceRepository(state),
	}, nil
}

func (e *Engine) Paths() Paths {
	return Paths{
		Profile:   e.profile.Name,
		Root:      e.profile.Root,
		StateDB:   e.profile.StateDB,
		ObjectDir: e.profile.ObjectDir,
	}
}

func (e *Engine) Close() error {
	if e.state == nil {
		return nil
	}
	err := e.state.Close()
	if e.lock != nil {
		if unlockErr := e.lock.Unlock(); err == nil {
			err = unlockErr
		}
	}
	return err
}
