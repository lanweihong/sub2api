//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBuildOpenAIChatCompletionsURL_Direct(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "empty uses platform default", base: "", want: openAIPlatformChatCompletionsURL},
		{name: "root appends v1 path", base: "https://example.com", want: "https://example.com/v1/chat/completions"},
		{name: "v1 appends chat completions", base: "https://example.com/v1", want: "https://example.com/v1/chat/completions"},
		{name: "responses endpoint replaced", base: "https://example.com/v1/responses", want: "https://example.com/v1/chat/completions"},
		{name: "responses endpoint trailing slash replaced", base: "https://example.com/v1/responses/", want: "https://example.com/v1/chat/completions"},
		{name: "non-v1 responses endpoint replaced", base: "https://example.com/api/responses", want: "https://example.com/api/chat/completions"},
		{name: "chat completions kept", base: "https://example.com/v1/chat/completions", want: "https://example.com/v1/chat/completions"},
		{name: "prefixed v1 kept", base: "https://example.com/openai/v1", want: "https://example.com/openai/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, buildOpenAIChatCompletionsURL(tt.base))
		})
	}
}

func TestExtractOpenAIUsageFromJSONBytes_ChatCompletions(t *testing.T) {
	usage, ok := extractOpenAIUsageFromJSONBytes([]byte(`{"usage":{"prompt_tokens":12,"completion_tokens":7,"prompt_tokens_details":{"cached_tokens":5}}}`))
	require.True(t, ok)
	require.Equal(t, 12, usage.InputTokens)
	require.Equal(t, 7, usage.OutputTokens)
	require.Equal(t, 5, usage.CacheReadInputTokens)
}

func TestParseSSEUsage_ChatCompletionsChunk(t *testing.T) {
	svc := &OpenAIGatewayService{}
	usage := &OpenAIUsage{InputTokens: 99, OutputTokens: 88, CacheReadInputTokens: 77}

	svc.parseSSEUsage(`{"id":"chunk","object":"chat.completion.chunk","choices":[{"delta":{"content":"hi"}}]}`, usage)
	require.Equal(t, 99, usage.InputTokens)
	require.Equal(t, 88, usage.OutputTokens)
	require.Equal(t, 77, usage.CacheReadInputTokens)

	svc.parseSSEUsage(`{"id":"chunk","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":3,"completion_tokens":2,"prompt_tokens_details":{"cached_tokens":1}}}`, usage)
	require.Equal(t, 3, usage.InputTokens)
	require.Equal(t, 2, usage.OutputTokens)
	require.Equal(t, 1, usage.CacheReadInputTokens)

	svc.parseSSEUsage("[DONE]", usage)
	require.Equal(t, 3, usage.InputTokens)
	require.Equal(t, 2, usage.OutputTokens)
	require.Equal(t, 1, usage.CacheReadInputTokens)
}

func TestForwardChatCompletionsDirect_NonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set("Accept", "application/json")
	c.Request.Header.Set("User-Agent", "test-client/1.0")
	c.Request.Header.Set("X-Should-Drop", "yes")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-direct"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"cmp_1","object":"chat.completion","model":"upstream-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":7,"prompt_tokens_details":{"cached_tokens":5}}}`,
		)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:       101,
		Name:     "cc-direct",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":       "sk-test",
			"base_url":      "https://upstream.example/v1/responses",
			"model_mapping": map[string]any{"client-model": "upstream-model"},
		},
		Extra: map[string]any{"openai_cc_direct_forward": true},
	}
	body := []byte(`{"model":"client-model","stream":false,"service_tier":"fast","reasoning_effort":"high","messages":[{"role":"user","content":"hi"}]}`)

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.NoError(t, err)
	require.Equal(t, "https://upstream.example/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "test-client/1.0", upstream.lastReq.Header.Get("User-Agent"))
	require.Empty(t, upstream.lastReq.Header.Get("X-Should-Drop"))
	require.Equal(t, "upstream-model", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "priority", gjson.GetBytes(upstream.lastBody, "service_tier").String())
	require.Equal(t, "client-model", gjson.GetBytes(rec.Body.Bytes(), "model").String())
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 5, result.Usage.CacheReadInputTokens)
	require.Equal(t, "client-model", result.Model)
	require.Equal(t, "upstream-model", result.BillingModel)
	require.Equal(t, "upstream-model", result.UpstreamModel)
	require.NotNil(t, result.ServiceTier)
	require.Equal(t, "priority", *result.ServiceTier)
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "high", *result.ReasoningEffort)
}

func TestForwardChatCompletionsDirect_Streaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	upstreamBody := strings.Join([]string{
		`data: {"id":"chunk","object":"chat.completion.chunk","model":"upstream-model","choices":[{"delta":{"content":"hi"}}]}`,
		``,
		`data: {"id":"chunk","object":"chat.completion.chunk","model":"upstream-model","choices":[],"usage":{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:       102,
		Name:     "cc-direct-stream",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":       "sk-test",
			"base_url":      "https://upstream.example/v1",
			"model_mapping": map[string]any{"client-model": "upstream-model"},
		},
		Extra: map[string]any{"openai_cc_direct_forward": true},
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, []byte(`{"model":"client-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`), "", "")
	require.NoError(t, err)
	require.True(t, result.Stream)
	require.NotNil(t, result.FirstTokenMs)
	require.Equal(t, 9, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	require.Equal(t, 2, result.Usage.CacheReadInputTokens)
	require.Contains(t, rec.Body.String(), `"model":"client-model"`)
	require.NotContains(t, rec.Body.String(), `"model":"upstream-model"`)
	require.Contains(t, rec.Body.String(), "data: [DONE]")
}

func TestForwardChatCompletionsDirect_ErrorFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(nil))

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-429"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          103,
		Name:        "cc-direct-429",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://upstream.example/v1"},
		Extra:       map[string]any{"openai_cc_direct_forward": true},
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, []byte(`{"model":"gpt-5","stream":false,"messages":[{"role":"user","content":"hi"}]}`), "", "")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "slow down")
}

func TestForwardChatCompletionsDirect_StreamingErrorEventFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(nil))

	upstreamBody := strings.Join([]string{
		`data: {"error":{"type":"server_error","message":"temporary outage"}}`,
		``,
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-stream-error"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          104,
		Name:        "cc-direct-stream-error",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://upstream.example/v1"},
		Extra:       map[string]any{"openai_cc_direct_forward": true},
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, []byte(`{"model":"gpt-5","stream":true,"messages":[{"role":"user","content":"hi"}]}`), "", "")
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.Contains(t, string(failoverErr.ResponseBody), "temporary outage")
	require.Empty(t, rec.Body.String())
}
