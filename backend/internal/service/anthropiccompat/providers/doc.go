// Package providers 包含所有内置的 Anthropic-compatible 渠道 Provider 定义。
//
// 每个 Provider 文件在 init() 中自动向 anthropiccompat.Registry 注册 ProviderSpec。
// 在 gateway_forward_anthropic_compat.go 中通过空白导入触发所有 init()：
//
//	import _ "github.com/Wei-Shaw/sub2api/internal/service/anthropiccompat/providers"
//
// 新增渠道时，只需在此目录下新增一个 .go 文件，在 init() 中调用 anthropiccompat.Register()。
package providers
