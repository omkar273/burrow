package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/omkar273/burrow/packages/engine"
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
