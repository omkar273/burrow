package repository

import (
	"go.uber.org/fx"

	"github.com/omkar273/burrow/packages/engine/internal/domain/blob"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	entrepo "github.com/omkar273/burrow/packages/engine/internal/repository/ent"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
)

type FactoryParams struct {
	fx.In

	Client *sqlitedb.Client
}

func NewObjectRepository(p FactoryParams) object.Repository {
	return entrepo.NewObjectRepository(p.Client)
}

func NewBlobRepository(p FactoryParams) blob.Repository {
	return entrepo.NewBlobRepository(p.Client)
}

func NewSourceRepository(p FactoryParams) source.Repository {
	return entrepo.NewSourceRepository(p.Client)
}
