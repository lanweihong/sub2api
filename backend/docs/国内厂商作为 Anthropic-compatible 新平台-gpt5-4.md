# 低侵入 Anthropic-compatible 渠道底座改造方案

## Summary

目标调整为：

- 后续新增渠道大多仍是 Anthropic-compatible。
- 这次改造优先做“底座式封装”，尽量不动现有主逻辑代码。
- 现有官方 Anthropic、OpenAI、Gemini、Antigravity、Sora 行为要尽可能保持原样。
- 新渠道通过独立封装接入，默认不复用官方 Anthropic 的 OAuth、window cost、Claude 特化逻辑。
- 新增渠道时，尽量通过“新增一个 provider 定义 + 可选小型 hook”完成，而不是修改现有 handler/service 主流程。

最佳落点不是重写 `GatewayService`，而是在其外侧增加一个新的 `AnthropicCompatibleGatewayFacade` 和 `ProviderRegistry`。现有官方 Anthropic 流量继续走老逻辑；只有明确标记为“自定义 Anthropic-compatible 渠道”的分组/账号，才进入新封装层。这样可以把风险集中在新增代码，不把现有稳定链路拉进重构。

## Key Changes

### 1. 保留现有主链路，新增“旁路封装层”

现有逻辑保持如下原则：

- 现有 `PlatformAnthropic` 官方账号继续走原 `GatewayHandler.Messages -> GatewayService` 链路。
- 不对现有 `GatewayService` 做大规模职责拆分。
- 不把现在依赖 `PlatformAnthropic` 的 Claude 官方特化逻辑整体重构。

新增一层旁路：

- `AnthropicCompatibleGatewayFacade`
- `AnthropicCompatibleProviderRegistry`
- `AnthropicCompatibleForwarder`

职责分工：

- `GatewayHandler.Messages` 只增加一个很薄的入口判断：
  - 如果当前分组/账号属于“自定义 Anthropic-compatible 渠道”，转给 facade
  - 否则继续原样调用现有 `GatewayService`
- facade 负责：
  - 解析 provider spec
  - 构造上游请求
  - 处理流式/非流式响应
  - 统一错误包装
  - 记录必要的 usage / failover / ops 事件
- 现有 `GatewayService` 只保留给官方 Anthropic 及当前既有平台逻辑

这样改造的核心收益是：

- 原功能不被迫进入“抽象重构期”
- 新渠道的复杂性隔离在新增文件里
- 回滚也简单，只需关闭新入口分流

### 2. 新渠道不再塞进 `PlatformAnthropic`，但也不要求大改现有枚举语义

为兼容现有系统，建议继续复用现有 `platform` 字段，但引入明确区分：

- 官方平台：
  - `anthropic`
  - `openai`
  - `gemini`
  - `antigravity`
  - `sora`
- 新增自定义渠道平台：
  - `anthropic-zhipu`
  - `anthropic-kimi`
  - `anthropic-minimax`
  - 未来可扩展为 `anthropic-<provider_key>`

不建议这次就上数据库层面的 `protocol_family + provider_key` 双字段拆分，因为那会侵入太多现有逻辑。  
本次只需要形成约定：

- `strings.HasPrefix(platform, "anthropic-")` 代表自定义 Anthropic-compatible 渠道
- `platform == "anthropic"` 继续表示官方 Anthropic

这样能满足两个目标：

- 避免和官方 Anthropic 混淆
- 尽量少改现有筛选、分组、后台模型

### 3. 新增 Provider Registry，后续新增渠道优先只加配置定义

新增一个代码内 registry，建议形式：

- `RegisterAnthropicCompatibleProvider(spec ProviderSpec)`
- `ResolveAnthropicCompatibleProvider(platform string) (ProviderSpec, bool)`

`ProviderSpec` 只覆盖新渠道所需最小能力：

- `Platform`
- `DisplayName`
- `DefaultBaseURL`
- `MessagesPath`
- `CountTokensPath`
- `AppendBetaQuery`
- `AuthMode`
- `AuthHeaderName`
- `DefaultHeaders`
- `SupportsCountTokens`
- `SupportsUsageFromBody`
- `SupportsStreaming`
- `SupportsTools`
- `SupportsThinking`
- `DefaultModels`
- `NormalizeModelID`
- `RequestMutator`
- `ErrorParser`
- `UsageParser`

