package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIKeyGroup holds the edge schema definition for the api_key_groups relationship.
// It enables a many-to-many relationship between APIKey and Group with extra
// fields (priority, model_patterns) and uses a composite primary key.
type APIKeyGroup struct {
	ent.Schema
}

func (APIKeyGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_key_groups"},
		// Composite primary key: (api_key_id, group_id).
		field.ID("api_key_id", "group_id"),
	}
}

func (APIKeyGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("api_key_id"),
		field.Int64("group_id"),
		field.Int("priority").
			Default(0).
			Comment("Priority for group selection when multiple groups match; lower value = higher priority"),
		field.JSON("model_patterns", []string{}).
			Optional().
			Comment("Optional model prefix patterns to override group-level supported_model_scopes, e.g. [\"claude-*\", \"gpt-4*\"]"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (APIKeyGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("api_key", APIKey.Type).
			Unique().
			Required().
			Field("api_key_id"),
		edge.To("group", Group.Type).
			Unique().
			Required().
			Field("group_id"),
	}
}

func (APIKeyGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("priority"),
	}
}
