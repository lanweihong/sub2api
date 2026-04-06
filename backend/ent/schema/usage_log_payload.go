package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UsageLogPayload 使用记录的请求/响应报文审计数据。
//
// 与 usage_logs 表通过 usage_log_id 关联（不建外键，因 usage_logs 为分区表）。
// 报文数据独立存储，按需加载，通过 retention_days 定期清理。
type UsageLogPayload struct {
	ent.Schema
}

func (UsageLogPayload) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_log_payloads"},
	}
}

func (UsageLogPayload) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("usage_log_id").
			Comment("关联 usage_logs.id"),
		field.Text("request_body").
			Optional().
			Nillable().
			Comment("截断后的请求报文"),
		field.Text("response_body").
			Optional().
			Nillable().
			Comment("截断后的响应报文"),
		field.Bool("request_truncated").
			Default(false).
			Comment("请求报文是否被截断"),
		field.Bool("response_truncated").
			Default(false).
			Comment("响应报文是否被截断"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("创建时间"),
	}
}

func (UsageLogPayload) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("usage_log_id").Unique(),
		index.Fields("created_at"),
	}
}
