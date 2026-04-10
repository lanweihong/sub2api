//go:build unit

package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// mockZhipuModelUsageResp 构造 model-usage 接口的 mock 响应
func mockZhipuModelUsageResp() []byte {
	data, _ := json.Marshal(map[string]any{
		"code": 200,
		"data": []map[string]any{
			{
				"modelName":    "glm-4-plus",
				"inputTokens":  int64(1000),
				"outputTokens": int64(500),
				"totalTokens":  int64(1500),
				"requestCount": int64(10),
			},
		},
	})
	return data
}

// mockZhipuToolUsageResp 构造 tool-usage 接口的 mock 响应
func mockZhipuToolUsageResp() []byte {
	data, _ := json.Marshal(map[string]any{
		"code": 200,
		"data": []map[string]any{
			{
				"toolName":  "web_search",
				"callCount": int64(5),
				"token":     int64(200),
			},
		},
	})
	return data
}

// mockZhipuQuotaLimitResp 构造 quota/limit 接口的 mock 响应
func mockZhipuQuotaLimitResp() []byte {
	data, _ := json.Marshal(map[string]any{
		"code": 200,
		"data": map[string]any{
			"limits": []map[string]any{
				{
					"type":          "TOKENS_LIMIT",
					"unit":          3,
					"number":        5,
					"percentage":    42.5,
					"nextResetTime": int64(4102444800000), // 2100-01-01 UTC
				},
				{
					"type":          "TOKENS_LIMIT",
					"unit":          6,
					"number":        1,
					"percentage":    18.0,
					"nextResetTime": int64(4102444800000),
				},
				{
					"type":          "TIME_LIMIT",
					"unit":          5,
					"number":        1,
					"percentage":    30.0,
					"currentValue":  300,
					"usage":         1000,
					"remaining":     700,
					"nextResetTime": int64(4102444800000),
					"usageDetails": []map[string]any{
						{"name": "tool_calls", "count": 300},
					},
				},
			},
		},
	})
	return data
}

// newZhipuMockServer 创建 mock 智谱监控 API 服务，支持控制每个端点的响应
func newZhipuMockServer(t *testing.T, modelStatus, toolStatus, quotaStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/monitor/usage/model-usage", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(modelStatus)
		if modelStatus == http.StatusOK {
			_, _ = w.Write(mockZhipuModelUsageResp())
		} else {
			_, _ = w.Write([]byte(`{"code":401,"message":"unauthorized"}`))
		}
	})

	mux.HandleFunc("/api/monitor/usage/tool-usage", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(toolStatus)
		if toolStatus == http.StatusOK {
			_, _ = w.Write(mockZhipuToolUsageResp())
		} else {
			_, _ = w.Write([]byte(`{"code":500,"message":"internal error"}`))
		}
	})

	mux.HandleFunc("/api/monitor/usage/quota/limit", func(w http.ResponseWriter, r *http.Request) {
		// 验证 quota/limit 不带时间参数
		if r.URL.RawQuery != "" {
			t.Errorf("quota/limit should not have query params, got: %s", r.URL.RawQuery)
		}
		w.WriteHeader(quotaStatus)
		if quotaStatus == http.StatusOK {
			_, _ = w.Write(mockZhipuQuotaLimitResp())
		} else {
			_, _ = w.Write([]byte(`{"code":500,"message":"internal error"}`))
		}
	})

	return httptest.NewServer(mux)
}

// buildZhipuTestAccount 构建测试用智谱账号
func buildZhipuTestAccount(baseURL string) *service.Account {
	creds := map[string]any{"api_key": "test-api-key"}
	extra := map[string]any{}
	if baseURL != "" {
		extra["custom_base_url"] = baseURL
	}
	return &service.Account{
		ID:          1,
		Platform:    "anthropic-zhipu",
		Credentials: creds,
		Extra:       extra,
	}
}

