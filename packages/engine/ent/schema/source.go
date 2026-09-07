package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

type Source struct {
	ent.Schema
}

func (Source) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

func (Source) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").SchemaType(sqliteText).Immutable().Unique(),
		field.String("kind").SchemaType(sqliteText).Immutable().NotEmpty(),
		field.String("account_email").SchemaType(sqliteText).NotEmpty(),
		field.String("status").SchemaType(sqliteText).NotEmpty(),
	}
}

func (Source) Edges() []ent.Edge {
	return []ent.Edge{edge.To("objects", Object.Type)}
}

func (Source) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("kind", "account_email").Unique(),
	}
}
