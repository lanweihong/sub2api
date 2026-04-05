//go:build unit

// Package anthropiccompat_test 包含 Provider Registry 的单元测试。
package anthropiccompat_test

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat"
	// 触发所有内置 provider 的 init 注册
	_ "github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat/providers"
)

// TestResolveRegisteredProviders 验证所有内置 provider 均已正确注册且可解析。
func TestResolveRegisteredProviders(t *testing.T) {
	expectedPlatforms := []string{
		"anthropic-compatible",
		"anthropic-zhipu",
		"anthropic-kimi",
		"anthropic-minimax",
		"anthropic-qwen",
		"anthropic-mimo",
	}

	for _, platform := range expectedPlatforms {
		t.Run(platform, func(t *testing.T) {
			spec, ok := anthropiccompat.Resolve(platform)
			if !ok {
				t.Fatalf("平台 %q 未注册，期望已注册", platform)
			}
			if spec == nil {
				t.Fatalf("平台 %q Resolve 返回了 nil spec", platform)
			}
			if spec.Platform != platform {
				t.Errorf("spec.Platform = %q，期望 %q", spec.Platform, platform)
			}
			if spec.DisplayName == "" {
				t.Errorf("平台 %q 的 DisplayName 为空", platform)
			}
			if platform != "anthropic-compatible" && spec.DefaultBaseURL == "" {
				t.Errorf("平台 %q 的 DefaultBaseURL 为空", platform)
			}
			if platform != "anthropic-compatible" && len(spec.DefaultModels) == 0 {
				t.Errorf("平台 %q 的 DefaultModels 为空", platform)
			}
			if platform == "anthropic-compatible" {
				if spec.Platform != "anthropic-compatible" {
					t.Errorf("平台 %q 的 spec.Platform = %q", platform, spec.Platform)
				}
			} else if !strings.HasPrefix(spec.Platform, "anthropic-") {
				t.Errorf("平台 %q 不以 'anthropic-' 为前缀", platform)
			}
		})
	}
}

// TestResolveUnregisteredPlatform 验证未注册的 platform 返回 false。
func TestResolveUnregisteredPlatform(t *testing.T) {
	_, ok := anthropiccompat.Resolve("anthropic-nonexistent-xyz")
	if ok {
		t.Fatal("期望未注册平台返回 false，但返回了 true")
	}
}

// TestResolveOfficialAnthropicNotRegistered 验证官方 anthropic 未进入 compat registry。
func TestResolveOfficialAnthropicNotRegistered(t *testing.T) {
	_, ok := anthropiccompat.Resolve("anthropic")
	if ok {
		t.Fatal("官方 anthropic 不应注册到 compat registry，但 Resolve 返回了 true")
	}
}

// TestListPlatformsContainsAllBuiltins 验证 ListPlatforms 包含所有内置 provider。
func TestListPlatformsContainsAllBuiltins(t *testing.T) {
	platforms := anthropiccompat.ListPlatforms()
	platformSet := make(map[string]bool, len(platforms))
	for _, p := range platforms {
		platformSet[p] = true
	}

	required := []string{
		"anthropic-compatible",
		"anthropic-zhipu",
		"anthropic-kimi",
		"anthropic-minimax",
		"anthropic-qwen",
		"anthropic-mimo",
	}
	for _, p := range required {
		if !platformSet[p] {
			t.Errorf("ListPlatforms 缺少平台 %q", p)
		}
	}
}

// TestDefaultModelsForPlatform 验证 DefaultModelsForPlatform 返回非空副本。
func TestDefaultModelsForPlatform(t *testing.T) {
	platforms := []string{
		"anthropic-compatible",
		"anthropic-zhipu",
		"anthropic-kimi",
		"anthropic-minimax",
		"anthropic-qwen",
		"anthropic-mimo",
	}
	for _, p := range platforms {
		t.Run(p, func(t *testing.T) {
			models := anthropiccompat.DefaultModelsForPlatform(p)
			if p != "anthropic-compatible" && len(models) == 0 {
				t.Errorf("平台 %q 的默认模型列表为空", p)
			}
			// 验证返回副本（修改副本不影响 registry）
			original := anthropiccompat.DefaultModelsForPlatform(p)
			if len(models) > 0 && len(original) > 0 {
				savedFirst := original[0]
				models[0] = "modified"
				fresh := anthropiccompat.DefaultModelsForPlatform(p)
				if len(fresh) > 0 && fresh[0] != savedFirst {
					t.Errorf("DefaultModelsForPlatform 未返回副本，外部修改影响了 registry 内数据")
				}
			}
		})
	}
}

func TestGenericProviderRequiresExplicitBaseURL(t *testing.T) {
	spec, ok := anthropiccompat.Resolve("anthropic-compatible")
	if !ok || spec == nil {
		t.Fatal("anthropic-compatible 未注册")
	}
	if spec.DefaultBaseURL != "" {
		t.Fatalf("DefaultBaseURL = %q，期望空字符串", spec.DefaultBaseURL)
	}
	if len(spec.DefaultModels) != 0 {
		t.Fatalf("DefaultModels = %v，期望空切片", spec.DefaultModels)
	}
}

// TestDefaultModelsForUnregisteredPlatform 验证未注册平台返回 nil。
func TestDefaultModelsForUnregisteredPlatform(t *testing.T) {
	models := anthropiccompat.DefaultModelsForPlatform("anthropic-nonexistent-xyz")
	if models != nil {
		t.Fatalf("未注册平台应返回 nil，但返回了 %v", models)
	}
}

func TestZhipuProviderDefaults(t *testing.T) {
	spec, ok := anthropiccompat.Resolve("anthropic-zhipu")
	if !ok || spec == nil {
		t.Fatal("anthropic-zhipu 未注册")
	}

	if spec.DefaultBaseURL != "https://open.bigmodel.cn/api/anthropic" {
		t.Fatalf("DefaultBaseURL = %q，期望 %q", spec.DefaultBaseURL, "https://open.bigmodel.cn/api/anthropic")
	}
	if got := spec.MessagesEndpointPath(); got != "/v1/messages" {
		t.Fatalf("MessagesEndpointPath() = %q，期望 %q", got, "/v1/messages")
	}
	name, value := spec.ResolveAuthHeader("test-key")
	if name != "x-api-key" {
		t.Fatalf("auth header name = %q，期望 %q", name, "x-api-key")
	}
	if value != "test-key" {
		t.Fatalf("auth header value = %q，期望 %q", value, "test-key")
	}
}