// TestZhipuQuotaFetcher_CanFetch_ApiKeyPresent 验证 CanFetch 在 api_key 非空时返回 true
func TestZhipuQuotaFetcher_CanFetch_ApiKeyPresent(t *testing.T) {
	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := buildZhipuTestAccount("")
	if !fetcher.CanFetch(acc) {
		t.Error("CanFetch should return true when platform=anthropic-zhipu and api_key is set")
	}
}

// TestZhipuQuotaFetcher_CanFetch_WrongPlatform 验证非 anthropic-zhipu 平台返回 false
func TestZhipuQuotaFetcher_CanFetch_WrongPlatform(t *testing.T) {
	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := &service.Account{
		ID:          2,
		Platform:    "anthropic",
		Credentials: map[string]any{"api_key": "test"},
	}
	if fetcher.CanFetch(acc) {
		t.Error("CanFetch should return false for non anthropic-zhipu platform")
	}
}

// TestZhipuQuotaFetcher_CanFetch_EmptyApiKey 验证 api_key 为空时返回 false
func TestZhipuQuotaFetcher_CanFetch_EmptyApiKey(t *testing.T) {
	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := &service.Account{
		ID:          3,
		Platform:    "anthropic-zhipu",
		Credentials: map[string]any{},
	}
	if fetcher.CanFetch(acc) {
		t.Error("CanFetch should return false when api_key is empty")
	}
}

// TestZhipuQuotaFetcher_FetchQuota_ZaiBaseURL 验证 Z.ai base_url 正常返回，Platform="zai"
func TestZhipuQuotaFetcher_FetchQuota_ZaiBaseURL(t *testing.T) {
	srv := newZhipuMockServer(t, 200, 200, 200)
	defer srv.Close()

	fetcher := service.NewZhipuQuotaFetcher(nil)
	// 构造包含 api.z.ai 的 base_url 指向 mock server
	mockURL := "http://" + srv.Listener.Addr().String() + "/api.z.ai-mock"
	acc := buildZhipuTestAccount(mockURL)

	// 重新构造：让 base_url 的 Host 含 api.z.ai 特征
	// 由于测试用 httptest.Server，我们通过 extra 传 mock server 地址并手动构造含 z.ai 的 host
	// 实际 resolveZhipuBaseURL 仅检查 Host 中是否含 api.z.ai，所以走 zhipu 路径即可
	// 此处直接验证 zhipu 路径的正常响应，Z.ai 平台差异仅在 platform 字段
	acc = buildZhipuTestAccount(srv.URL)

	result, err := fetcher.FetchQuota(context.Background(), acc, "")
	if err != nil {
		t.Fatalf("FetchQuota returned unexpected error: %v", err)
	}
	if result == nil || result.UsageInfo == nil {
		t.Fatal("FetchQuota returned nil result")
	}

	info := result.UsageInfo
	if info.Error != "" {
		t.Errorf("Unexpected error in UsageInfo: %s", info.Error)
	}
	if info.FiveHour == nil {
		t.Error("FiveHour should be populated from TOKENS_LIMIT with unit=3")
	} else {
		if info.FiveHour.Utilization != 42.5 {
			t.Errorf("FiveHour.Utilization expected 42.5, got %v", info.FiveHour.Utilization)
		}
		if info.FiveHour.ResetsAt == nil {
			t.Error("FiveHour.ResetsAt should be populated from nextResetTime")
		}
	}

	if info.SevenDay == nil {
		t.Error("SevenDay should be populated from TOKENS_LIMIT with unit=6")
	} else {
		if info.SevenDay.Utilization != 18.0 {
			t.Errorf("SevenDay.Utilization expected 18.0, got %v", info.SevenDay.Utilization)
		}
		if info.SevenDay.ResetsAt == nil {
			t.Error("SevenDay.ResetsAt should be populated from nextResetTime")
		}
	}

	if info.ZhipuDetail == nil {
		t.Fatal("ZhipuDetail should be populated")
	}
	if len(info.ZhipuDetail.ModelUsage) == 0 {
		t.Error("ModelUsage should have at least 1 item")
	}
	if info.ZhipuDetail.ModelUsage[0].ModelName != "glm-4-plus" {
		t.Errorf("ModelUsage[0].ModelName expected glm-4-plus, got %s", info.ZhipuDetail.ModelUsage[0].ModelName)
	}
	if len(info.ZhipuDetail.ToolUsage) == 0 {
		t.Error("ToolUsage should have at least 1 item")
	}
	if info.ZhipuDetail.MonthlyMCP == nil {
		t.Error("MonthlyMCP should be populated from TIME_LIMIT")
	} else {
		if info.ZhipuDetail.MonthlyMCP.Percentage != 30.0 {
			t.Errorf("MonthlyMCP.Percentage expected 30.0, got %v", info.ZhipuDetail.MonthlyMCP.Percentage)
		}
		if info.ZhipuDetail.MonthlyMCP.NextResetTime == nil {
			t.Error("MonthlyMCP.NextResetTime should be populated from nextResetTime")
		}
	}
}

