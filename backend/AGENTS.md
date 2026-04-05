# Repository Guidelines

## 后端整体结构
`backend/` 是一个典型的 Gin + Wire + Ent 分层服务。启动入口在 `cmd/server`：`main.go` 负责 setup/server 两种模式，`wire.go`/`wire_gen.go` 负责依赖注入和后台任务清理。`internal/config` 管配置加载与校验，`internal/setup` 管首次安装和自动初始化，`internal/server` 负责 HTTP server、路由注册和服务器级中间件。

业务代码主要在 `internal/handler`、`internal/service`、`internal/repository`。其中 `internal/handler/admin` 是后台管理接口，`internal/handler/dto` 负责把 service 模型转换成对外响应。注意：本项目的大部分业务实体如 `User`、`Account`、`Group`、`APIKey` 实际定义在 `internal/service`，`internal/domain` 主要放常量。`internal/repository` 实现 Ent/Redis/外部 HTTP 访问，`internal/pkg` 与 `internal/util` 放协议适配、日志、错误、响应、脱敏等横切能力。数据库结构以 `migrations/*.sql` 为准，`ent/schema` 和生成代码 `ent/` 需要同步维护。

## 分层与依赖约束
保持调用链为 `routes -> handler -> service -> repository`。`service` 和 `handler` 不应直接依赖 `internal/repository`、`gorm`、`redis`；`.golangci.yml` 已用 `depguard` 强约束。Repository 接口定义在 `internal/service`，实现放在 `internal/repository`。新增后台 worker、定时任务或缓存刷新器时，除了加 Wire provider，还要把启动/停止接入 `internal/service/wire.go` 和 `cmd/server/wire.go` 的 cleanup。

## 代码风格与命名
统一使用 `gofmt`，并按现有 lint 约束写代码：优先用 `any`，保留大写缩写风格如 `APIKey`、`UserID`、`TTL`、`HTTP`。构造函数通常为 `NewXxx`，Wire provider 通常为 `ProvideXxx`。Handler 层请求结构体通常就地定义，复杂输出放到 `internal/handler/dto`。更新类输入大量使用指针字段区分“未提供”和“显式置零/置空”，新增更新接口时延续这个模式。注释以解释“为什么”优先，项目允许中英混合注释，但应保持直接、技术化。

## HTTP、响应与异常处理
普通管理/用户接口统一走 `internal/pkg/response`：成功包裹为 `{code:0,message:"success",data:...}`，错误优先使用 `response.ErrorFrom(c, err)`，由 `internal/pkg/errors` 的 `ApplicationError` 映射到 HTTP 状态、`reason` 和 `metadata`。新增业务错误时，优先在 service 层定义包级 `ErrXxx = infraerrors.BadRequest/Forbidden/...`；需要保留底层原因时用 `fmt.Errorf("op: %w", err)` 或 `WithCause(err)`。

Repository 层不要把 Ent、`pq`、`sql` 细节直接泄漏到上层；优先复用 `translatePersistenceError` 做 not found / unique conflict 翻译。`Recovery()` 只返回通用 500，并对 broken pipe / connection reset 不再写响应。`response.ErrorFrom` 会对 5xx 做脱敏日志记录。

注意例外：认证中间件、限流中间件和网关协议端点并不总是使用统一响应包。`internal/server/middleware` 中的鉴权错误、`internal/middleware/rate_limiter.go` 的限流错误，以及 `/v1/messages`、`/v1beta/models`、`/responses` 等协议兼容接口，往往必须返回 Anthropic/OpenAI/Google 兼容格式。修改这些路径时，先确认当前端点应该遵循“项目统一 envelope”还是“上游协议原生格式”，不要混用。

## 事务、缓存与后台任务
多表写操作通常显式使用 Ent 事务，典型场景有用户创建、使用记录扣费、批量配置更新。涉及余额、并发、分组、API Key、调度可用性等状态变更时，要同步评估认证缓存、计费缓存、调度快照和队列状态是否需要失效。项目对部分旁路失败采取“记录日志但不中断主流程”的策略，例如优惠码附加动作、邀请码标记、Ops 异步入库、部分隐私设置补偿；新增逻辑时要区分“核心不变量失败应返回错误”和“最佳努力失败只记录日志”。

## 测试、生成代码与提交流程
常用命令：`make test`、`make test-unit`、`make test-integration`、`make test-e2e-local`、`golangci-lint run ./...`。测试大量使用 build tags：`unit`、`integration`、`e2e`，共享 stub/fixture 在 `internal/testutil`。修改 handler/service/repository 时，优先在对应层旁边补测试；涉及跨层契约时再加 integration。修改 `ent/schema` 后必须运行 `make -C backend generate` 并提交生成后的 `ent/`。数据库变更优先新增 SQL migration，不依赖 Ent 自动迁移。

## 新增功能时的落点建议
新增普通 API 时，先在 `internal/server/routes` 注册路由，再写薄 handler，业务判断放 service，存储/外部调用放 repository。新增设置项时，先加 `internal/config` 或 `SettingService`，只有确实需要前端感知时再透出到 public settings / DTO。新增日志时优先使用 request-scoped logger，并避免打印 token、cookie、完整凭证或未经脱敏的上游响应体。
