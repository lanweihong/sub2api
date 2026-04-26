//go:build unit

package service

import (
	"context"
	"testing"
)

func TestChannelMonitorValidateProviderSupportsAnthropicCompatPlatforms(t *testing.T) {
	supported := []string{
		MonitorProviderOpenAI,
		MonitorProviderAnthropic,
		MonitorProviderGemini,
		PlatformAnthropicCompatible,
		PlatformAnthropicZhipu,
		PlatformAnthropicKimi,
		PlatformAnthropicMinimax,
		PlatformAnthropicQwen,
		PlatformAnthropicMimo,
	}
	for _, provider := range supported {
		if err := validateProvider(provider); err != nil {
			t.Fatalf("validateProvider(%q) returned error: %v", provider, err)
		}
	}
}

func TestChannelMonitorValidateProviderRejectsUnknownAnthropicCompat(t *testing.T) {
	if err := validateProvider("anthropic-unknown"); err != ErrChannelMonitorInvalidProvider {
		t.Fatalf("validateProvider(anthropic-unknown) = %v, want %v", err, ErrChannelMonitorInvalidProvider)
	}
}

func TestChannelMonitorEndpointPathPolicyForAnthropicCompat(t *testing.T) {
	if err := validateEndpointForProvider(MonitorProviderAnthropic, "https://api.example.com/v1"); err != ErrChannelMonitorEndpointPath {
		t.Fatalf("official anthropic endpoint with path = %v, want %v", err, ErrChannelMonitorEndpointPath)
	}
	if err := validateEndpointForProvider(PlatformAnthropicZhipu, "https://127.0.0.1/api/anthropic"); err != ErrChannelMonitorEndpointPrivate {
		t.Fatalf("anthropic-compatible endpoint with path = %v, want private-host validation after path is allowed", err)
	}
	if err := validateEndpointForProvider(PlatformAnthropicZhipu, "https://api.example.com/api/anthropic?debug=1"); err != ErrChannelMonitorEndpointPath {
		t.Fatalf("anthropic-compatible endpoint with query = %v, want %v", err, ErrChannelMonitorEndpointPath)
	}
}

func TestRunCheckForModel_AnthropicCompatUsesRegisteredProviderSpec(t *testing.T) {
	h := &captureHandler{respondText: "the answer is 42"}
	endpoint := setupFakeAnthropic(t, h) + "/api/anthropic"

	opts := &CheckOptions{
		BodyOverrideMode: MonitorBodyOverrideModeMerge,
		BodyOverride: map[string]any{
			"model":      "hacked-model",
			"messages":   []any{},
			"max_tokens": float64(999),
			"system":     "compat monitor",
		},
	}
	_ = runCheckForModel(context.Background(), PlatformAnthropicZhipu, endpoint, "zhipu-key", "glm-4-plus", opts)

	if h.lastPath != "/api/anthropic/v1/messages" {
		t.Fatalf("request path = %q, want /api/anthropic/v1/messages", h.lastPath)
	}
	if got := h.lastHeaders.Get("x-api-key"); got != "zhipu-key" {
		t.Fatalf("x-api-key = %q, want zhipu-key", got)
	}
	if got := h.lastHeaders.Get("anthropic-version"); got != monitorAnthropicAPIVersion {
		t.Fatalf("anthropic-version = %q, want %q", got, monitorAnthropicAPIVersion)
	}
	if got := h.lastBody["model"]; got != "glm-4-plus" {
		t.Fatalf("merge deny list should preserve model, got %v", got)
	}
	if msgs, _ := h.lastBody["messages"].([]any); len(msgs) == 0 {
		t.Fatal("merge deny list should preserve default messages")
	}
	if got := h.lastBody["max_tokens"]; got != float64(999) {
		t.Fatalf("merge mode should allow max_tokens override, got %v", got)
	}
}