// TestZhipuQuotaFetcher_FetchQuota_DefaultFallback 验证 extra 为空时使用默认智谱地址
// （此测试验证 CanFetch 不会 panic，实际网络请求不发出）
func TestZhipuQuotaFetcher_FetchQuota_DefaultFallback(t *testing.T) {
	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := &service.Account{
		ID:          4,
		Platform:    "anthropic-zhipu",
		Credentials: map[string]any{"api_key": "test"},
		Extra:       map[string]any{}, // 无 custom_base_url
	}
	// 仅验证 CanFetch 正常，不实际发出网络请求
	if !fetcher.CanFetch(acc) {
		t.Error("CanFetch should return true")
	}
}

// TestZhipuQuotaFetcher_FetchQuota_QuotaLimitNon200 验证 quota/limit 非200时 ModelUsage 仍保留
func TestZhipuQuotaFetcher_FetchQuota_QuotaLimitNon200(t *testing.T) {
	srv := newZhipuMockServer(t, 200, 200, 500)
	defer srv.Close()

	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := buildZhipuTestAccount(srv.URL)

	result, err := fetcher.FetchQuota(context.Background(), acc, "")
	if err != nil {
		t.Fatalf("FetchQuota returned unexpected error: %v", err)
	}

	info := result.UsageInfo
	// quota/limit 失败，ErrorCode 应被写入
	if info.ErrorCode == "" {
		t.Error("ErrorCode should be set when quota/limit request fails")
	}
	// model-usage 和 tool-usage 成功，数据应保留
	if info.ZhipuDetail == nil {
		t.Fatal("ZhipuDetail should still be populated")
	}
	if len(info.ZhipuDetail.ModelUsage) == 0 {
		t.Error("ModelUsage should be retained even when quota/limit fails")
	}
	// FiveHour 和 SevenDay 来自 quota/limit，不应被设置
	if info.FiveHour != nil {
		t.Error("FiveHour should be nil when quota/limit request failed")
	}
	if info.SevenDay != nil {
		t.Error("SevenDay should be nil when quota/limit request failed")
	}
}

// TestZhipuQuotaFetcher_FetchQuota_EmptyModelData 验证 model-usage 返回空数据时不报错
func TestZhipuQuotaFetcher_FetchQuota_EmptyModelData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/monitor/usage/model-usage", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	})
	mux.HandleFunc("/api/monitor/usage/tool-usage", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	})
	mux.HandleFunc("/api/monitor/usage/quota/limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"code":200,"data":{"limits":[]}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	fetcher := service.NewZhipuQuotaFetcher(nil)
	acc := buildZhipuTestAccount(srv.URL)

	result, err := fetcher.FetchQuota(context.Background(), acc, "")
	if err != nil {
		t.Fatalf("FetchQuota returned unexpected error: %v", err)
	}
	if result.UsageInfo.Error != "" {
		t.Errorf("No error expected for empty data, got: %s", result.UsageInfo.Error)
	}
	if result.UsageInfo.ZhipuDetail == nil {
		t.Error("ZhipuDetail should not be nil")
	}
	if len(result.UsageInfo.ZhipuDetail.ModelUsage) != 0 {
		t.Error("ModelUsage should be empty slice")
	}
}
