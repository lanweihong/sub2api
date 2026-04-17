//go:build unit

package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type compatHTTPUpstreamRecorder struct {
	lastReq *http.Request
	resp    *http.Response
	err     error
}

func (u *compatHTTPUpstreamRecorder) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.lastReq = req
	if u.err != nil {
		return nil, u.err
	}
	return u.resp, nil
}

func (u *compatHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func newAccountTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}

func TestAccountTestService_AnthropicCompatZhipuUsesProviderDefaultBaseURL(t *testing.T) {
	c, rec := newAccountTestContext()
	upstream := &compatHTTPUpstreamRecorder{
		resp: newJSONResponse(http.StatusOK, `{"content":[{"type":"text","text":"ok"}]}`),
	}
	svc := &AccountTestService{
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
		ID:          7,
		Platform:    PlatformAnthropicZhipu,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "zhipu-test-key",
			"base_url": "https://api.anthropic.com",
		},
	}

	err := svc.testAnthropicCompatAccountConnection(c, account, "")
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://open.bigmodel.cn/api/anthropic/v1/messages", upstream.lastReq.URL.String())
	require.Equal(t, "zhipu-test-key", upstream.lastReq.Header.Get("x-api-key"))
	require.Empty(t, upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "2023-06-01", upstream.lastReq.Header.Get("anthropic-version"))

	body, readErr := io.ReadAll(upstream.lastReq.Body)
	require.NoError(t, readErr)
	require.Contains(t, string(body), `"model":"glm-4-plus"`)
	require.Contains(t, rec.Body.String(), `"type":"done"`)
}

func TestAccountTestService_AnthropicCompatibleRequiresExplicitBaseURL(t *testing.T) {
	c, rec := newAccountTestContext()
	upstream := &compatHTTPUpstreamRecorder{
		resp: newJSONResponse(http.StatusOK, `{"content":[{"type":"text","text":"ok"}]}`),
	}
	svc := &AccountTestService{
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          8,
		Platform:    PlatformAnthropicCompatible,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "compat-test-key",
		},
	}

	err := svc.testAnthropicCompatAccountConnection(c, account, "")
	require.Error(t, err)
	require.Nil(t, upstream.lastReq)
	require.Contains(t, rec.Body.String(), "必须设置 base_url")
}
