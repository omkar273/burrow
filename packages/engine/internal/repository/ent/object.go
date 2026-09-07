package ent

import (
	"context"
	"time"

	generated "github.com/omkar273/burrow/packages/engine/ent"
	entobject "github.com/omkar273/burrow/packages/engine/ent/object"
	entalias "github.com/omkar273/burrow/packages/engine/ent/objectalias"
	entversion "github.com/omkar273/burrow/packages/engine/ent/objectversion"
	"github.com/omkar273/burrow/packages/engine/internal/domain/object"
	ierr "github.com/omkar273/burrow/packages/engine/internal/errors"
	sqlitedb "github.com/omkar273/burrow/packages/engine/internal/sqlite"
	"github.com/omkar273/burrow/packages/engine/internal/types"
)

type objectRepository struct{ c *sqlitedb.Client }

func NewObjectRepository(c *sqlitedb.Client) object.Repository { return &objectRepository{c: c} }

func objectFromEnt(o *generated.Object) object.Object {
	return object.Object{
		ID:              o.ID,
		SourceID:        o.SourceID,
		Kind:            object.Kind(o.Kind),
		ExternalID:      o.ExternalID,
		FirstSeenAt:     o.FirstSeenAt,
		LastSeenAt:      o.LastSeenAt,
		DeletedAtSource: o.DeletedAtSource,
	}
}

func versionFromEnt(v *generated.ObjectVersion) object.Version {
	return object.Version{
		ID:                    v.ID,
		ObjectID:              v.ObjectID,
		BlobID:                v.BlobID,
		RestoredFromVersionID: v.RestoredFromVersionID,
		CapturedAt:            v.CapturedAt,
	}
}

func (r *objectRepository) Create(ctx context.Context, o *object.Object) error {
	now := time.Now().UTC()
	if o.FirstSeenAt.IsZero() {
		o.FirstSeenAt = now
	}
	if o.LastSeenAt.IsZero() {
		o.LastSeenAt = now
	}
	err := r.c.Querier(ctx).Object.Create().
		SetID(o.ID).
		SetSourceID(o.SourceID).
		SetKind(string(o.Kind)).
		SetExternalID(o.ExternalID).
		SetFirstSeenAt(o.FirstSeenAt).
		SetLastSeenAt(o.LastSeenAt).
		SetNillableDeletedAtSource(o.DeletedAtSource).
		Exec(ctx)
	if err != nil {
		return ierr.Wrap(err, "creating object").Mark(ierr.ErrInternal)
	}
	return nil
}

func (r *objectRepository) Get(ctx context.Context, id string) (object.Object, error) {
	row, err := r.c.Querier(ctx).Object.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return object.Object{}, ierr.New("no object with id " + id).Mark(ierr.ErrNotFound)
		}
		return object.Object{}, ierr.Wrap(err, "getting object").Mark(ierr.ErrInternal)
	}
	return objectFromEnt(row), nil
}

// GetByExternalID resolves a provider ID, falling through to aliases.
//
// The alias fallthrough is what makes restore safe: a restored message
// returns from the provider under a new ID that points at content we
// already hold.
func (r *objectRepository) GetByExternalID(ctx context.Context, sourceID, externalID string) (object.Object, error) {
	row, err := r.c.Querier(ctx).Object.Query().
		Where(entobject.SourceID(sourceID), entobject.ExternalID(externalID)).
		Only(ctx)
	if err == nil {
		return objectFromEnt(row), nil
	}
	if !generated.IsNotFound(err) {
		return object.Object{}, ierr.Wrap(err, "querying object by external id").Mark(ierr.ErrInternal)
	}

	alias, err := r.c.Querier(ctx).ObjectAlias.Query().
		Where(entalias.SourceID(sourceID), entalias.ExternalID(externalID)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return object.Object{}, ierr.New("no object for external id " + externalID).
				Mark(ierr.ErrNotFound)
		}
		return object.Object{}, ierr.Wrap(err, "querying object alias").Mark(ierr.ErrInternal)
	}
	return r.Get(ctx, alias.ObjectID)
}

// AddAlias refuses to shadow an identity an Object already claims.
//
// objects and object_alias hold separate unique indexes on
// (source_id, external_id), so the schema alone permits the same identity in
// both. GetByExternalID consults objects first, so a shadowing alias would
// resolve to the wrong object. The check runs in the same transaction as the
// insert, because two concurrent AddAlias calls could otherwise both pass it.
func (r *objectRepository) AddAlias(ctx context.Context, objectID, sourceID, externalID string) error {
	return r.c.WithTx(ctx, func(ctx context.Context) error {
		shadowed, err := r.c.Querier(ctx).Object.Query().
			Where(entobject.SourceID(sourceID), entobject.ExternalID(externalID)).
			Exist(ctx)
		if err != nil {
			return ierr.Wrap(err, "checking for a shadowed object identity").Mark(ierr.ErrInternal)
		}
		if shadowed {
			return ierr.New("external id " + externalID + " already belongs to an object").
				WithHint("an alias may not shadow an identity an object already claims").
				Mark(ierr.ErrValidation)
		}
		if err := r.c.Querier(ctx).ObjectAlias.Create().
			SetID(types.NewID(types.PrefixObject)).
			SetObjectID(objectID).
			SetSourceID(sourceID).
			SetExternalID(externalID).
			Exec(ctx); err != nil {
			return ierr.Wrap(err, "creating object alias").Mark(ierr.ErrInternal)
		}
		return nil
	})
}
func (r *objectRepository) CreateVersion(ctx context.Context, v *object.Version) error {
	if v.CapturedAt.IsZero() {
		v.CapturedAt = time.Now().UTC()
	}
	create := r.c.Querier(ctx).ObjectVersion.Create().
		SetID(v.ID).
		SetObjectID(v.ObjectID).
		SetBlobID(v.BlobID).
		SetCapturedAt(v.CapturedAt)
	if v.RestoredFromVersionID != nil {
		create = create.SetRestoredFromVersionID(*v.RestoredFromVersionID)
	}
	if err := create.Exec(ctx); err != nil {
		return ierr.Wrap(err, "creating object version").Mark(ierr.ErrInternal)
	}
	return nil
}

func (r *objectRepository) CurrentVersion(ctx context.Context, objectID string) (object.Version, error) {
	row, err := r.c.Querier(ctx).ObjectVersion.Query().
		Where(entversion.ObjectID(objectID)).
		Order(generated.Desc(entversion.FieldCapturedAt), generated.Desc(entversion.FieldID)).
		First(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return object.Version{}, ierr.New("no version for object " + objectID).Mark(ierr.ErrNotFound)
		}
		return object.Version{}, ierr.Wrap(err, "getting current version").Mark(ierr.ErrInternal)
	}
	return versionFromEnt(row), nil
}
