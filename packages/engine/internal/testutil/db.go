package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

// OpenMigratedClient returns a migrated, empty state database in a temp
// directory, closed when the test ends.
func OpenMigratedClient(t *testing.T) *sqlitedb.Client {
	t.Helper()
	c, err := sqlitedb.Open(sqlitedb.FileDSN(filepath.Join(t.TempDir(), "state.db")))
	if err != nil {
		t.Fatalf("open state db: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return c
}

// SeedSource inserts one connected source and returns its ID.
func SeedSource(t *testing.T, c *sqlitedb.Client, email string) string {
	t.Helper()
	id := types.NewID(types.PrefixSource)
	err := entrepo.NewSourceRepository(c).Create(context.Background(), source.Source{
		ID: id, Kind: source.KindGmail, AccountEmail: email, Status: source.StatusActive,
	})
	if err != nil {
		t.Fatalf("seed source: %v", err)
	}
	return id
}
