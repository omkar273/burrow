package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

type ObjectVersion struct {
	ent.Schema
}

func (ObjectVersion) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

func (ObjectVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(sqliteText).Immutable().Unique(),
		field.String("object_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("blob_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		// Set when this content re-entered the provider through a restore.
		field.String("restored_from_version_id").SchemaType(sqliteText).
			Optional().Nillable(),
		field.Time("captured_at"),
	}
}

func (ObjectVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("object", Object.Type).
			Ref("versions").
			Field("object_id").
			Unique().
			Required().
			Immutable(),
		edge.From("blob", Blob.Type).
			Ref("versions").
			Field("blob_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ObjectVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("object_id", "captured_at"),
		index.Fields("blob_id"),
	}
}
