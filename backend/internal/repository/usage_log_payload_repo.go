package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type usageLogPayloadRepository struct {
	db *sql.DB
}

// NewUsageLogPayloadRepository 创建报文审计仓储实例
func NewUsageLogPayloadRepository(db *sql.DB) service.UsageLogPayloadRepository {
	return &usageLogPayloadRepository{db: db}
}

// Upsert 插入或更新报文记录（基于 usage_log_id 的 UNIQUE 约束）
func (r *usageLogPayloadRepository) Upsert(ctx context.Context, p *service.UsageLogPayloadRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO usage_log_payloads (usage_log_id, request_body, response_body, request_truncated, response_truncated, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (usage_log_id) DO UPDATE SET
			request_body = EXCLUDED.request_body,
			response_body = EXCLUDED.response_body,
			request_truncated = EXCLUDED.request_truncated,
			response_truncated = EXCLUDED.response_truncated
	`, p.UsageLogID, p.RequestBody, p.ResponseBody, p.RequestTruncated, p.ResponseTruncated)
	return err
}

// GetByUsageLogID 按 usage_log_id 查询单条报文记录
func (r *usageLogPayloadRepository) GetByUsageLogID(ctx context.Context, usageLogID int64) (*service.UsageLogPayloadRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, usage_log_id, request_body, response_body, request_truncated, response_truncated, created_at
		FROM usage_log_payloads WHERE usage_log_id = $1
	`, usageLogID)

	var rec service.UsageLogPayloadRecord
	err := row.Scan(
		&rec.ID,
		&rec.UsageLogID,
		&rec.RequestBody,
		&rec.ResponseBody,
		&rec.RequestTruncated,
		&rec.ResponseTruncated,
		&rec.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// DeleteBefore 删除指定时间之前的所有报文记录（分批删除，避免长事务）
func (r *usageLogPayloadRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	var total int64
	for {
		result, err := r.db.ExecContext(ctx, `
			DELETE FROM usage_log_payloads WHERE id IN (
				SELECT id FROM usage_log_payloads WHERE created_at < $1 LIMIT 1000
			)
		`, before)
		if err != nil {
			return total, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return total, err
		}
		total += affected
		if affected < 1000 {
			break
		}
	}
	return total, nil
}

// DeleteByUsageLogIDs 批量按 usage_log_id 删除
func (r *usageLogPayloadRepository) DeleteByUsageLogIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM usage_log_payloads WHERE usage_log_id = ANY($1)`,
		pq.Array(ids),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
