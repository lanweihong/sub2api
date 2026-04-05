package providers

import (
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
)

func init() {
	anthropiccompat.Register(&anthropiccompat.ProviderSpec{
		Platform:            "anthropic-compatible",
		DisplayName:         "其他 (Anthropic-compatible)",
		DefaultBaseURL:      "",
		MessagesPath:        "/v1/messages",
		AuthMode:            anthropiccompat.AuthModeAPIKey,
		DefaultHeaders:      map[string]string{},
		SupportsStreaming:   true,
		SupportsTools:       true,
		SupportsThinking:    true,
		SupportsCountTokens: false,
	})
}
