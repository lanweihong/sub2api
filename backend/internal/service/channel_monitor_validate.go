package service

import (
	"context"
	"net/url"
	"strings"
)

// 渠道监控参数校验与归一化辅助函数。
// 校验失败一律返回 channel_monitor_const.go 中预定义的 Err* 错误，错误信息不含具体 IP/hostname，避免泄露内网拓扑。

// validateProvider 校验 provider 字符串。
// 唯一来源于 providerAdapters：新增 provider 只需要在 channel_monitor_checker.go 注册 adapter。
func validateProvider(p string) error {
	if !isSupportedProvider(p) {
		return ErrChannelMonitorInvalidProvider
	}
	return nil
}

// validateAPIMode 校验 provider 与 api_mode 的组合。
// responses 只对 OpenAI 有意义；其它 provider 使用 chat_completions 作为默认占位。
func validateAPIMode(provider, apiMode string) error {
	apiMode = defaultAPIMode(apiMode)
	switch apiMode {
	case MonitorAPIModeChatCompletions:
		return nil
	case MonitorAPIModeResponses:
		if provider == "" || provider == MonitorProviderOpenAI {
			return nil
		}
		return ErrChannelMonitorInvalidAPIMode
	default:
		return ErrChannelMonitorInvalidAPIMode
	}
}

// validateInterval 校验 interval_seconds 范围。
func validateInterval(sec int) error {
	if sec < monitorMinIntervalSeconds || sec > monitorMaxIntervalSeconds {
		return ErrChannelMonitorInvalidInterval
	}
	return nil
}

// validateEndpointForProvider 按 provider 校验 endpoint。
// Anthropic-compatible 平台的默认 Base URL 可能带 path 前缀（如 /api/anthropic），
// 因此允许 path，但仍拒绝 query/fragment/http/私网地址。
func validateEndpointForProvider(provider, ep string) error {
	return validateEndpointWithPathPolicy(ep, IsAnthropicCompatPlatform(provider))
}

func validateEndpointWithPathPolicy(ep string, allowPath bool) error {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ErrChannelMonitorInvalidEndpoint
	}
	u, err := url.Parse(ep)
	if err != nil {
		return ErrChannelMonitorInvalidEndpoint
	}
	if u.Scheme != "https" {
		return ErrChannelMonitorEndpointScheme
	}
	if u.Host == "" {
		return ErrChannelMonitorInvalidEndpoint
	}
	if !allowPath && u.Path != "" && u.Path != "/" {
		return ErrChannelMonitorEndpointPath
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return ErrChannelMonitorEndpointPath
	}

	hostname := u.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), monitorEndpointResolveTimeout)
	defer cancel()
	blocked, err := isPrivateOrLoopbackHost(ctx, hostname)
	if err != nil {
		return ErrChannelMonitorEndpointUnreachable
	}
	if blocked {
		return ErrChannelMonitorEndpointPrivate
	}
	return nil
}

// normalizeEndpoint 去除前后空白与末尾 `/`。
// 官方 provider 存储为 origin；Anthropic-compatible provider 允许保留 base path 前缀。
func normalizeEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	ep = strings.TrimRight(ep, "/")
	return ep
}

// normalizeModels 去除空白、重复模型名。保留输入顺序（map 的迭代顺序无关）。
func normalizeModels(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, m := range in {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

// defaultAPIMode 空串归一为 chat_completions，保证历史数据与旧客户端兼容。
func defaultAPIMode(apiMode string) string {
	if strings.TrimSpace(apiMode) == "" {
		return MonitorAPIModeChatCompletions
	}
	return strings.TrimSpace(apiMode)
}
