-- 083_create_api_key_groups.sql
-- 多分组 API Key：允许单个 API Key 绑定多个分组，根据模型前缀自动切换服务商

CREATE TABLE IF NOT EXISTS api_key_groups (
    api_key_id     BIGINT      NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    group_id       BIGINT      NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    priority       INT         NOT NULL DEFAULT 0,
    model_patterns JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (api_key_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_akg_group_id ON api_key_groups (group_id);
CREATE INDEX IF NOT EXISTS idx_akg_priority ON api_key_groups (priority);

COMMENT ON TABLE api_key_groups IS '多分组 API Key 关联表：支持单个 Key 绑定多个分组';
COMMENT ON COLUMN api_key_groups.priority IS '分组优先级，值越小优先级越高（多分组模型冲突时使用）';
COMMENT ON COLUMN api_key_groups.model_patterns IS '可选的模型前缀匹配模式覆盖，如 ["claude-*", "gpt-4*"]';