默认策略：

- 首期 registry 代码内置
- 后续新增 Anthropic-compatible 渠道时，优先只新增一个 `ProviderSpec`
- 若 spec 不够表达差异，再加一个小 hook，但不修改 facade 主流程

这符合你“以后尽量不要再改逻辑代码”的目标。

### 4. 新增 Facade 的实现边界，避免重写现有 GatewayService

`AnthropicCompatibleGatewayFacade` 只做新渠道，不取代旧服务。

建议内部能力：

- `ForwardMessages`
- `ForwardCountTokens`
- `ListModels`
- `ParseUpstreamError`
- `ParseUsage`
- `HandleStreamingResponse`
- `HandleBufferedResponse`

数据来源：

- 请求解析继续复用现有 `ParseGatewayRequest`
- `model_mapping` 继续复用现有 `Account.GetMappedModel` / `ResolveMappedModel`
- 调度仍复用现有账号/分组/缓存/并发控制体系
- 只有“真正发上游请求以及解析返回”由新 facade 独立实现

也就是说：

- 入口鉴权、并发控制、账单资格检查、粘性会话、账号选择等仍尽量复用现有 handler/service 上层逻辑
- 上游请求构造与协议处理单独封装，不去改造老 `GatewayService.buildUpstreamRequest`

### 5. 现有 `GatewayHandler.Messages` 只做一个最小分流

本次对现有主逻辑代码的侵入应控制在“单点薄改动”。

建议只加：

- 一个 helper：`isCustomAnthropicCompatiblePlatform(platform string) bool`
- 一个 facade 注入
- 一个早期分流判断

伪行为：

1. 读取 group/platform
2. 如果是 `anthropic-*` 自定义渠道
   - 走新 facade
3. 否则
   - 保持现有 `GatewayService` 原路径

不要在现有主流程里散落 provider 判断。  
所有新渠道特化都下沉到 facade 和 registry。

### 6. `count_tokens`、`models`、`usage` 同样采用旁路处理

为了保证用户体验一致，这几个端点也应该纳入新底座，但仍按低侵入方式接入。

#### `/v1/messages/count_tokens`

- 对自定义 Anthropic-compatible 渠道：
  - 若 provider spec 支持，调用 facade 转发
  - 若不支持，直接返回 Anthropic 风格“不支持”
- 对官方 Anthropic：
  - 保持原逻辑

#### `/v1/models`

- 对自定义 Anthropic-compatible 渠道：
  - 若分组下账号配置了 `model_mapping`，返回聚合白名单
  - 否则返回 provider spec 的 `DefaultModels`
- 对官方 Anthropic：
  - 保持原逻辑

#### `/v1/usage`

- 对自定义 Anthropic-compatible 渠道：
  - 首期建议只返回本地系统可表达的最小兼容结果
  - 不去接远端余额/套餐接口
- 对官方 Anthropic：
  - 保持原逻辑

### 7. 后台与账号管理也采用“新增分支，不重构旧逻辑”

后台侧不做大重构，只增加对 `anthropic-*` 平台的支持。

创建/更新账号：

- 新增平台值允许 `anthropic-zhipu`、`anthropic-kimi`、`anthropic-minimax`
- 这些平台仅允许 `type=apikey`
- `base_url` 必填
- `api_key` 必填
- 允许 `model_mapping`
- 可选 `custom_headers`
- 仍使用现有 URL allowlist

账号测试：

- 新增一个统一的 `testCustomAnthropicCompatibleAccountConnection`
- 官方 Anthropic 测试逻辑不动
- 自定义渠道统一走新 provider spec 测试流

模型列表：

- 新增 provider 默认模型目录，例如：
  - `internal/pkg/anthropiccompat/providers/zhipu_models.go`
  - `internal/pkg/anthropiccompat/providers/kimi_models.go`
  - `internal/pkg/anthropiccompat/providers/minimax_models.go`
