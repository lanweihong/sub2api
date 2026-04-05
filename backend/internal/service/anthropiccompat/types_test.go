//go:build unit

// Package anthropiccompat_test 包含 ProviderSpec 方法的单元测试。
package anthropiccompat_test

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

// TestProviderSpecMessagesEndpointPath 验证消息端点路径的默认值回退行为。
func TestProviderSpecMessagesEndpointPath(t *testing.T) {
	t.Run("自定义路径生效", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform:     "anthropic-test",
			MessagesPath: "/custom/messages",
		}
		got := spec.MessagesEndpointPath()
		if got != "/custom/messages" {
			t.Errorf("MessagesEndpointPath() = %q，期望 %q", got, "/custom/messages")
		}
	})

	t.Run("空路径时回退默认值", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform: "anthropic-test",
		}
		got := spec.MessagesEndpointPath()
		if got != "/v1/messages" {
			t.Errorf("MessagesEndpointPath() = %q，期望默认 %q", got, "/v1/messages")
		}
	})
}

// TestProviderSpecAnthropicVersionHeader 验证 anthropic-version 头的默认值回退行为。
func TestProviderSpecAnthropicVersionHeader(t *testing.T) {
	t.Run("自定义版本生效", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform:         "anthropic-test",
			AnthropicVersion: "2024-01-01",
		}
		got := spec.AnthropicVersionHeader()
		if got != "2024-01-01" {
			t.Errorf("AnthropicVersionHeader() = %q，期望 %q", got, "2024-01-01")
		}
	})

	t.Run("空版本时回退默认值", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform: "anthropic-test",
		}
		got := spec.AnthropicVersionHeader()
		if got != "2023-06-01" {
			t.Errorf("AnthropicVersionHeader() = %q，期望默认 %q", got, "2023-06-01")
		}
	})
}

// TestProviderSpecResolveAuthHeader 验证鉴权请求头的解析逻辑。
func TestProviderSpecResolveAuthHeader(t *testing.T) {
	t.Run("APIKey 模式使用默认头名称", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform: "anthropic-test",
			AuthMode: anthropiccompat.AuthModeAPIKey,
		}
		name, value := spec.ResolveAuthHeader("my-api-key")
		if name != "x-api-key" {
			t.Errorf("头名称 = %q，期望 %q", name, "x-api-key")
		}
		if value != "my-api-key" {
			t.Errorf("头值 = %q，期望 %q", value, "my-api-key")
		}
	})

	t.Run("APIKey 模式使用自定义头名称", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform:       "anthropic-test",
			AuthMode:       anthropiccompat.AuthModeAPIKey,
			AuthHeaderName: "api-key",
		}
		name, value := spec.ResolveAuthHeader("my-api-key")
		if name != "api-key" {
			t.Errorf("头名称 = %q，期望 %q", name, "api-key")
		}
		if value != "my-api-key" {
			t.Errorf("头值 = %q，期望 %q", value, "my-api-key")
		}
	})

	t.Run("Bearer 模式生成 Authorization 头", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform: "anthropic-test",
			AuthMode: anthropiccompat.AuthModeBearer,
		}
		name, value := spec.ResolveAuthHeader("my-token")
		if name != "Authorization" {
			t.Errorf("头名称 = %q，期望 %q", name, "Authorization")
		}
		if value != "Bearer my-token" {
			t.Errorf("头值 = %q，期望 %q", value, "Bearer my-token")
		}
	})

	t.Run("默认模式（零值）等同于 APIKey 模式", func(t *testing.T) {
		spec := &anthropiccompat.ProviderSpec{
			Platform: "anthropic-test",
			// AuthMode 未设置，使用零值
		}
		name, value := spec.ResolveAuthHeader("default-key")
		if name != "x-api-key" {
			t.Errorf("默认模式头名称 = %q，期望 %q", name, "x-api-key")
		}
		if value != "default-key" {
			t.Errorf("默认模式头值 = %q，期望 %q", value, "default-key")
		}
	})
}
