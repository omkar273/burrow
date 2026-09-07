package source

import "context"

type Repository interface {
	Create(ctx context.Context, s Source) error
	Get(ctx context.Context, id string) (Source, error)
	List(ctx context.Context) ([]Source, error)
}
