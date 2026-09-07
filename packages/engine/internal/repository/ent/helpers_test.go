package ent_test

import (
	"context"
	"path/filepath"
	"testing"

	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
)

// A real SQLite file, not a mock: these tests exist to prove the schema and
// the driver actually work.
func openTestClient(t *testing.T) *sqlitedb.Client {
	t.Helper()
	c, err := sqlitedb.Open(sqlitedb.FileDSN(filepath.Join(t.TempDir(), "state.db")))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return c
}
