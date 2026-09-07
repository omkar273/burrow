package engine

import (
	"context"
	"testing"

	"go.uber.org/fx"

	"github.com/omkar273/burrow/packages/engine/internal/config"
	"github.com/omkar273/burrow/packages/engine/internal/service"
)

// A missing provider is a runtime failure on a user's machine, not a compile
// error. ValidateApp resolves the graph without starting anything.
func TestGraphIsSatisfiable(t *testing.T) {
	err := fx.ValidateApp(
		fx.NopLogger,
		fx.Supply(config.Profile{}),
		options,
		fx.Invoke(func(*service.Ingest, *service.Restore) {}),
	)
	if err != nil {
		t.Fatalf("dependency graph does not resolve: %v", err)
	}
}

// And it must start: migrations run, storage opens, hooks fire in order.
func TestGraphStartsAndStops(t *testing.T) {
	profile, err := config.ResolveProfile("default", func(string) string { return "" }, t.TempDir())
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if err := profile.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	var ingest *service.Ingest
	app := fx.New(fx.NopLogger, fx.Supply(profile), options, fx.Populate(&ingest))

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if ingest == nil {
		t.Fatal("ingest was not populated")
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
