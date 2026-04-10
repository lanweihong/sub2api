package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	httpclient "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

const (
	zhipuDefaultBaseURL = "https://open.bigmodel.cn"

	zhipuModelUsagePath = "/api/monitor/usage/model-usage"
	zhipuToolUsagePath  = "/api/monitor/usage/tool-usage"
	zhipuQuotaLimitPath = "/api/monitor/usage/quota/limit"

	zhipuHTTPTimeout = 10 * time.Second
)

// ZhipuQuotaFetcher 从智谱/Z.ai 监控 API 获取账号用量额度
type ZhipuQuotaFetcher struct {
	proxyRepo ProxyRepository
}

// NewZhipuQuotaFetcher 创建 ZhipuQuotaFetcher
func NewZhipuQuotaFetcher(proxyRepo ProxyRepository) *ZhipuQuotaFetcher {
	return &ZhipuQuotaFetcher{proxyRepo: proxyRepo}
}

// CanFetch 检查是否可以获取此账户的额度
func (f *ZhipuQuotaFetcher) CanFetch(account *Account) bool {
	if account.Platform != "anthropic-zhipu" {
		return false
	}
	return account.GetCredential("api_key") != ""
}

// GetProxyURL 获取账户配置的代理 URL（复用 Antigravity 相同模式）
func (f *ZhipuQuotaFetcher) GetProxyURL(ctx context.Context, account *Account) string {
	if account.ProxyID == nil || f.proxyRepo == nil {
		return ""
	}
	proxy, err := f.proxyRepo.GetByID(ctx, *account.ProxyID)
	if err != nil || proxy == nil {
		return ""
	}
	return proxy.URL()
}

// FetchQuota 获取智谱/Z.ai 账户用量信息
// 串行调用三路监控 API，任一子请求失败时保留已成功部分并写入 Error/ErrorCode
func (f *ZhipuQuotaFetcher) FetchQuota(ctx context.Context, account *Account, proxyURL string) (*QuotaResult, error) {
	apiKey := account.GetCredential("api_key")
	baseURL, platform := resolveZhipuBaseURL(account)
	startTime, endTime := buildZhipuTimeWindow()

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: proxyURL,
		Timeout:  zhipuHTTPTimeout,
	})
	if err != nil {
		now := time.Now()
		return &QuotaResult{
			UsageInfo: &UsageInfo{
				UpdatedAt: &now,
				ErrorCode: errorCodeNetworkError,
				Error:     fmt.Sprintf("failed to create http client: %v", err),
			},
		}, nil
	}

	detail := &ZhipuUsageDetail{
		Platform:  platform,
		StartTime: startTime,
		EndTime:   endTime,
	}

	info := &UsageInfo{}
	var firstErr string
	var firstErrCode string

	// 1. 请求 model-usage
	modelData, err := doZhipuHTTPGet(ctx, client, baseURL+zhipuModelUsagePath, apiKey, startTime, endTime)
	if err != nil {
		firstErr = fmt.Sprintf("model-usage: %v", err)
		firstErrCode = classifyZhipuError(err)
	} else {
		detail.ModelUsage = parseZhipuModelUsage(modelData)
	}

	// 2. 请求 tool-usage
	toolData, err := doZhipuHTTPGet(ctx, client, baseURL+zhipuToolUsagePath, apiKey, startTime, endTime)
	if err != nil {
		if firstErr == "" {
			firstErr = fmt.Sprintf("tool-usage: %v", err)
			firstErrCode = classifyZhipuError(err)
		}
	} else {
		detail.ToolUsage = parseZhipuToolUsage(toolData)
	}

	// 3. 请求 quota/limit（不传时间参数）
	quotaData, err := doZhipuHTTPGet(ctx, client, baseURL+zhipuQuotaLimitPath, apiKey, "", "")
	if err != nil {
		if firstErr == "" {
			firstErr = fmt.Sprintf("quota/limit: %v", err)
			firstErrCode = classifyZhipuError(err)
		}
	} else {
		parseZhipuQuotaLimit(quotaData, info, detail)
	}

	now := time.Now()
	info.UpdatedAt = &now
	info.ZhipuDetail = detail

	// 有错误但不整体 panic，降级返回已成功部分
	if firstErr != "" {
		info.Error = firstErr
		info.ErrorCode = firstErrCode
	}

	return &QuotaResult{UsageInfo: info}, nil
}