- 后台 `/accounts/:id/models` 遇到 `anthropic-*` 平台，优先读取 provider 默认模型集

### 8. 封装目录建议独立，体现“底下加一层”的设计

为了降低认知干扰，新增代码建议集中到独立目录，而不是散插到现有文件各处。

建议新增目录：

- `internal/service/anthropiccompat/`

内部可包含：

- `registry.go`
- `types.go`
- `facade.go`
- `forward_messages.go`
- `forward_count_tokens.go`
- `stream.go`
- `errors.go`
- `usage.go`
- `providers/`
  - `zhipu.go`
  - `kimi.go`
  - `minimax.go`

这样实现上有几个好处：

- 与现有 `GatewayService` 明确隔离
- 后续扩渠道时变更集中
- 回归分析容易
- 便于未来抽象出更多协议族时复制模式

### 9. 风险控制与上线策略

因为你的核心诉求是“防止影响原来功能运行”，所以必须加明确的保护策略：

- 新 facade 默认只服务 `anthropic-*` 平台
- 官方 `anthropic` 平台绝不自动切到新 facade
- 如有需要，增加一个配置开关：
  - `gateway.anthropic_compatible_facade_enabled`
- 分阶段上线：
  1. 先接入一个新渠道验证
  2. 观察 ops / error / billing / usage
  3. 再逐步加其他渠道

同时要求：

- 新 facade 的日志命名空间独立，例如 `service.anthropic_compat`
- ops 中真实记录 platform，便于隔离问题
- 出现异常时可以只停用某个 `anthropic-*` 平台，不影响官方 Anthropic

### 10. 对未来“尽量不再改代码”的真实承诺方式

在这套方案下，可以合理承诺：

- 未来大多数 Anthropic-compatible 渠道新增时，不改现有主逻辑
- 未来通常只需要：
  - 注册一个新 provider spec
  - 增加默认模型定义
  - 配好 base URL / headers / feature flags
  - 必要时补一个小型 parser / mutator

但不能承诺：

- 任何渠道都完全零代码
- 协议差异特别大的厂商也无需写新 hook

所以本次改造的目标应写成：

- “让新增 Anthropic-compatible 渠道不再触碰现有稳定主链路”
- 而不是“未来任何渠道都不需要写代码”

## Public API / Interface Changes

- 外部 API 路径保持不变：
  - `/v1/messages`
  - `/v1/messages/count_tokens`
  - `/v1/models`
  - `/v1/usage`
- 管理后台新增可选平台值：
  - `anthropic-zhipu`
  - `anthropic-kimi`
  - `anthropic-minimax`
  - 未来 `anthropic-<provider>`
- 内部新增 facade/registry 接口，例如：
  - `ResolveAnthropicCompatibleProvider(platform string)`
  - `AnthropicCompatibleGatewayFacade.ForwardMessages(...)`
- 现有 `GatewayService` 对官方 Anthropic 的 public behavior 不变

## Test Plan

- 路由分流：
  - `platform=anthropic-*` 会进入新 facade
  - `platform=anthropic` 继续走老链路
- 回归保护：
  - 官方 Anthropic `/v1/messages` 行为不变
  - OpenAI/Gemini/Antigravity/Sora 行为不变
- Provider registry：
  - 新平台能正确解析到 spec
  - 未注册平台不会误走 facade
- 请求构造：
  - 自定义渠道能正确拼接 `base_url + messages_path`
  - auth header / default headers 生效
- 流式与错误：
  - Anthropic-compatible SSE 能被正确透传或缓冲
  - `401/429/5xx` 能被正确包装和记录
- 后台：
  - 新平台账号创建校验生效
  - 账号测试可用
  - `/accounts/:id/models` 返回 provider 默认模型或 mapping 白名单
- 低侵入验证：
  - 针对现有官方 Anthropic 流量的测试基线无回归
  - 现有 `GatewayService` 相关测试只需最小修改或不修改

## Assumptions

