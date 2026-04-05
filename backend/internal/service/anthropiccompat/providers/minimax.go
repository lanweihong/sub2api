package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:       "anthropic-minimax",
		DisplayName:    "MiniMax (Anthropic-compatible)",
		DefaultBaseURL: "https://api.minimax.chat",
		MessagesPath:   "/v1/messages",
		AuthMode:       anthropiccompat.AuthModeAPIKey,
		DefaultHeaders: map[string]string{},
		SupportsStreaming:    true,
		SupportsTools:       true,
		SupportsThinking:    false,
		SupportsCountTokens: false,
		DefaultModels: []string{
			"MiniMax-Text-01",
			"abab6.5s-chat",
			"abab6.5-chat",
		},
	})
}
