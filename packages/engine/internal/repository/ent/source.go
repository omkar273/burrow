package ent

import (
	"context"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	"github.com/omkar273/burrow/packages/engine/internal/domain/source"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
)

type sourceRepository struct{ c *Client }

func NewSourceRepository(c *Client) source.Repository { return &sourceRepository{c: c} }

// sourceFromEnt converts a generated row to the domain struct. Generated
// ent types must not escape this package.
func sourceFromEnt(s *generated.Source) source.Source {
	return source.Source{
		ID:           s.ID,
		Kind:         source.Kind(s.Kind),
		AccountEmail: s.AccountEmail,
		Status:       source.Status(s.Status),
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func (r *sourceRepository) Create(ctx context.Context, s source.Source) error {
	err := r.c.ent.Source.Create().
		SetID(s.ID).
		SetKind(string(s.Kind)).
		SetAccountEmail(s.AccountEmail).
		SetStatus(string(s.Status)).
		Exec(ctx)
	if err != nil {
		return ierr.Wrap(err, "creating source").Mark(ierr.ErrInternal)
	}
	return nil
}

func (r *sourceRepository) Get(ctx context.Context, id string) (source.Source, error) {
	row, err := r.c.ent.Source.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return source.Source{}, ierr.New("no source with id " + id).Mark(ierr.ErrNotFound)
		}
		return source.Source{}, ierr.Wrap(err, "getting source").Mark(ierr.ErrInternal)
	}
	return sourceFromEnt(row), nil
}

func (r *sourceRepository) List(ctx context.Context) ([]source.Source, error) {
	rows, err := r.c.ent.Source.Query().All(ctx)
	if err != nil {
		return nil, ierr.Wrap(err, "listing sources").Mark(ierr.ErrInternal)
	}
	out := make([]source.Source, len(rows))
	for i, row := range rows {
		out[i] = sourceFromEnt(row)
	}
	return out, nil
}