- 后续新增渠道大多仍是 Anthropic-compatible。
- 你接受新渠道平台命名采用 `anthropic-*` 风格，以换取最小侵入接入。
- 首期只支持 API Key。
- 本次不做数据库结构大调整，不引入完整动态插件系统。
- 新增底座目录和 facade 属于“新增能力”，不是“重构现有核心链路”。
- 若未来渠道协议明显偏离 Anthropic-compatible，再在这个模式旁边新增第二个协议底座，而不是继续污染当前 facade。

---

## 补充说明：基于代码审查的可行性分析与修正建议

> 以下内容是在原方案基础上，结合实际代码结构进行深入审查后的补充。目的是修正原方案中与现有架构不匹配的部分，补全遗漏点，并给出更精确的实施路径。

### 整体可行性评估

**评分：7.5/10 — 方向正确，但部分设计需要修正**

原方案的核心思路（侧通道隔离、`anthropic-*` 命名、Provider Registry 模式）是正确的。以下针对需要修正和补充的部分逐项说明。

### 修正 1：不建议独立 Facade struct，改为 GatewayService 方法 + 独立文件

**原方案**：新增 `AnthropicCompatibleGatewayFacade` 作为独立 struct，注入到 `GatewayHandler`。

**问题**：
- 独立 facade struct 需要大量 Wire DI 变更（新增 ProviderSet、修改 `NewGatewayHandler` 参数、重新生成 `wire_gen.go`）
- facade 需要依赖 `HTTPUpstream`、`BillingCacheService`、`UsageRecordWorkerPool` 等多个服务，依赖图复杂
- 与现有代码库的平台扩展模式不一致

**现有先例分析**：

| 平台 | Handler | Service | 路由 | 分流方式 |
|------|---------|---------|------|----------|
| OpenAI | 独立 Handler | 独立 Service | 路由级分流 | `getGroupPlatform` |
| **Antigravity** | **复用 GatewayHandler** | **独立 Service** | **handler 内分流 L671** | **account.Platform 判断** |
| Gemini | 复用 GatewayHandler | 独立 Service | handler 内分流 L284 | platform 判断 |
| Sora | 独立 Handler | 独立 Service | 独立路由组 | 独立路由 |

**最接近的先例是 Antigravity 模式**：在 `GatewayHandler.Messages` 的 forward 分支中，根据 `account.Platform` 分流到不同的 service 方法。

**修正方案**：
- 不创建独立 facade struct
- 在 `GatewayService` 上新增 `ForwardAnthropicCompat` 方法，放在独立文件 `gateway_forward_anthropic_compat.go` 中
- Provider Registry 作为包级变量放在 `internal/service/anthropiccompat/` 下
- Wire DI 几乎不需要变更

### 修正 2：分流点需更精确定位

**原方案**：在 `GatewayHandler.Messages` 中"增加一个很薄的入口判断"，在账号选择之前就分流到 facade。

**问题**：实际代码中，`GatewayHandler.Messages` 有 800+ 行，大量前置逻辑（body 读取、billing 检查、并发控制、粘性会话）是可复用的。如果在入口处就分流，新 facade 要重新实现这些逻辑。

**代码路径分析**（`internal/handler/gateway_handler.go`）：

```
L112: 获取 API Key
L134: 读取请求体
L152: ParseGatewayRequest(body, domain.PlatformAnthropic)   ← 可复用，协议相同
L163: isMaxTokensOneHaikuRequest                             ← Claude 特有，需跳过
L169: SetClaudeCodeClientContext                             ← Claude 特有，需跳过
L173: checkClaudeCodeVersion                                 ← Claude 特有，需跳过
L200: 并发控制（用户槽位）                                      ← 可复用
L241: 计费资格检查                                             ← 可复用
L248: 粘性会话 hash                                           ← 可复用
L258: platform 确定（从 forcePlatform 或 group.Platform）
L284: if platform == PlatformGemini → Gemini 分支
L511: for 循环 → Anthropic/Antigravity 分支
L517: SelectAccountWithLoadAwareness                          ← 可复用
L671: if account.Platform == PlatformAntigravity → forward 分流   ← 分流点
L674: else → gatewayService.Forward（标准 Anthropic）
```

**修正方案**：

分流不在入口，而在 **L671 的 forward 分支处**新增：

