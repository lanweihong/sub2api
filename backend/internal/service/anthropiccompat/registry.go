package anthropiccompat

import (
	"fmt"
	"sync"
)

var (
	// mu 保护 providers 映射，注册阶段（进程启动 init）写入，运行阶段只读。
	mu sync.RWMutex
	// providers 存储所有已注册渠道的 ProviderSpec，以 Platform 为键。
	providers = make(map[string]*ProviderSpec)
)

// Register 注册一个 ProviderSpec。
// 同一 Platform 重复注册时 panic，属于编程错误，应在开发阶段发现。
// 所有注册操作应在包 init 函数中完成，保证进程启动时已初始化完毕。
func Register(spec *ProviderSpec) {
	if spec == nil || spec.Platform == "" {
		panic("anthropiccompat: ProviderSpec 的 Platform 字段不能为空")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[spec.Platform]; exists {
		panic(fmt.Sprintf("anthropiccompat: 平台 %q 已注册，不允许重复注册", spec.Platform))
	}
	providers[spec.Platform] = spec
}

// Resolve 根据 platform 查找对应的 ProviderSpec。
// 未注册时返回 (nil, false)，转发层据此拒绝路由，防止误走本包。
func Resolve(platform string) (*ProviderSpec, bool) {
	mu.RLock()
	defer mu.RUnlock()
	spec, ok := providers[platform]
	return spec, ok
}

// ListPlatforms 返回所有已注册的 platform 名称列表，供管理后台展示可选渠道。
func ListPlatforms() []string {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]string, 0, len(providers))
	for p := range providers {
		result = append(result, p)
	}
	return result
}

// DefaultModelsForPlatform 返回指定平台的默认模型列表副本。
// 供管理后台 /accounts/:id/models 接口在未配置 model_mapping 时使用。
func DefaultModelsForPlatform(platform string) []string {
	mu.RLock()
	defer mu.RUnlock()
	spec, ok := providers[platform]
	if !ok {
		return nil
	}
	// 返回副本，防止外部代码意外修改 registry 内的切片。
	out := make([]string, len(spec.DefaultModels))
	copy(out, spec.DefaultModels)
	return out
}
