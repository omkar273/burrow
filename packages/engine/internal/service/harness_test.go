package service_test

import (
	"testing"

	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	"github.com/omkar273/burrow/packages/engine/internal/service"
	"github.com/omkar273/burrow/packages/engine/internal/testutil"
)

type harness struct {
	ingest  service.IngestService
	restore service.RestoreService
	src     *testutil.FakeSource
	store   *testutil.FakeStore
	objects object.Repository
	blobs   blob.Repository
}

// A real SQLite file with the real schema, and in-memory fakes for the
// provider and the blob store: no credentials, no network, real constraints.
func newHarness(t *testing.T) *harness {
	t.Helper()
	c := testutil.OpenMigratedClient(t)
	srcID := testutil.SeedSource(t, c, "test@example.com")

	objects := entrepo.NewObjectRepository(c)
	blobs := entrepo.NewBlobRepository(c)
	store := testutil.NewFakeStore()

	params := service.ServiceParams{
		Objects: objects,
		Blobs:   blobs,
		Sources: entrepo.NewSourceRepository(c),
		Store:   store,
		Tx:      c,
	}

	return &harness{
		ingest:  service.NewIngestService(params),
		restore: service.NewRestoreService(params),
		src:     testutil.NewFakeSource(srcID),
		store:   store,
		objects: objects,
		blobs:   blobs,
	}
}
