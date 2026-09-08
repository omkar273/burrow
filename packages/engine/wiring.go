package engine

import (
	"context"

	"go.uber.org/fx"

	"github.com/omkar273/burrow/packages/engine/internal/repository"
	"github.com/omkar273/burrow/packages/engine/internal/service"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
	"github.com/omkar273/burrow/packages/engine/internal/storage/localfs"
)

// options is the engine's entire dependency graph, in one place.
//
// It lives here rather than in cmd/burrow because the compiler forbids cmd/
// from reaching internal/, and because an embedder should get the same wired
// engine the binary does rather than reconstructing it.
//
// config.Profile is supplied by the caller: it comes from a flag.
var options = fx.Options(
	fx.Provide(
		// State.
		sqlitedb.NewClient,

		// Repositories, each as its domain interface.
		repository.NewObjectRepository,
		repository.NewBlobRepository,
		repository.NewSourceRepository,

		// Storage.
		localfs.NewFromProfile,

		// The sqlite client satisfies service.Txer. Naming it here keeps the
		// service layer free of any concrete database type.
		func(c *sqlitedb.Client) service.Txer { return c },

		// Use cases.
		service.NewIngestService,
		service.NewRestoreService,
	),

	// Migrate on start, close on stop, ordered by fx against everything that
	// depends on the database.
	fx.Invoke(func(lc fx.Lifecycle, c *sqlitedb.Client) {
		lc.Append(fx.Hook{
			OnStart: c.Migrate,
			OnStop:  func(context.Context) error { return c.Close() },
		})
	}),
)
