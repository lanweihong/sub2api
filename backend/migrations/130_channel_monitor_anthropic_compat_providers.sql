-- Migration: 130_channel_monitor_anthropic_compat_providers
-- 扩展渠道监控 provider 字段，支持 anthropic-compatible / anthropic-* 国产兼容平台。
--
-- provider 后续由 service 层基于 monitor adapter / anthropiccompat registry 校验，
-- 数据库层不再维护硬编码三家白名单，避免每新增一个兼容平台都修改约束。

ALTER TABLE channel_monitors
    DROP CONSTRAINT IF EXISTS channel_monitors_provider_check;

ALTER TABLE channel_monitor_request_templates
    DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_provider_check;

ALTER TABLE channel_monitors
    ALTER COLUMN provider TYPE VARCHAR(50);

ALTER TABLE channel_monitor_request_templates
    ALTER COLUMN provider TYPE VARCHAR(50);
