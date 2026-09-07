package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/ent/schema/mixin"
)

// Source is one connected provider account.
type Source struct {
	ent.Schema
}

// Mixin of the Source.
func (Source) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

// Fields of the Source.
func (Source) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(sqliteText).Immutable().Unique(),
		field.String("kind").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("account_email").SchemaType(sqliteText).NotEmpty(),
		field.String("status").SchemaType(sqliteText).NotEmpty(),
	}
}

// Indexes of the Source.
func (Source) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kind", "account_email").Unique(),
	}
}
