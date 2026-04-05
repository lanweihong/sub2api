package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

// init 注册通义千问（Qwen）渠道。
// 阿里云百炼平台提供 Anthropic-compatible 接口，使用 DashScope API。
func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:       "anthropic-qwen",
		DisplayName:    "通义千问 / Qwen (Anthropic-compatible)",
		DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode",
		MessagesPath:   "/v1/messages",
		AuthMode:       anthropiccompat.AuthModeAPIKey,
		DefaultHeaders: map[string]string{},
		SupportsStreaming:    true,
		SupportsTools:       true,
		SupportsThinking:    true,
		SupportsCountTokens: false,
		DefaultModels: []string{
			"qwen-max",
			"qwen-plus",
			"qwen-turbo",
			"qwen-long",
			"qwen3-235b-a22b",
			"qwen3-32b",
			"qwen3-14b",
			"qwen3-8b",
		},
	})
}
