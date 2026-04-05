package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:       "anthropic-kimi",
		DisplayName:    "Kimi / Moonshot (Anthropic-compatible)",
		DefaultBaseURL: "https://api.moonshot.cn",
		MessagesPath:   "/v1/messages",
		AuthMode:       anthropiccompat.AuthModeAPIKey,
		DefaultHeaders: map[string]string{},
		SupportsStreaming:    true,
		SupportsTools:       true,
		SupportsThinking:    true,
		SupportsCountTokens: false,
		DefaultModels: []string{
			"kimi-latest",
			"moonshot-v1-auto",
			"moonshot-v1-8k",
			"moonshot-v1-32k",
			"moonshot-v1-128k",
		},
	})
}
