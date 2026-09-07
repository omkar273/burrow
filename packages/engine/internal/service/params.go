package service

import (
	"go.uber.org/fx"

	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

// ServiceParams is everything the use cases need, passed as one value.
//
// Every service takes exactly this, so adding a dependency changes one struct
// rather than every constructor and call site, and a service can build another
// from its own params without threading arguments.
//
// fx.In lets the container populate it field by field. It is a zero-size
// marker, so tests still construct the struct literally with fakes.
type ServiceParams struct {
	fx.In

	Objects object.Repository
	Blobs   blob.Repository
	Sources source.Repository
	Store   storage.Store
	Tx      Txer
}
