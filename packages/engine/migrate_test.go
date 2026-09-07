package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/omkar273/burrow/packages/engine"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
)

func TestMigrationPlanListsPendingWithoutApplying(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if !strings.Contains(plan, "CREATE TABLE") {
		t.Fatalf("fresh archive produced no reviewable SQL:\n%s", plan)
	}

	// A dry run must not have applied anything.
	again, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("second MigrationPlan: %v", err)
	}
	if again != plan {
		t.Fatal("a dry run changed what the next dry run would do")
	}
}

func TestMigrateAppliesThenReportsNothingPending(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan after Migrate: %v", err)
	}
	if plan != "" {
		t.Fatalf("statements still pending after Migrate:\n%s", plan)
	}

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

// Migrating writes to the archive, so it must respect the profile lock.
func TestMigrateRefusesWhileAnotherProcessHoldsTheProfile(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	held, err := engine.Open(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer held.Close()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err == nil {
		t.Fatal("Migrate ran while another process held the profile")
	}
}

// A dry run must create nothing at all — not the profile directory, not the
// database file, not the lock. Creating the database it then diffs against is
// what made every dry run on a fresh machine emit the entire schema.
func TestMigrationPlanTouchesNothingOnDisk(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if !strings.Contains(plan, "CREATE TABLE") {
		t.Fatalf("no archive yet, so the plan should be the full schema:\n%s", plan)
	}

	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("dry run created %v in an untouched home", names)
	}
}

// Against an existing archive the plan is the delta, not the whole schema.
func TestMigrationPlanIsIncrementalAgainstAnExistingArchive(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if plan != "" {
		t.Fatalf("migrated archive still reports pending statements:\n%s", plan)
	}
}

func TestMigrationPlanLeavesTheDatabaseUntouched(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if _, err := engine.MigrationPlan(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}

	tables, err := engine.TableNames(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("TableNames: %v", err)
	}
	if len(tables) != 0 {
		t.Fatalf("dry run created %v", tables)
	}
}

func TestSchemaVersionAdvancesWithMigrations(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	before, err := engine.SchemaVersion(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if before != 0 {
		t.Fatalf("fresh archive at version %d, want 0", before)
	}

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	after, err := engine.SchemaVersion(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("SchemaVersion after Migrate: %v", err)
	}
	if after == 0 {
		t.Fatal("schema version still 0 after Migrate")
	}
}

// An archive written by a newer Burrow must be refused, not half-read.
func TestArchiveFromANewerBuildIsRefused(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := engine.SetSchemaVersionForTest(ctx, engine.Config{Home: home}, 9999); err != nil {
		t.Fatalf("bumping version: %v", err)
	}

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err == nil {
		t.Fatal("an archive from a newer build was accepted")
	}
}

// openArchiveDB opens an already-migrated archive directly, so a test can
// damage it the way a person with sqlite3 could.
func openArchiveDB(t *testing.T, home string) *sqlitedb.Client {
	t.Helper()
	c, err := sqlitedb.Open(sqlitedb.FileDSN(
		filepath.Join(home, ".burrow", "profiles", "default", "state.db")))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// A table missing from the database must show up in the plan as exactly that
// table, not as the whole schema.
func TestPlanProposesRecreatingADroppedTable(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	func() {
		c := openArchiveDB(t, home)
		if _, err := c.DB().ExecContext(ctx, `DROP TABLE object_alias;`); err != nil {
			t.Fatalf("dropping table: %v", err)
		}
	}()

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if !strings.Contains(plan, "object_alias") {
		t.Fatalf("plan does not mention the dropped table:\n%s", plan)
	}
	for _, untouched := range []string{"CREATE TABLE `blobs`", "CREATE TABLE `sources`"} {
		if strings.Contains(plan, untouched) {
			t.Fatalf("plan rebuilds tables that still exist — not a delta:\n%s", plan)
		}
	}

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("repairing Migrate: %v", err)
	}
	after, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan after repair: %v", err)
	}
	if after != "" {
		t.Fatalf("archive still drifted after Migrate:\n%s", after)
	}
}

// A column missing from a table must show up as an ALTER on that one table.
func TestPlanProposesRestoringADroppedColumn(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	func() {
		c := openArchiveDB(t, home)
		if _, err := c.DB().ExecContext(ctx,
			"ALTER TABLE `objects` DROP COLUMN `deleted_at_source`;"); err != nil {
			t.Fatalf("dropping column: %v", err)
		}
	}()

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if !strings.Contains(plan, "deleted_at_source") {
		t.Fatalf("plan does not mention the dropped column:\n%s", plan)
	}
	if strings.Contains(plan, "CREATE TABLE `blobs`") {
		t.Fatalf("a missing column made the plan rebuild unrelated tables:\n%s", plan)
	}

	if err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("repairing Migrate: %v", err)
	}
	after, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan after repair: %v", err)
	}
	if after != "" {
		t.Fatalf("archive still drifted after Migrate:\n%s", after)
	}
}
