package service

import (
	"context"
	"time"
)

// UsageLogPayloadRepository 报文审计仓储接口
type UsageLogPayloadRepository interface {
	// Upsert 插入或更新（基于 usage_log_id 的 UNIQUE 约束）
	Upsert(ctx context.Context, payload *UsageLogPayloadRecord) error
	// GetByUsageLogID 按 usage_log_id 查询单条
	GetByUsageLogID(ctx context.Context, usageLogID int64) (*UsageLogPayloadRecord, error)
	// DeleteBefore 删除指定时间之前的记录，返回删除行数
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
	// DeleteByUsageLogIDs 批量按 usage_log_id 删除
	DeleteByUsageLogIDs(ctx context.Context, ids []int64) (int64, error)
}

// UsageLogPayloadRecord 报文审计数据传输对象
type UsageLogPayloadRecord struct {
	ID                int64     `json:"id"`
	UsageLogID        int64     `json:"usage_log_id"`
	RequestBody       *string   `json:"request_body"`
	ResponseBody      *string   `json:"response_body"`
	RequestTruncated  bool      `json:"request_truncated"`
	ResponseTruncated bool      `json:"response_truncated"`
	CreatedAt         time.Time `json:"created_at"`
}
