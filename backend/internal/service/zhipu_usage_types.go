package service

import "time"

// ZhipuUsageDetail 智谱/Z.ai 账号的用量详情（多路聚合结果）
type ZhipuUsageDetail struct {
	// Platform 平台标识："zai" 或 "zhipu"
	Platform  string `json:"platform"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	// ModelUsage 模型用量明细（按模型分组）
	ModelUsage []ZhipuModelUsageItem `json:"model_usage,omitempty"`

	// ToolUsage 工具调用用量明细
	ToolUsage []ZhipuToolUsageItem `json:"tool_usage,omitempty"`

	// MonthlyMCP 月度 MCP 配额（来自 quota/limit 接口的 TIME_LIMIT 项）
	MonthlyMCP *ZhipuMonthlyQuota `json:"monthly_mcp,omitempty"`
}

// ZhipuModelUsageItem 单个模型的用量行
// 字段按智谱 model-usage 接口响应原样映射，保留 any 容忍结构微差
type ZhipuModelUsageItem struct {
	ModelName    string `json:"modelName,omitempty"`
	InputTokens  int64  `json:"inputTokens,omitempty"`
	OutputTokens int64  `json:"outputTokens,omitempty"`
	TotalTokens  int64  `json:"totalTokens,omitempty"`
	RequestCount int64  `json:"requestCount,omitempty"`
	// 以下为智谱接口可能返回的扩展字段，使用 omitempty 容忍缺失
	CacheReadTokens  int64  `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int64  `json:"cacheWriteTokens,omitempty"`
	Date             string `json:"date,omitempty"`
}

// ZhipuModelUsageResponse 模型用量 API (/api/monitor/usage/model-usage) 的实际 data 对象
// 响应为时序格式，包含日期轴 x_time 和按模型分组的每日用量数组
type ZhipuModelUsageResponse struct {
	XTime            []string                `json:"x_time"`
	ModelDataList    []ZhipuModelDataItem    `json:"modelDataList"`
	ModelSummaryList []ZhipuModelSummaryItem `json:"modelSummaryList,omitempty"`
	TotalUsage       *ZhipuModelTotalUsage   `json:"totalUsage,omitempty"`
	Granularity      string                  `json:"granularity,omitempty"`
	ModelCallCount   []int64                 `json:"modelCallCount,omitempty"`
	TokensUsage      []int64                 `json:"tokensUsage,omitempty"`
}

// ZhipuModelDataItem modelDataList 中的单个模型时序数据
type ZhipuModelDataItem struct {
	ModelName   string  `json:"modelName"`
	SortOrder   int     `json:"sortOrder"`
	TokensUsage []int64 `json:"tokensUsage"`
	TotalTokens int64   `json:"totalTokens"`
}

// ZhipuModelSummaryItem modelSummaryList 中的单个模型汇总
type ZhipuModelSummaryItem struct {
	ModelName   string `json:"modelName"`
	TotalTokens int64  `json:"totalTokens"`
	SortOrder   int    `json:"sortOrder"`
}

// ZhipuModelTotalUsage totalUsage 汇总信息
type ZhipuModelTotalUsage struct {
	TotalModelCallCount int64                   `json:"totalModelCallCount"`
	TotalTokensUsage    int64                   `json:"totalTokensUsage"`
	ModelSummaryList    []ZhipuModelSummaryItem `json:"modelSummaryList,omitempty"`
}

// ZhipuToolUsageItem 单个工具调用的用量行
type ZhipuToolUsageItem struct {
	ToolName  string `json:"toolName,omitempty"`
	CallCount int64  `json:"callCount,omitempty"`
	// 以下为智谱接口可能返回的扩展字段
	Token int64  `json:"token,omitempty"`
	Date  string `json:"date,omitempty"`
}

