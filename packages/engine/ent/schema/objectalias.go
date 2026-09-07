package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

// An extra provider ID resolving to the same Object. Restoring mints a new
// provider ID; recording it here stops the next sync forking the archive.
type ObjectAlias struct {
	ent.Schema
}

func (ObjectAlias) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

func (ObjectAlias) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(sqliteText).Immutable().Unique(),
		field.String("object_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("source_id").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("external_id").SchemaType(sqliteText).Immutable().NotEmpty(),
	}
}

func (ObjectAlias) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("object", Object.Type).
			Ref("aliases").
			Field("object_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (ObjectAlias) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_id", "external_id").Unique(),
	}
}