// resolveZhipuBaseURL 根据账号配置决定 base URL 与平台标识
// 优先读 extra.custom_base_url，否则 fallback 到智谱默认地址
func resolveZhipuBaseURL(account *Account) (baseURL, platform string) {
	raw := account.GetExtraString("custom_base_url")
	if raw == "" {
		raw = account.GetCredential("base_url")
	}

	if raw != "" {
		parsed, err := url.Parse(raw)
		if err == nil && parsed.Host != "" {
			base := parsed.Scheme + "://" + parsed.Host
			if strings.Contains(parsed.Host, "api.z.ai") {
				return base, "zai"
			}
			return base, "zhipu"
		}
	}

	return zhipuDefaultBaseURL, "zhipu"
}

// buildZhipuTimeWindow 生成查询时间窗口字符串（本地时区，与原始脚本逻辑一致）
// 起始：昨日当前小时整点（HH:00:00）
// 结束：今日当前小时末尾（HH:59:59）
func buildZhipuTimeWindow() (startTime, endTime string) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day()-1, now.Hour(), 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 59, 59, 0, now.Location())
	return formatZhipuTime(start), formatZhipuTime(end)
}

// formatZhipuTime 格式化为智谱接口期望的时间格式 "yyyy-MM-dd HH:mm:ss"
func formatZhipuTime(t time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

// doZhipuHTTPGet 执行单次 GET 请求并返回原始响应体字节
// startTime/endTime 为空时不拼接查询参数（quota/limit 接口）
func doZhipuHTTPGet(ctx context.Context, client *http.Client, rawURL, apiKey, startTime, endTime string) ([]byte, error) {
	reqURL := rawURL
	if startTime != "" && endTime != "" {
		reqURL = rawURL + "?startTime=" + url.QueryEscape(startTime) + "&endTime=" + url.QueryEscape(endTime)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Accept-Language", "en-US,en")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, &zhipuHTTPError{StatusCode: resp.StatusCode, Body: preview}
	}

	return body, nil
}

// zhipuHTTPError HTTP 非 200 错误
type zhipuHTTPError struct {
	StatusCode int
	Body       string
}

func (e *zhipuHTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// classifyZhipuError 将请求错误映射为机器可读错误码
func classifyZhipuError(err error) string {
	if err == nil {
		return ""
	}
	var httpErr *zhipuHTTPError
	if ok := isZhipuHTTPError(err, &httpErr); ok {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized:
			return errorCodeUnauthenticated
		case http.StatusForbidden:
			return errorCodeForbidden
		case http.StatusTooManyRequests:
			return errorCodeRateLimited
		}
	}
	return errorCodeNetworkError
}

// isZhipuHTTPError 类型断言辅助
func isZhipuHTTPError(err error, target **zhipuHTTPError) bool {
	if e, ok := err.(*zhipuHTTPError); ok {
		*target = e
		return true
	}
	return false
}

// parseZhipuModelUsage 解析 model-usage 接口返回的 JSON 数据（旧扁平数组格式，仅 FetchQuota 缓存流程使用）
// 接口返回格式：{"code":200,"data":[{...},...]}
func parseZhipuModelUsage(data []byte) []ZhipuModelUsageItem {
	var resp struct {
		Data []ZhipuModelUsageItem `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	return resp.Data
}

// parseZhipuModelUsageResponse 解析 model-usage 接口返回的时序对象格式
// 接口返回格式：{"code":200,"data":{"x_time":[...],"modelDataList":[...],...}}
func parseZhipuModelUsageResponse(data []byte) (*ZhipuModelUsageResponse, error) {
	var resp struct {
		Code int                      `json:"code"`
		Data *ZhipuModelUsageResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal model-usage response: %w", err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("model-usage response data is nil")
	}
	return resp.Data, nil
}

// parseZhipuToolUsage 解析 tool-usage 接口返回的 JSON 数据（旧扁平数组格式，仅 FetchQuota 缓存流程使用）
func parseZhipuToolUsage(data []byte) []ZhipuToolUsageItem {
	var resp struct {
		Data []ZhipuToolUsageItem `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	return resp.Data
}

// parseZhipuToolUsageResponse 解析 tool-usage 接口返回的时序对象格式
// 接口返回格式：{"code":200,"data":{"x_time":[...],"toolDataList":[...],...}}
func parseZhipuToolUsageResponse(data []byte) (*ZhipuToolUsageResponse, error) {
	var resp struct {
		Code int                     `json:"code"`
		Data *ZhipuToolUsageResponse `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal tool-usage response: %w", err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("tool-usage response data is nil")
	}
	return resp.Data, nil
}

// parseZhipuQuotaLimit 解析 quota/limit 接口响应，填充 UsageInfo 与 ZhipuUsageDetail
// TOKENS_LIMIT 按 unit 区分：unit=3 (小时) → FiveHour, unit=6 (周) → SevenDay
// TIME_LIMIT → ZhipuUsageDetail.MonthlyMCP
func parseZhipuQuotaLimit(data []byte, info *UsageInfo, detail *ZhipuUsageDetail) {
	var resp struct {
		Data zhipuQuotaLimitData `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}

	for _, item := range resp.Data.Limits {
		switch item.Type {
		case "TOKENS_LIMIT":
			progress := &UsageProgress{
				Utilization: item.Percentage,
			}
			if item.NextResetTime > 0 {
				resetAt := epochMillisToTime(item.NextResetTime)
				progress.ResetsAt = &resetAt
				progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
			}
			// unit=3 → 小时级（如 5 小时）; unit=6 → 周级（如 1 周）
			switch item.Unit {
			case 6:
				info.SevenDay = progress
			default:
				info.FiveHour = progress
			}
		case "TIME_LIMIT":
			mcp := &ZhipuMonthlyQuota{
				Percentage:   item.Percentage,
				CurrentUsage: item.CurrentValue,
				Total:        item.Usage,
				Remaining:    item.Remaining,
				UsageDetails: item.UsageDetails,
			}
			if item.NextResetTime > 0 {
				resetAt := epochMillisToTime(item.NextResetTime)
				mcp.NextResetTime = &resetAt
			}
			detail.MonthlyMCP = mcp
		}
	}
}

// epochMillisToTime 将 epoch 毫秒转换为 time.Time
func epochMillisToTime(ms int64) time.Time {
	return time.Unix(ms/1000, (ms%1000)*int64(time.Millisecond))
}

// computeZhipuPeriod 根据 period 参数计算查询时间范围
// today: 今日 00:00:00 → 当前时间 HH:59:59
// 7d:   7天前 00:00:00 → 当前时间 HH:59:59
// 30d:  30天前 00:00:00 → 当前时间 HH:59:59
func computeZhipuPeriod(period string) (startTime, endTime string) {
	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 59, 59, 0, now.Location())

	var start time.Time
	switch period {
	case "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "7d":
		start = time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, now.Location())
	default: // "30d"
		start = time.Date(now.Year(), now.Month(), now.Day()-29, 0, 0, 0, 0, now.Location())
	}
	return formatZhipuTime(start), formatZhipuTime(end)
}

// FetchUsage 按指定时间范围获取智谱模型或工具用量（管理员按需查询，不走缓存）
func (f *ZhipuQuotaFetcher) FetchUsage(ctx context.Context, account *Account, proxyURL, usageType, startTime, endTime string) (any, error) {
	apiKey := account.GetCredential("api_key")
	baseURL, _ := resolveZhipuBaseURL(account)

	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL: proxyURL,
		Timeout:  zhipuHTTPTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	var path string
	switch usageType {
	case "model":
		path = zhipuModelUsagePath
	case "tool":
		path = zhipuToolUsagePath
	default:
		return nil, fmt.Errorf("invalid usage type: %s", usageType)
	}

	data, err := doZhipuHTTPGet(ctx, client, baseURL+path, apiKey, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", usageType, err)
	}

	switch usageType {
	case "model":
		return parseZhipuModelUsageResponse(data)
	default:
		return parseZhipuToolUsageResponse(data)
	}
}
