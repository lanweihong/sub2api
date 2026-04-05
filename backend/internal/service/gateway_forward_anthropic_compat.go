package service

// ForwardAnthropicCompat 处理 Anthropic-compatible 渠道（anthropic-compatible 与 anthropic-* 平台）的请求转发。
//
// 设计原则：
//   - 所有前置逻辑（鉴权、并发控制、粘性会话、账号选择）由 GatewayHandler 复用现有链路完成
//   - 本方法仅负责"真正发上游请求与解析响应"，与官方 Anthropic 的 OAuth/TLS/ClaudeCode 逻辑完全隔离
//   - 通过 anthropiccompat.Resolve 从 Provider Registry 获取渠道规格，无需修改本文件即可扩展新渠道
//
// 支持的渠道：anthropic-compatible、anthropic-zhipu、anthropic-kimi、anthropic-minimax、anthropic-qwen、anthropic-mimo 等

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
	// 导入所有 provider 以触发 init() 自动注册
	_ "github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat/providers"
	"github.com/gin-gonic/gin"
)

// ForwardAnthropicCompat 转发 Anthropic-compatible 平台的请求。
// 签名与 Forward 保持一致，便于 GatewayHandler 在分流点直接替换调用。
func (s *GatewayService) ForwardAnthropicCompat(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *ParsedRequest,
) (*ForwardResult, error) {
	startTime := time.Now()

	// 1. 从 Provider Registry 解析渠道规格
	// 如果渠道未注册（配置错误），提前拒绝而非走官方 Anthropic 链路
	spec, ok := anthropiccompat.Resolve(account.Platform)
	if !ok {
		return nil, writeAnthropicCompatError(c, http.StatusBadGateway,
			"api_error", fmt.Sprintf("未注册的 Anthropic-compatible 渠道平台: %s", account.Platform))
	}
	if requiresExplicitAnthropicCompatBaseURL(account.GetCredential("base_url"), spec) {
		return nil, writeAnthropicCompatError(c, http.StatusBadGateway,
			"api_error", "该 Anthropic-compatible 渠道必须设置 base_url")
	}

	// 2. 仅支持 APIKey 类型账号（国内厂商均使用 API Key 认证）
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, &UpstreamFailoverError{
			StatusCode:   http.StatusUnauthorized,
			ResponseBody: []byte(`{"type":"error","error":{"type":"authentication_error","message":"Failed to get API key"}}`),
		}
	}

	// 3. 模型映射：优先使用账号级 model_mapping，其次使用 spec.NormalizeModelID
	originalModel := parsed.Model
	mappedModel := account.GetMappedModel(originalModel)
	if spec.NormalizeModelID != nil {
		mappedModel = spec.NormalizeModelID(mappedModel)
	}

	// 4. 构造上游 URL：优先使用账号配置的 base_url，否则使用 spec.DefaultBaseURL
	baseURL := resolveAnthropicCompatBaseURL(account.GetCredential("base_url"), spec)
	validatedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, writeAnthropicCompatError(c, http.StatusBadGateway,
			"api_error", fmt.Sprintf("无效的 base_url: %s", err.Error()))
	}
	targetURL := strings.TrimRight(validatedBaseURL, "/") + spec.MessagesEndpointPath()

	// 5. 构造请求体（可选 RequestMutator 做渠道差异化修改）
	body := parsed.Body
	if spec.RequestMutator != nil {
		body = spec.RequestMutator(body)
	}

	// 6. 替换请求体中的模型字段为映射后的模型名
	if mappedModel != originalModel {
		body = s.replaceModelInBody(body, mappedModel)
		logger.LegacyPrintf("service.anthropic_compat",
			"[AnthropicCompat] 模型映射: %s -> %s (账号: %s, 渠道: %s)",
			originalModel, mappedModel, account.Name, account.Platform)
	}

	// 7. 构造 HTTP 请求
	req, err := buildAnthropicCompatRequest(ctx, targetURL, body, token, spec, c)
	if err != nil {
		return nil, writeAnthropicCompatError(c, http.StatusBadGateway,
			"api_error", fmt.Sprintf("构造上游请求失败: %s", err.Error()))
	}

	// 8. 代理配置（与 Forward 保持一致）
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 9. 发送请求（不使用 TLS 指纹，国内厂商无此需求）
	setOpsUpstreamRequestBody(c, body)
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:    account.Platform,
			AccountID:   account.ID,
			AccountName: account.Name,
			Kind:        "request_error",
			Message:     safeErr,
		})
		return nil, writeAnthropicCompatError(c, http.StatusBadGateway, "api_error", "上游请求发送失败")
	}
	defer func() { _ = resp.Body.Close() }()

	// 10. 处理上游错误响应（支持 failover / 重试信号）
	if resp.StatusCode >= 400 {
		return s.handleAnthropicCompatErrorResponse(ctx, c, account, resp, spec)
	}

	// 11. 分发流式或非流式响应处理
	reqStream := parsed.Stream
	if reqStream {
		sr, streamErr := s.handleStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account, startTime, mappedModel)
		if streamErr != nil && (sr == nil || (sr.usage == nil && !sr.clientDisconnect)) {
			logger.LegacyPrintf("service.anthropic_compat",
				"[AnthropicCompat] 流式响应处理错误: %v (账号: %d)", streamErr, account.ID)
		}
		result := &ForwardResult{
			RequestID: resp.Header.Get("x-request-id"),
			Model:     originalModel,
			Stream:    true,
			Duration:  time.Since(startTime),
		}
		if sr != nil {
			if sr.usage != nil {
				result.Usage = *sr.usage
			}
			result.FirstTokenMs = sr.firstTokenMs
			result.ClientDisconnect = sr.clientDisconnect
		}
		if mappedModel != originalModel {
			result.UpstreamModel = mappedModel
		}
		return result, streamErr
	}

	// 非流式响应
	usage, handleErr := s.handleNonStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account)
	if handleErr != nil {
		return nil, handleErr
	}
	result := &ForwardResult{
		RequestID: resp.Header.Get("x-request-id"),
		Model:     originalModel,
		Stream:    false,
		Duration:  time.Since(startTime),
	}
	if usage != nil {
		result.Usage = *usage
	}
	if mappedModel != originalModel {
		result.UpstreamModel = mappedModel
	}
	return result, nil
}

