package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/ent/schema/mixin"
)

// ObjectVersion is an Object's content at a point in time.
type ObjectVersion struct {
	ent.Schema
}

// Mixin of the ObjectVersion.
func (ObjectVersion) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

// Fields of the ObjectVersion.
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

// Indexes of the ObjectVersion.
func (ObjectVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("object_id", "captured_at"),
		index.Fields("blob_id"),
	}
}
