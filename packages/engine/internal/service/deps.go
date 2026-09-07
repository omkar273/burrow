package service

import (
	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

// Deps is everything the use cases need, passed as one value.
//
// Every service takes exactly this, so adding a dependency later — a job
// queue, a logger, a verifier — changes one struct instead of every
// constructor and every call site. It also lets a service build another
// without threading arguments: see the constructor methods below.
//
// The fields are interfaces, so tests substitute fakes without a mocking
// framework.
type Deps struct {
	Objects object.Repository
	Blobs   blob.Repository
	Sources source.Repository
	Store   storage.Store
}

// Ingest returns the ingest use case.
func (d Deps) Ingest() *Ingest { return &Ingest{deps: d} }

// Restore returns the restore use case.
func (d Deps) Restore() *Restore { return &Restore{deps: d} }

// NewIngest exists for fx, which provides constructors rather than methods.
func NewIngest(d Deps) *Ingest { return d.Ingest() }

// NewRestore exists for fx, which provides constructors rather than methods.
func NewRestore(d Deps) *Restore { return d.Restore() }