// ZhipuToolUsageResponse 工具用量 API (/api/monitor/usage/tool-usage) 的实际 data 对象
// 响应为时序格式，包含日期轴 x_time 和内置工具计数数组
type ZhipuToolUsageResponse struct {
	XTime              []string                `json:"x_time"`
	ToolDataList       []ZhipuToolDataItem     `json:"toolDataList"`
	ToolSummaryList    []ZhipuToolSummaryItem  `json:"toolSummaryList,omitempty"`
	TotalUsage         *ZhipuToolTotalUsage    `json:"totalUsage,omitempty"`
	Granularity        string                  `json:"granularity,omitempty"`
	NetworkSearchCount []int64                 `json:"networkSearchCount,omitempty"`
	WebReadMcpCount    []int64                 `json:"webReadMcpCount,omitempty"`
	ZreadMcpCount      []int64                 `json:"zreadMcpCount,omitempty"`
}

// ZhipuToolDataItem toolDataList 中的单个工具时序数据
type ZhipuToolDataItem struct {
	ToolName       string  `json:"toolName"`
	SortOrder      int     `json:"sortOrder"`
	CallCount      []int64 `json:"callCount"`
	TotalCallCount int64   `json:"totalCallCount"`
}

// ZhipuToolSummaryItem toolSummaryList 中的单个工具汇总
type ZhipuToolSummaryItem struct {
	ToolName       string `json:"toolName"`
	TotalCallCount int64  `json:"totalCallCount"`
	SortOrder      int    `json:"sortOrder"`
}

// ZhipuToolTotalUsage 工具用量 totalUsage 汇总
type ZhipuToolTotalUsage struct {
	TotalNetworkSearchCount int64                  `json:"totalNetworkSearchCount"`
	TotalWebReadMcpCount    int64                  `json:"totalWebReadMcpCount"`
	TotalZreadMcpCount      int64                  `json:"totalZreadMcpCount"`
	TotalSearchMcpCount     int64                  `json:"totalSearchMcpCount"`
	ToolDetails             []map[string]any       `json:"toolDetails,omitempty"`
	ToolSummaryList         []ZhipuToolSummaryItem `json:"toolSummaryList,omitempty"`
}

// ZhipuMonthlyQuota 月度 MCP 配额信息（来自 quota/limit 接口的 TIME_LIMIT 项）
type ZhipuMonthlyQuota struct {
	// Percentage 已使用百分比 0-100
	Percentage float64 `json:"percentage"`
	// CurrentUsage 当月已用量（单位由上游决定，通常为次数）
	CurrentUsage any `json:"currentUsage,omitempty"`
	// Total 月度总配额
	Total any `json:"total,omitempty"`
	// Remaining 剩余配额
	Remaining any `json:"remaining,omitempty"`
	// NextResetTime 下次重置时间
	NextResetTime *time.Time `json:"next_reset_time,omitempty"`
	// UsageDetails 用量明细列表（原始结构，容忍字段变化）
	UsageDetails []map[string]any `json:"usageDetails,omitempty"`
}

// zhipuRawAPIResponse 内部 HTTP 响应解析结构（unexported）
// 智谱/Z.ai 监控接口统一响应格式：{"code": 200, "data": {...}}
type zhipuRawAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message,omitempty"`
	Data    zhipuRawDataAny `json:"data"`
}

// zhipuRawDataAny 可以是对象也可以是数组，先用 json.RawMessage 延迟解析
type zhipuRawDataAny = any

// zhipuQuotaLimitData quota/limit 接口的 data 字段
type zhipuQuotaLimitData struct {
	Limits []zhipuQuotaLimitItem `json:"limits,omitempty"`
}

// zhipuQuotaLimitItem quota/limit 接口中 limits 数组的单项
type zhipuQuotaLimitItem struct {
	Type          string           `json:"type"`
	Unit          int              `json:"unit"`          // 时间单位：3=小时, 5=月, 6=周
	Number        int              `json:"number"`        // 数量：如 5 → 5 小时, 1 → 1 周
	Percentage    float64          `json:"percentage"`
	NextResetTime int64            `json:"nextResetTime"` // 重置时间（epoch 毫秒）
	CurrentValue  any              `json:"currentValue,omitempty"`
	Usage         any              `json:"usage,omitempty"`
	Remaining     any              `json:"remaining,omitempty"`
	UsageDetails  []map[string]any `json:"usageDetails,omitempty"`
}
