package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	basemixin "github.com/omkar273/burrow/packages/engine/ent/schema/mixin"
)

// Blob is content: bytes identified by their hash, and nothing about
// where those bytes physically live. Location belongs to Replica. This
// separation is what allows adding, repairing, or migrating a storage
// backend without touching restore.
type Blob struct {
	ent.Schema
}

// Mixin of the Blob.
func (Blob) Mixin() []ent.Mixin {
	return []ent.Mixin{basemixin.BaseMixin{}}
}

// Fields of the Blob.
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

// Indexes of the Blob.
func (Blob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("content_hash").Unique(),
	}
}
