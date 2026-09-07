package engine_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/omkar273/burrow/packages/engine"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

func TestOpenCreatesAndMigratesAnArchive(t *testing.T) {
	home := t.TempDir()

	e, err := engine.Open(context.Background(), engine.Config{Home: home})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer e.Close()

	p := e.Paths()
	if p.Profile != "default" {
		t.Fatalf("Profile = %q, want default", p.Profile)
	}
	if want := filepath.Join(home, ".burrow", "profiles", "default"); p.Root != want {
		t.Fatalf("Root = %q, want %q", p.Root, want)
	}
	for _, path := range []string{p.Root, p.ObjectDir, p.StateDB} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Open did not create %s: %v", path, err)
		}
	}
}

func TestOpenIsIdempotentAcrossRuns(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	first, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("reopening an existing archive: %v", err)
	}
	defer second.Close()
}

// Two burrowd processes on one profile would race on blob writes, job
// claiming, and Gmail quota. SQLite's busy_timeout protects only the
// database.
func TestSecondOpenOfTheSameProfileIsRefused(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	first, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer first.Close()

	_, err = engine.Open(ctx, engine.Config{Home: home})
	if !errors.Is(err, ierr.ErrProfileLocked) {
		t.Fatalf("second Open err = %v, want ErrProfileLocked", err)
	}
}

func TestProfileIsReusableAfterClose(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	first, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("Open after Close: %v", err)
	}
	defer second.Close()
}

// A different profile is a different archive and must not be blocked by
// the first one's lock.
func TestAnotherProfileIsNotBlocked(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	work, err := engine.Open(ctx, engine.Config{Profile: "work", Home: home})
	if err != nil {
		t.Fatalf("open work: %v", err)
	}
	defer work.Close()

	personal, err := engine.Open(ctx, engine.Config{Profile: "personal", Home: home})
	if err != nil {
		t.Fatalf("open personal while work is held: %v", err)
	}
	defer personal.Close()
}
