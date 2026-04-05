package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

const legacyAnthropicCompatPlaceholderBaseURL = "https://api.anthropic.com"

func resolveAnthropicCompatBaseURL(raw string, spec *anthropiccompat.ProviderSpec) string {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		if spec == nil {
			return ""
		}
		return spec.DefaultBaseURL
	}
	if spec != nil && isLegacyAnthropicCompatPlaceholderBaseURL(baseURL) {
		return spec.DefaultBaseURL
	}
	return baseURL
}

func isLegacyAnthropicCompatPlaceholderBaseURL(raw string) bool {
	normalized := strings.TrimRight(strings.TrimSpace(raw), "/")
	return strings.EqualFold(normalized, legacyAnthropicCompatPlaceholderBaseURL)
}
