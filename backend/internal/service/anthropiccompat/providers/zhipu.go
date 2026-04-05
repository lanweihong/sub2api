package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:    "anthropic-zhipu",
		DisplayName: "智谱 AI (Anthropic-compatible)",
		DefaultBaseURL: "https://open.bigmodel.cn",
		MessagesPath:   "/api/paas/v4/messages",
		AuthMode:       anthropiccompat.AuthModeAPIKey,
		DefaultHeaders: map[string]string{},
		SupportsStreaming:    true,
		SupportsTools:       true,
		SupportsThinking:    false,
		SupportsCountTokens: false,
		DefaultModels: []string{
			"glm-4-plus",
			"glm-4-flash",
			"glm-4-long",
			"glm-4",
		},
	})
}
