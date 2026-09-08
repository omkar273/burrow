package service

import (
	"go.uber.org/fx"

	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	"github.com/omkar273/burrow/packages/engine/internal/storage"
)

type ServiceParams struct {
	fx.In

	Objects object.Repository
	Blobs   blob.Repository
	Sources source.Repository
	Store   storage.Store
	Tx      Txer
}
