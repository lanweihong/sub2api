//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func TestResolveAnthropicCompatBaseURL(t *testing.T) {
	spec := &anthropiccompat.ProviderSpec{
		Platform:       "anthropic-zhipu",
		DefaultBaseURL: "https://open.bigmodel.cn/api/anthropic",
	}

	t.Run("empty base url uses provider default", func(t *testing.T) {
		got := resolveAnthropicCompatBaseURL("", spec)
		if got != spec.DefaultBaseURL {
			t.Fatalf("resolveAnthropicCompatBaseURL() = %q, want %q", got, spec.DefaultBaseURL)
		}
	})

	t.Run("legacy placeholder falls back to provider default", func(t *testing.T) {
		got := resolveAnthropicCompatBaseURL("https://api.anthropic.com/", spec)
		if got != spec.DefaultBaseURL {
			t.Fatalf("resolveAnthropicCompatBaseURL() = %q, want %q", got, spec.DefaultBaseURL)
		}
	})

	t.Run("custom base url is preserved", func(t *testing.T) {
		const custom = "https://relay.example.com/zhipu"
		got := resolveAnthropicCompatBaseURL(custom, spec)
		if got != custom {
			t.Fatalf("resolveAnthropicCompatBaseURL() = %q, want %q", got, custom)
		}
	})
}
