package engine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/omkar273/burrow/packages/engine"
)

func TestMigrationPlanListsPendingWithoutApplying(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan: %v", err)
	}
	if len(plan) == 0 {
		t.Fatal("fresh archive reported no pending migrations")
	}
	for _, m := range plan {
		if m.Applied {
			t.Fatalf("%s reported as applied on a fresh archive", m.Name)
		}
		if !strings.Contains(m.SQL, "CREATE TABLE") {
			t.Fatalf("%s carries no SQL to review", m.Name)
		}
	}

	// A dry run must not have applied anything.
	again, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("second MigrationPlan: %v", err)
	}
	if len(again) != len(plan) {
		t.Fatalf("dry run changed pending count: %d then %d", len(plan), len(again))
	}
}

func TestMigrateAppliesThenReportsNothingPending(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	applied, err := engine.Migrate(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("Migrate applied nothing on a fresh archive")
	}

	plan, err := engine.MigrationPlan(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("MigrationPlan after Migrate: %v", err)
	}
	if len(plan) != 0 {
		t.Fatalf("%d migrations still pending after Migrate", len(plan))
	}

	second, err := engine.Migrate(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("re-running Migrate applied %v", second)
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

	if _, err := engine.Migrate(ctx, engine.Config{Home: home}); err == nil {
		t.Fatal("Migrate ran while another process held the profile")
	}
}

// A dry run must not create so much as the bookkeeping table.
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

	applied, err := engine.Migrate(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	after, err := engine.SchemaVersion(ctx, engine.Config{Home: home})
	if err != nil {
		t.Fatalf("SchemaVersion after Migrate: %v", err)
	}
	if after != len(applied) {
		t.Fatalf("version %d after applying %d migrations", after, len(applied))
	}
}

// An archive written by a newer Burrow must be refused, not half-read.
func TestArchiveFromANewerBuildIsRefused(t *testing.T) {
	home := t.TempDir()
	ctx := context.Background()

	if _, err := engine.Migrate(ctx, engine.Config{Home: home}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := engine.SetSchemaVersionForTest(ctx, engine.Config{Home: home}, 9999); err != nil {
		t.Fatalf("bumping version: %v", err)
	}

	if _, err := engine.MigrationPlan(ctx, engine.Config{Home: home}); err == nil {
		t.Fatal("an archive from a newer build was accepted")
	}
}