```go
// gateway_handler.go L671 附近
if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
    result, err = h.antigravityGatewayService.Forward(...)
} else if domain.IsAnthropicCompatPlatform(account.Platform) {
    result, err = h.gatewayService.ForwardAnthropicCompat(requestCtx, c, account, parsedReq)
} else {
    result, err = h.gatewayService.Forward(requestCtx, c, account, parsedReq)
}
```

这样所有前置逻辑（billing、并发、粘性会话、账号选择、failover 循环）全部复用，只有"真正发上游请求和解析返回"由新方法处理。

### 修正 3：Claude Code 检查存在隐患，需加 platform guard

**问题**：`gateway_handler.go` 中 L163-L178 的 Claude Code 相关逻辑（`isMaxTokensOneHaikuRequest`、`SetClaudeCodeClientContext`、`checkClaudeCodeVersion`）在 **platform 确定之前**执行（platform 在 L258 才确定）。

- `SetClaudeCodeClientContext`（L169）：检查 user-agent 中是否包含 "claude-code"。对非 Claude Code 客户端无副作用。
- `checkClaudeCodeVersion`（L173）：**如果管理员设了最低 Claude Code 版本要求，非 Claude Code 客户端会被误拦截**。

**修正方案**：

将 platform 确定逻辑提前到 L163 之前，然后用 guard 包裹 Claude Code 逻辑：

```go
// 提前确定 platform
platform := ""
if forcePlatform, ok := middleware2.GetForcePlatformFromContext(c); ok {
    platform = forcePlatform
} else if apiKey.Group != nil {
    platform = apiKey.Group.Platform
}

// Claude Code 逻辑仅对官方 Anthropic 平台生效
if !domain.IsAnthropicCompatPlatform(platform) {
    if isMaxTokensOneHaikuRequest(reqModel, parsedReq.MaxTokens, reqStream) {
        // ...
    }
    SetClaudeCodeClientContext(c, body, parsedReq)
    if !h.checkClaudeCodeVersion(c) {
        return
    }
}
```

### 补充 1：`anthropic-*` 命名安全性确认

经代码审查确认，`strings.HasPrefix(platform, "anthropic-")` 是安全的：

- `"anthropic"` 不含 `-` 后缀，不会触发前缀匹配
- 现有所有平台（`openai`、`gemini`、`antigravity`、`sora`）都不以 `anthropic-` 开头
- 数据库 `platform` 字段是 `varchar(50)`，无枚举约束，**无需 migration**
- `SchedulerSnapshotService.resolveMode()` 中的 `platform == PlatformAnthropic` 不会对新平台触发混合调度（预期行为）
- `repository.ListSchedulableByPlatform()` 使用 `PlatformEQ(platform)` 精确匹配，传入 `"anthropic-zhipu"` 会正确匹配

建议在 `internal/domain/constants.go` 新增统一辅助函数，避免散布 `strings.HasPrefix`：

```go
// IsAnthropicCompatPlatform 判断是否为自定义 Anthropic-compatible 渠道
func IsAnthropicCompatPlatform(platform string) bool {
    return strings.HasPrefix(platform, "anthropic-")
}
```

### 补充 2：组件复用详细分析

| 组件 | 可复用 | 说明 |
|------|--------|------|
| `ParseGatewayRequest` | ✅ | 协议相同，传 `PlatformAnthropic` 即可 |
| `SelectAccountWithLoadAwareness` | ✅ | 按 `group.Platform` 精确匹配账号，自动工作 |
| `BillingCacheService` | ✅ | 与平台无关 |
| `ConcurrencyHelper` | ✅ | 与平台无关 |
| `GenerateSessionHash` | ✅ | 基于请求内容 hash，与平台无关 |
| `RecordUsage` / `UsageRecordWorkerPool` | ✅ | 与平台无关 |
| `model_mapping` (`GetMappedModel`) | ✅ | `AccountTypeAPIKey` 已支持 |
| `GetAccessToken` | ⚠️ | 对 APIKey 类型直接返回 key，可用；OAuth 不可用（专为 Anthropic 设计） |
| `buildUpstreamRequest` | ❌ | 硬编码了 OAuth/TLS/ClaudeCode 逻辑，需新建简化版 |
| `handleStreamingResponse` | ❌ | 含大量 Claude 特有逻辑（model replacement、thinking block 签名验证），需新建简化版 |
| `handleErrorResponse` | ❌ | 含 Anthropic 特有错误码解析，需参考新建 |

