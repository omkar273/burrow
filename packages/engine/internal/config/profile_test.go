package config_test

import (
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

func noEnv(string) string { return "" }

func TestDefaultsToTheDefaultProfile(t *testing.T) {
	p, err := config.ResolveProfile("", noEnv, "/home/u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "default" {
		t.Fatalf("Name = %q, want default", p.Name)
	}
	if want := "/home/u/.burrow/profiles/default"; p.Root != want {
		t.Fatalf("Root = %q, want %q", p.Root, want)
	}
	if want := filepath.Join(p.Root, "state.db"); p.StateDB != want {
		t.Fatalf("StateDB = %q, want %q", p.StateDB, want)
	}
	if want := filepath.Join(p.Root, "objects"); p.ObjectDir != want {
		t.Fatalf("ObjectDir = %q, want %q", p.ObjectDir, want)
	}
}

func TestEnvironmentSelectsTheProfile(t *testing.T) {
	env := func(k string) string {
		if k == "BURROW_PROFILE" {
			return "work"
		}
		return ""
	}
	p, err := config.ResolveProfile("", env, "/home/u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "work" {
		t.Fatalf("Name = %q, want work", p.Name)
	}
}

func TestFlagBeatsEnvironment(t *testing.T) {
	env := func(k string) string {
		if k == "BURROW_PROFILE" {
			return "work"
		}
		return ""
	}
	p, err := config.ResolveProfile("personal", env, "/home/u")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "personal" {
		t.Fatalf("Name = %q, want personal", p.Name)
	}
}

func TestRejectsNamesThatEscapeTheProfileDirectory(t *testing.T) {
	for _, name := range []string{"../etc", "a/b", "..", "a\\b", "."} {
		if _, err := config.ResolveProfile(name, noEnv, "/home/u"); err == nil {
			t.Fatalf("profile name %q was accepted", name)
		} else if !stderrors.Is(err, ierr.ErrValidation) {
			t.Fatalf("profile name %q rejected with %v, want ErrValidation", name, err)
		}
	}
}

func TestEnsureCreatesDirectoriesPrivateToTheUser(t *testing.T) {
	home := t.TempDir()
	p, err := config.ResolveProfile("default", noEnv, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := p.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	for _, dir := range []string{p.Root, p.ObjectDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if got := info.Mode().Perm(); got != fs.FileMode(0o700) {
			t.Fatalf("%s mode = %o, want 700", dir, got)
		}
	}
}

func TestEnsureIsIdempotent(t *testing.T) {
	home := t.TempDir()
	p, _ := config.ResolveProfile("default", noEnv, home)
	if err := p.Ensure(); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	if err := p.Ensure(); err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
}
