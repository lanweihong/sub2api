package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Department holds the schema definition for the Department entity.
// 组织架构部门，与计费分组 (groups) 解耦。
type Department struct {
	ent.Schema
}

func (Department) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "departments"},
	}
}

func (Department) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (Department) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("部门名称"),
		field.String("code").
			MaxLen(50).
			Default("").
			Comment("部门短代码，可选；非空时全表唯一"),
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default("").
			Comment("部门描述"),
		field.Int64("parent_id").
			Optional().
			Nillable().
			Comment("父部门 ID，NULL 表示顶层部门"),
		field.Int("sort_order").
			Default(0).
			Comment("同级排序，数值越小越靠前"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("active 表示可被新分配；disabled 仅保留旧绑定"),
		field.Bool("is_default").
			Default(false).
			Comment("系统默认部门标记，全局唯一且禁止删除"),
	}
}

func (Department) Edges() []ent.Edge {
	return []ent.Edge{
		// 父部门（自引用）
		edge.To("parent", Department.Type).
			Unique().
			Field("parent_id"),
		// 子部门
		edge.From("children", Department.Type).
			Ref("parent"),
		// 部门下的用户
		edge.From("users", User.Type).
			Ref("department"),
	}
}

func (Department) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_id"),
		index.Fields("status"),
		index.Fields("sort_order"),
		index.Fields("deleted_at"),
	}
}