### 补充 3：方案遗漏的散点改动

以下是原方案未提及但实际需要改动的位置：

1. **`account_service.go` L413 — `TestCredentials`**：使用 `switch account.Platform`，default 分支返回 `"unsupported platform"` 错误。**必须新增 case 分支**，否则新平台账号无法测试连接。

2. **`IsSingleAntigravityAccountGroup`（handler L506）**：在 Anthropic 通用分支中也会被调用。虽然对非 Antigravity 分组返回 false 不会有害，但逻辑上不正确，建议后续加 platform guard。

3. **定价配置**：原方案未提及新平台模型的 token 定价如何配置。现有 `PricingService` 需要为新模型配置价格，否则计费会使用 fallback 价格（可能不准确）。建议在 `resources/` 下添加定价表。

4. **前端变更**：
   - 管理端账号创建界面：新增平台选项
   - 分组创建界面：新增平台选项
   - `useModelWhitelist.ts`：模型白名单可能需要扩展
   - `src/i18n/`：国际化文件新增翻译

5. **`admin_service.go` 中 CreateAccount 后的隐私设置**：`switch account.Platform` 仅处理 OpenAI 和 Antigravity。新平台不需要，但 default 分支跳过不会报错，无需改动。

### 补充 4：SSE 流式处理工作量

原方案提到"response parsing are new"，但实际工作量需要明确。现有 `handleStreamingResponse` 包含：

- Anthropic SSE 事件类型处理（`message_start`, `content_block_delta`, `message_delta`, `message_stop`）
- Model name replacement in events
- Thinking block 签名验证
- Stream timeout 检测
- 增量用量从 `message_start`（input_tokens）和 `message_delta`（output_tokens）事件中提取

对于自定义 Anthropic-compatible 平台，可以**大幅简化**：
- 直接透传 SSE 事件（不做 model name 替换和 thinking block 验证）
- 只在 `message_delta`/`message_stop` 事件中提取用量
- 估计工作量约 1.5 天

### 补充 5：修正后的代码组织建议

```
backend/
├── internal/
│   ├── domain/
│   │   └── constants.go                          # + IsAnthropicCompatPlatform()
│   ├── service/
│   │   ├── anthropiccompat/                      # 新增目录
│   │   │   ├── registry.go                       # Provider Registry（包级变量）
│   │   │   ├── types.go                          # ProviderSpec 定义
│   │   │   └── providers/                        # 各厂商 spec
│   │   │       ├── zhipu.go
│   │   │       ├── kimi.go
│   │   │       └── minimax.go
│   │   ├── gateway_forward_anthropic_compat.go   # 新增：ForwardAnthropicCompat 方法
│   │   └── domain_constants.go                   # + 转发新常量
│   └── handler/
│       └── gateway_handler.go                    # 修改：platform 提前 + 分支
```

与原方案的区别：
- `ForwardAnthropicCompat` 是 `GatewayService` 的方法，不是独立 facade struct
- 不需要 `facade.go`、`forward_messages.go` 等独立文件
- Wire DI 几乎不需要变更

### 补充 6：工作量估算

| 模块 | 复杂度 | 估计时间 |
|------|--------|---------|
| Domain 常量与辅助函数 | 低 | 0.5 天 |
| Provider Registry + ProviderSpec | 低 | 0.5 天 |
| ForwardAnthropicCompat + 请求构建 | 中 | 1 天 |
| SSE 流式响应处理（简化版） | 中-高 | 1.5 天 |
| Handler 分流改造 | 中 | 1 天 |
| 后台管理适配（账号创建/测试/模型） | 中 | 1 天 |
| 前端适配 | 中 | 1.5 天 |
| 测试（单元 + 集成 + 回归） | 中 | 1.5 天 |
| **总计** | | **~8.5 天** |