// buildAnthropicCompatRequest 构造发往自定义 Anthropic-compatible 渠道的 HTTP 请求。
// 不注入 OAuth、TLS 指纹、Claude Code 相关头，仅添加必要的鉴权和渠道默认头。
func buildAnthropicCompatRequest(
	ctx context.Context,
	targetURL string,
	body []byte,
	token string,
	spec *anthropiccompat.ProviderSpec,
	c *gin.Context,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 设置 Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 设置 anthropic-version（兼容 Anthropic 协议的渠道需要此头）
	req.Header.Set("anthropic-version", spec.AnthropicVersionHeader())

	// 设置鉴权头（支持 x-api-key 和 Authorization: Bearer 两种模式）
	authHeaderName, authHeaderValue := spec.ResolveAuthHeader(token)
	req.Header.Set(authHeaderName, authHeaderValue)

	// 透传客户端的 anthropic-beta 头（部分渠道支持 beta 功能）
	if c != nil && c.Request != nil {
		if betaHeader := c.GetHeader("anthropic-beta"); betaHeader != "" {
			req.Header.Set("anthropic-beta", betaHeader)
		}
	}

	// 追加渠道默认头（如 Kimi 的特定功能开关）
	for k, v := range spec.DefaultHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

// handleAnthropicCompatErrorResponse 处理自定义 Anthropic-compatible 渠道返回的错误响应。
// 对可切换账号的错误（401/429/5xx）返回 UpstreamFailoverError 以触发 handler 层的 failover 机制。
func (s *GatewayService) handleAnthropicCompatErrorResponse(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	resp *http.Response,
	spec *anthropiccompat.ProviderSpec,
) (*ForwardResult, error) {
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	_ = resp.Body.Close()

	// 使用渠道自定义解析（若有），否则使用通用 Anthropic 错误提取
	var errMsg string
	if spec.ErrorParser != nil {
		_, errMsg = spec.ErrorParser(resp.StatusCode, respBody)
	} else {
		errMsg = strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		errMsg = sanitizeUpstreamErrorMessage(errMsg)
	}

	logger.LegacyPrintf("service.anthropic_compat",
		"[AnthropicCompat] 上游错误: 账号=%d(%s) 渠道=%s 状态=%d 消息=%s",
		account.ID, account.Name, account.Platform, resp.StatusCode, errMsg)

	// 触发 failover 的状态码（401、429、5xx）
	if s.shouldFailoverUpstreamError(resp.StatusCode) {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Kind:               "failover",
			Message:            errMsg,
		})
		return nil, &UpstreamFailoverError{
			StatusCode:   resp.StatusCode,
			ResponseBody: respBody,
		}
	}

	// 不可 failover 的错误（如 400 参数错误），直接透传给客户端
	setOpsUpstreamError(c, resp.StatusCode, errMsg, "")
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	return nil, fmt.Errorf("上游错误 %d: %s", resp.StatusCode, errMsg)
}

// writeAnthropicCompatError 向客户端写入 Anthropic 风格的错误响应并返回 error。
func writeAnthropicCompatError(c *gin.Context, statusCode int, errType, message string) error {
	if c != nil {
		c.JSON(statusCode, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
	}
	return fmt.Errorf("anthropic_compat error %d: %s", statusCode, message)
}
