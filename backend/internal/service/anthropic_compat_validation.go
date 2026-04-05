package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func requiresExplicitAnthropicCompatBaseURL(raw string, spec *anthropiccompat.ProviderSpec) bool {
	if spec == nil {
		return false
	}
	return strings.TrimSpace(spec.DefaultBaseURL) == "" && strings.TrimSpace(raw) == ""
}

func validateAnthropicCompatAccountSettings(platform, accountType string, credentials map[string]any) error {
	if platform != PlatformAnthropicCompatible {
		return nil
	}

	if accountType != AccountTypeAPIKey {
		return errors.New("anthropic-compatible 仅支持 API Key 账号")
	}

	spec, ok := anthropiccompat.Resolve(platform)
	if !ok {
		return fmt.Errorf("未注册的 Anthropic-compatible 平台: %s", platform)
	}

	baseURL, _ := credentials["base_url"].(string)
	if requiresExplicitAnthropicCompatBaseURL(baseURL, spec) {
		return errors.New("anthropic-compatible 账号必须设置 base_url")
	}

	return nil
}
