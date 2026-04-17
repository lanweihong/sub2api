package domain

import "testing"

// TestIsAnthropicCompatPlatform 验证 IsAnthropicCompatPlatform 函数的判断逻辑。
// 确保官方 Anthropic 平台不被误判，自定义 anthropic-* 渠道能正确识别。
func TestIsAnthropicCompatPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		platform string
		want     bool
		desc     string
	}{
		// 自定义 Anthropic-compatible 渠道应返回 true
		{PlatformAnthropicCompatible, true, "通用兼容渠道"},
		{PlatformAnthropicZhipu, true, "智谱 GLM 渠道"},
		{PlatformAnthropicKimi, true, "Kimi/Moonshot 渠道"},
		{PlatformAnthropicMinimax, true, "MiniMax 渠道"},
		{PlatformAnthropicQwen, true, "通义千问渠道"},
		{PlatformAnthropicMimo, true, "小米 MiMo 渠道"},
		{"anthropic-custom-vendor", true, "任意 anthropic-* 前缀渠道"},

		// 官方平台应返回 false
		{PlatformAnthropic, false, "官方 Anthropic 不应被识别为 compat"},
		{PlatformOpenAI, false, "OpenAI 平台"},
		{PlatformGemini, false, "Gemini 平台"},
		{PlatformAntigravity, false, "Antigravity 平台"},

		// 边缘情况
		{"", false, "空字符串"},
		{"anthropic", false, "精确等于 anthropic（不含 -）"},
		{"anthropic2", false, "anthropic2 不含 - 分隔符"},
	}

	for _, tc := range tests {
		tc := tc // 循环变量捕获
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := IsAnthropicCompatPlatform(tc.platform)
			if got != tc.want {
				t.Errorf("IsAnthropicCompatPlatform(%q) = %v，期望 %v（%s）",
					tc.platform, got, tc.want, tc.desc)
			}
		})
	}
}

// TestAnthropicCompatPlatformConstants 验证所有内置常量值均以 "anthropic-" 为前缀。
func TestAnthropicCompatPlatformConstants(t *testing.T) {
	t.Parallel()

	compatPlatforms := []string{
		PlatformAnthropicCompatible,
		PlatformAnthropicZhipu,
		PlatformAnthropicKimi,
		PlatformAnthropicMinimax,
		PlatformAnthropicQwen,
		PlatformAnthropicMimo,
	}

	for _, p := range compatPlatforms {
		if !IsAnthropicCompatPlatform(p) {
			t.Errorf("平台常量 %q 应被 IsAnthropicCompatPlatform 识别为 compat 渠道", p)
		}
	}
}

// TestOfficialPlatformsNotCompatPlatforms 验证所有官方平台常量均不被误判为 compat 渠道。
func TestOfficialPlatformsNotCompatPlatforms(t *testing.T) {
	t.Parallel()

	officialPlatforms := []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
	}

	for _, p := range officialPlatforms {
		if IsAnthropicCompatPlatform(p) {
			t.Errorf("官方平台常量 %q 不应被 IsAnthropicCompatPlatform 识别为 compat 渠道", p)
		}
	}
}
