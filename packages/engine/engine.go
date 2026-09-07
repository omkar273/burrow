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

	"github.com/omkar273/burrow/packages/engine/internal/config"
	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
	"github.com/omkar273/burrow/packages/engine/internal/storage/localfs"
)

// Config selects which archive to open.
//
// Profile and Home are plain strings rather than resolved types so that
// nothing from internal/ crosses the boundary.
type Config struct {
	// Profile names the archive. Empty means the BURROW_PROFILE
	// environment variable, then "default".
	Profile string
	// Home overrides the user's home directory. Empty means os.UserHomeDir.
	Home string
}

// Paths reports where an opened archive lives on disk.
type Paths struct {
	Profile   string
	Root      string
	StateDB   string
	ObjectDir string
}

// Engine is an opened archive. It is not safe for concurrent use by
// multiple goroutines until the job engine lands.
type Engine struct {
	profile config.Profile
	state   *entrepo.Client
	store   storage.Store

	objects object.Repository
	blobs   blob.Repository
	sources source.Repository
}

// Open resolves the profile, applies pending migrations, and prepares
// storage. The caller must Close the result.
func Open(ctx context.Context, cfg Config) (*Engine, error) {
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

	state, err := entrepo.Open(entrepo.DSN(profile))
	if err != nil {
		return nil, err
	}
	if err := state.Migrate(ctx); err != nil {
		_ = state.Close()
		return nil, err
	}

	store, err := localfs.New(profile.ObjectDir)
	if err != nil {
		_ = state.Close()
		return nil, err
	}

	return &Engine{
		profile: profile,
		state:   state,
		store:   store,
		objects: entrepo.NewObjectRepository(state),
		blobs:   entrepo.NewBlobRepository(state),
		sources: entrepo.NewSourceRepository(state),
	}, nil
}

// Paths reports where this archive lives.
func (e *Engine) Paths() Paths {
	return Paths{
		Profile:   e.profile.Name,
		Root:      e.profile.Root,
		StateDB:   e.profile.StateDB,
		ObjectDir: e.profile.ObjectDir,
	}
}

// Close releases the archive's resources.
func (e *Engine) Close() error {
	if e.state == nil {
		return nil
	}
	return e.state.Close()
}
