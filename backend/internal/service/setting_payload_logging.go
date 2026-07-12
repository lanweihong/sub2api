package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// PayloadLoggingSettings 报文审计记录配置
type PayloadLoggingSettings struct {
	Enabled         bool  `json:"enabled"`           // 是否开启报文记录
	MaxRequestSize  int64 `json:"max_request_size"`  // 请求体最大保存字节
	MaxResponseSize int64 `json:"max_response_size"` // 响应体最大保存字节
	RetentionDays   int   `json:"retention_days"`    // 保留天数，0=不自动清理
}

// DefaultPayloadLoggingSettings 返回默认配置
func DefaultPayloadLoggingSettings() *PayloadLoggingSettings {
	return &PayloadLoggingSettings{
		Enabled:         false,
		MaxRequestSize:  65536,
		MaxResponseSize: 65536,
		RetentionDays:   7,
	}
}

// cachedPayloadLoggingSettings 缓存报文审计配置（进程内缓存，60s TTL）
type cachedPayloadLoggingSettings struct {
	settings  *PayloadLoggingSettings
	expiresAt int64 // unix nano
}

var payloadLoggingCache atomic.Value // *cachedPayloadLoggingSettings
var payloadLoggingSF singleflight.Group

const payloadLoggingCacheTTL = 60 * time.Second
const payloadLoggingErrorTTL = 5 * time.Second
const payloadLoggingDBTimeout = 5 * time.Second

// GetPayloadLoggingSettings 获取报文记录配置
// 使用进程内 atomic.Value 缓存，60 秒 TTL，热路径零锁开销
// singleflight 防止缓存过期时 thundering herd
func (s *SettingService) GetPayloadLoggingSettings(ctx context.Context) (*PayloadLoggingSettings, error) {
	if cached, ok := payloadLoggingCache.Load().(*cachedPayloadLoggingSettings); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			if cached.settings != nil {
				slog.Debug("payload logging settings resolved",
					"source", "cache",
					"enabled", cached.settings.Enabled,
					"max_request_size", cached.settings.MaxRequestSize,
					"max_response_size", cached.settings.MaxResponseSize,
					"retention_days", cached.settings.RetentionDays,
					"expires_at_unix_nano", cached.expiresAt,
				)
			}
			return cached.settings, nil
		}
	}

	val, _, _ := payloadLoggingSF.Do("payload_logging", func() (any, error) {
		if cached, ok := payloadLoggingCache.Load().(*cachedPayloadLoggingSettings); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.settings, nil
			}
		}

		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), payloadLoggingDBTimeout)
		defer cancel()

		settings, err := s.loadPayloadLoggingSettingsFromDB(dbCtx)
		if err != nil {
			slog.Warn("failed to get payload logging settings", "error", err)
			defaults := DefaultPayloadLoggingSettings()
			payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
				settings:  defaults,
				expiresAt: time.Now().Add(payloadLoggingErrorTTL).UnixNano(),
			})
			return defaults, nil
		}

		payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
			settings:  settings,
			expiresAt: time.Now().Add(payloadLoggingCacheTTL).UnixNano(),
		})
		slog.Info("payload logging settings resolved",
			"source", "db",
			"enabled", settings.Enabled,
			"max_request_size", settings.MaxRequestSize,
			"max_response_size", settings.MaxResponseSize,
			"retention_days", settings.RetentionDays,
			"cache_ttl_seconds", payloadLoggingCacheTTL.Seconds(),
		)
		return settings, nil
	})

	if s, ok := val.(*PayloadLoggingSettings); ok {
		return s, nil
	}
	return DefaultPayloadLoggingSettings(), nil
}

// loadPayloadLoggingSettingsFromDB 从数据库加载报文记录配置（内部方法，供缓存刷新使用）
func (s *SettingService) loadPayloadLoggingSettingsFromDB(ctx context.Context) (*PayloadLoggingSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPayloadLoggingSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultPayloadLoggingSettings(), nil
		}
		return nil, fmt.Errorf("get payload logging settings: %w", err)
	}
	if value == "" {
		return DefaultPayloadLoggingSettings(), nil
	}

	var settings PayloadLoggingSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultPayloadLoggingSettings(), nil
	}

	if settings.MaxRequestSize < 1024 {
		settings.MaxRequestSize = 1024
	}
	if settings.MaxRequestSize > 524288 {
		settings.MaxRequestSize = 524288
	}
	if settings.MaxResponseSize < 1024 {
		settings.MaxResponseSize = 1024
	}
	if settings.MaxResponseSize > 524288 {
		settings.MaxResponseSize = 524288
	}
	if settings.RetentionDays < 0 {
		settings.RetentionDays = 0
	}
	if settings.RetentionDays > 365 {
		settings.RetentionDays = 365
	}

	return &settings, nil
}

// SetPayloadLoggingSettings 保存报文记录配置
func (s *SettingService) SetPayloadLoggingSettings(ctx context.Context, settings *PayloadLoggingSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	if settings.MaxRequestSize < 1024 || settings.MaxRequestSize > 524288 {
		return fmt.Errorf("max_request_size must be between 1024-524288")
	}
	if settings.MaxResponseSize < 1024 || settings.MaxResponseSize > 524288 {
		return fmt.Errorf("max_response_size must be between 1024-524288")
	}
	if settings.RetentionDays < 0 || settings.RetentionDays > 365 {
		return fmt.Errorf("retention_days must be between 0-365")
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal payload logging settings: %w", err)
	}

	if err := s.settingRepo.Set(ctx, SettingKeyPayloadLoggingSettings, string(data)); err != nil {
		return err
	}

	payloadLoggingSF.Forget("payload_logging")
	payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
		settings:  settings,
		expiresAt: time.Now().Add(payloadLoggingCacheTTL).UnixNano(),
	})
	slog.Info("payload logging settings updated",
		"enabled", settings.Enabled,
		"max_request_size", settings.MaxRequestSize,
		"max_response_size", settings.MaxResponseSize,
		"retention_days", settings.RetentionDays,
		"cache_scope", "local_process_only",
		"cache_ttl_seconds", payloadLoggingCacheTTL.Seconds(),
	)
	return nil
}

// IsPayloadLoggingEnabled 快捷判断（用于网关热路径）
func (s *SettingService) IsPayloadLoggingEnabled(ctx context.Context) bool {
	settings, err := s.GetPayloadLoggingSettings(ctx)
	if err != nil {
		return false
	}
	return settings.Enabled
}
