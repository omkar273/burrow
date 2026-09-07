package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

// Bytes identified by their hash, with nothing about where they live —
// that belongs to Replica. The split is what lets a storage backend be
// added, repaired, or migrated without touching restore.
type Blob struct {
	ent.Schema
}

func (Blob) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

func (Blob) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			SchemaType(map[string]string{"sqlite3": "text"}).
			Immutable().
			Unique(),
		field.String("content_hash").
			SchemaType(map[string]string{"sqlite3": "text"}).
			Immutable().
			NotEmpty(),
		field.Int64("size_bytes").
			Immutable().
			NonNegative(),
	}
}

func (Blob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("versions", ObjectVersion.Type),
	}
}

func (Blob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("content_hash").Unique(),
	}
}
