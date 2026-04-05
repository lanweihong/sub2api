//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_ForwardAnthropicCompatZhipuUsesProviderDefaultBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(""))

	upstream := &httpUpstreamRecorder{
		resp: newJSONResponse(http.StatusOK, `{"content":[{"type":"text","text":"compat ok"}]}`),
	}
	svc := &GatewayService{
		httpUpstream: upstream,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{
					Enabled:       true,
					UpstreamHosts: []string{"open.bigmodel.cn"},
				},
			},
		},
	}
	account := &Account{
		ID:          9,
		Name:        "zhipu-compat",
		Platform:    PlatformAnthropicZhipu,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "zhipu-forward-key",
			"base_url": "https://api.anthropic.com/",
		},
	}
	parsed := &ParsedRequest{
		Model:  "glm-4-plus",
		Stream: false,
		Body:   []byte(`{"model":"glm-4-plus","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`),
	}

	result, err := svc.ForwardAnthropicCompat(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "https://open.bigmodel.cn/api/anthropic/v1/messages", upstream.lastReq.URL.String())
	require.Equal(t, "zhipu-forward-key", upstream.lastReq.Header.Get("x-api-key"))
	require.Empty(t, upstream.lastReq.Header.Get("Authorization"))
	require.JSONEq(t, `{"content":[{"type":"text","text":"compat ok"}]}`, rec.Body.String())
}

func TestGatewayService_ForwardAnthropicCompatibleRequiresExplicitBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(""))

	upstream := &httpUpstreamRecorder{
		resp: newJSONResponse(http.StatusOK, `{"content":[{"type":"text","text":"compat ok"}]}`),
	}
	svc := &GatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:          10,
		Name:        "other-compat",
		Platform:    PlatformAnthropicCompatible,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "compat-forward-key",
		},
	}
	parsed := &ParsedRequest{
		Model:  "claude-3-5-haiku-20241022",
		Stream: false,
		Body:   []byte(`{"model":"claude-3-5-haiku-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`),
	}

	result, err := svc.ForwardAnthropicCompat(context.Background(), c, account, parsed)
	require.Nil(t, result)
	require.Error(t, err)
	require.Nil(t, upstream.lastReq)
	require.Contains(t, rec.Body.String(), "必须设置 base_url")
}
