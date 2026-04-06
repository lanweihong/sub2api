-- 082_create_usage_log_payloads.sql
-- 报文审计记录：存储 API 请求/响应的原始报文数据

CREATE TABLE IF NOT EXISTS usage_log_payloads (
    id                 BIGSERIAL       PRIMARY KEY,
    usage_log_id       BIGINT          NOT NULL,
    request_body       TEXT,
    response_body      TEXT,
    request_truncated  BOOLEAN         NOT NULL DEFAULT FALSE,
    response_truncated BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_ulp_usage_log_id ON usage_log_payloads (usage_log_id);
CREATE INDEX idx_ulp_created_at ON usage_log_payloads (created_at);

COMMENT ON TABLE usage_log_payloads IS '使用记录的请求/响应报文审计数据';
COMMENT ON COLUMN usage_log_payloads.usage_log_id IS '关联 usage_logs.id（不建外键，因 usage_logs 为分区表）';
COMMENT ON COLUMN usage_log_payloads.request_body IS '截断后的请求报文（JSON 文本）';
COMMENT ON COLUMN usage_log_payloads.response_body IS '截断后的响应报文（JSON 文本或 SSE 文本）';
COMMENT ON COLUMN usage_log_payloads.request_truncated IS '请求报文是否因超长被截断';
COMMENT ON COLUMN usage_log_payloads.response_truncated IS '响应报文是否因超长被截断';
