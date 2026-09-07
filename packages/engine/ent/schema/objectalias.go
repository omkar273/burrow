package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

// ObjectAlias is an additional provider ID resolving to the same Object.
// Restoring content mints a new provider ID; recording it here is what
// stops the next sync from forking the archive.
type ObjectAlias struct {
	ent.Schema
}

// Mixin of the ObjectAlias.
func (ObjectAlias) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

// Fields of the ObjectAlias.
func (ObjectAlias) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(sqliteText).Immutable().Unique(),
		field.String("object_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("source_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("external_id").SchemaType(sqliteText).Immutable().NotEmpty(),
	}
}

// Indexes of the ObjectAlias.
func (ObjectAlias) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_id", "external_id").Unique(),
	}
}
