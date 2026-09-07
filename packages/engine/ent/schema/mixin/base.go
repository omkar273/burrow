// Package mixin holds field sets shared across schemas.
package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	entmixin "entgo.io/ent/schema/mixin"
)

type BaseMixin struct {
	entmixin.Schema
}

func now() time.Time { return time.Now().UTC() }

func (BaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(now).Immutable(),
		field.Time("updated_at").Default(now).UpdateDefault(now),
	}
}
