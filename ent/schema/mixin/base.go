// Package mixin holds field sets shared across schemas.
package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	entmixin "entgo.io/ent/schema/mixin"
)

// BaseMixin is deliberately just timestamps.
//
// It does not carry TenantID, CreatedBy, UpdatedBy, or a soft-delete
// Status. Burrow is single-user and local-first; those are multi-tenant
// SaaS concerns. Deletion in particular needs its own semantics here,
// because deleted-at-source is emphatically not deleted-from-archive.
type BaseMixin struct {
	entmixin.Schema
}

func now() time.Time { return time.Now().UTC() }

// Fields of the BaseMixin.
func (BaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(now).Immutable(),
		field.Time("updated_at").Default(now).UpdateDefault(now),
	}
}
