// Package anthropiccompat 提供 Anthropic-compatible 渠道的旁路封装层。
//
// 设计目标：
//   - 新增渠道时只需注册一个 ProviderSpec，不修改任何现有主链路代码。
//   - 官方 "anthropic" 平台绝不进入本包，继续走原 GatewayService 路径。
//   - 通用平台 "anthropic-compatible" 以及所有 "anthropic-*" 平台流量由本包统一处理。
package anthropiccompat

// AuthMode 定义上游请求的鉴权方式。
type AuthMode string

const (
	// AuthModeAPIKey 使用 Anthropic 标准的 x-api-key 请求头。
	AuthModeAPIKey AuthMode = "apikey"
	// AuthModeBearer 使用 Authorization: Bearer <token> 请求头。
	AuthModeBearer AuthMode = "bearer"
)

// ProviderSpec 描述一个自定义 Anthropic-compatible 渠道的最小能力集。
// 引入新渠道时，只需填写此结构体并调用 Register 注册，无需修改转发主流程。
type ProviderSpec struct {
	// Platform 是该渠道唯一的平台标识。
	// 内置平台通常使用 "anthropic-" 前缀；通用兜底平台使用 "anthropic-compatible"。
	Platform string

	// DisplayName 是管理后台展示的可读名称。
	DisplayName string

	// DefaultBaseURL 是账号未配置 base_url 时使用的默认上游地址。
	// 为空表示该渠道必须由账号显式配置 base_url。
	DefaultBaseURL string

	// MessagesPath 拼接在 base_url 后构成消息端点 URL。
	// 默认值为 "/v1/messages"。
	MessagesPath string

	// CountTokensPath 拼接在 base_url 后构成 count_tokens 端点 URL。
	// 为空表示该渠道不支持 count_tokens 接口。
	CountTokensPath string

	// AuthMode 决定上游鉴权头的设置方式。
	AuthMode AuthMode

	// AuthHeaderName 覆盖默认鉴权头名称。
	// AuthModeAPIKey 时默认为 "x-api-key"；AuthModeBearer 时忽略此字段，始终使用 "Authorization"。
	AuthHeaderName string

	// DefaultHeaders 是附加到每条上游请求的固定请求头。
	DefaultHeaders map[string]string

	// AnthropicVersion 是 anthropic-version 请求头的值。
	// 默认值为 "2023-06-01"。
	AnthropicVersion string

	// SupportsStreaming 标记该渠道是否支持 SSE 流式响应。
	SupportsStreaming bool

	// SupportsTools 标记该渠道是否支持 tool_use。
	SupportsTools bool

	// SupportsThinking 标记该渠道是否支持 thinking/推理块。
	SupportsThinking bool

	// SupportsCountTokens 标记该渠道是否支持 count_tokens 端点。
	SupportsCountTokens bool

	// DefaultModels 是未配置 model_mapping 时该渠道的默认可用模型列表。
	DefaultModels []string

	// NormalizeModelID 在转发前对请求模型 ID 做可选变换（账号级 model_mapping 优先）。
	// 为 nil 时直接透传原始模型 ID。
	NormalizeModelID func(model string) string

	// RequestMutator 在转发前可选地修改请求体。
	// 为 nil 时透传原始请求体。
	RequestMutator func(body []byte) []byte

	// ErrorParser 对渠道特有的错误响应做自定义解析，返回 (errorType, errorMessage)。
	// 为 nil 时使用通用 Anthropic 错误解析逻辑。
	ErrorParser func(statusCode int, body []byte) (string, string)

	// UsageParser 从缓冲的非流式响应体中提取用量信息，返回 (inputTokens, outputTokens)。
	// 为 nil 时使用标准 Anthropic usage 字段解析。
	UsageParser func(body []byte) (int64, int64)
}

// MessagesPath 返回实际生效的消息端点路径（带默认值回退）。
func (s *ProviderSpec) MessagesEndpointPath() string {
	if s.MessagesPath != "" {
		return s.MessagesPath
	}
	return "/v1/messages"
}

// AnthropicVersionHeader 返回实际生效的 anthropic-version 值（带默认值回退）。
func (s *ProviderSpec) AnthropicVersionHeader() string {
	if s.AnthropicVersion != "" {
		return s.AnthropicVersion
	}
	return "2023-06-01"
}

// ResolveAuthHeader 返回鉴权请求头名称和值格式。
// 对 AuthModeAPIKey：返回 (authHeaderName, token)；
// 对 AuthModeBearer：返回 ("Authorization", "Bearer "+token)。
func (s *ProviderSpec) ResolveAuthHeader(token string) (headerName, headerValue string) {
	switch s.AuthMode {
	case AuthModeBearer:
		return "Authorization", "Bearer " + token
	default:
		// 默认 API Key 模式
		name := s.AuthHeaderName
		if name == "" {
			name = "x-api-key"
		}
		return name, token
	}
}
