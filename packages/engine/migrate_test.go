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
