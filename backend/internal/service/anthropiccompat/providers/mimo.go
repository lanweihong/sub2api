package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

// init 注册小米 MiMo 渠道。
// 小米大模型平台提供 Anthropic-compatible 接口。
func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:       "anthropic-mimo",
		DisplayName:    "小米 MiMo (Anthropic-compatible)",
		DefaultBaseURL: "https://api.mimo.xiaomi.com",
		MessagesPath:   "/v1/messages",
		AuthMode:       anthropiccompat.AuthModeAPIKey,
		DefaultHeaders: map[string]string{},
		SupportsStreaming:    true,
		SupportsTools:       true,
		SupportsThinking:    true,
		SupportsCountTokens: false,
		DefaultModels: []string{
			"MiMo-72B-A27B-Instruct",
			"MiMo-7B-RL",
		},
	})
}
