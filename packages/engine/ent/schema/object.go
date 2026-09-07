package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

var sqliteText = map[string]string{"sqlite3": "text"}

type Object struct {
	ent.Schema
}

func (Object) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

func (Object) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			SchemaType(sqliteText).
			Immutable().
			Unique(),

		field.String("source_id").
			SchemaType(sqliteText).
			Immutable().
			NotEmpty(),

		field.String("kind").
			SchemaType(sqliteText).
			Immutable().
			NotEmpty(),

		field.String("external_id").
			SchemaType(sqliteText).
			Immutable().
			NotEmpty(),

		field.Time("first_seen_at"),
		field.Time("last_seen_at"),

		// Nillable: the provider no longer has this object. Emphatically
		// not a deletion from the archive.
		field.Time("deleted_at_source").
			Optional().
			Nillable(),
	}
}

func (Object) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_id", "external_id").Unique(),
	}
}
