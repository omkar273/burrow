package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The library test covers engine.MigrationPlan; this covers the command,
// which is where a second side-effecting call slipped in.
func TestDryRunCreatesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer devNull.Close()

	if err := run([]string{"--dry-run"}, devNull); err != nil {
		t.Fatalf("run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".burrow")); !os.IsNotExist(err) {
		t.Fatal("dry run created ~/.burrow")
	}
}
