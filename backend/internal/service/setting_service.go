package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/singleflight"
)

var (
	ErrRegistrationDisabled   = infraerrors.Forbidden("REGISTRATION_DISABLED", "registration is currently disabled")
	ErrSettingNotFound        = infraerrors.NotFound("SETTING_NOT_FOUND", "setting not found")
	ErrDefaultSubGroupInvalid = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_INVALID",
		"default subscription group must exist and be subscription type",
	)
	ErrDefaultSubGroupDuplicate = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE",
		"default subscription group cannot be duplicated",
	)
)

type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
	GetAll(ctx context.Context) (map[string]string, error)
	Delete(ctx context.Context, key string) error
}

// cachedVersionBounds 缓存 Claude Code 版本号上下限（进程内缓存，60s TTL）
type cachedVersionBounds struct {
	min       string // 空字符串 = 不检查
	max       string // 空字符串 = 不检查
	expiresAt int64  // unix nano
}

// versionBoundsCache 版本号上下限进程内缓存
var versionBoundsCache atomic.Value // *cachedVersionBounds

// versionBoundsSF 防止缓存过期时 thundering herd
var versionBoundsSF singleflight.Group

// versionBoundsCacheTTL 缓存有效期
const versionBoundsCacheTTL = 60 * time.Second

// versionBoundsErrorTTL DB 错误时的短缓存，快速重试
const versionBoundsErrorTTL = 5 * time.Second

// versionBoundsDBTimeout singleflight 内 DB 查询超时，独立于请求 context
const versionBoundsDBTimeout = 5 * time.Second

// cachedBackendMode Backend Mode cache (in-process, 60s TTL)
type cachedBackendMode struct {
	value     bool
	expiresAt int64 // unix nano
}

var backendModeCache atomic.Value // *cachedBackendMode
var backendModeSF singleflight.Group

const backendModeCacheTTL = 60 * time.Second
const backendModeErrorTTL = 5 * time.Second
const backendModeDBTimeout = 5 * time.Second

// cachedGatewayForwardingSettings 缓存网关转发行为设置（进程内缓存，60s TTL）
type cachedGatewayForwardingSettings struct {
	fingerprintUnification           bool
	metadataPassthrough              bool
	cchSigning                       bool
	claudeOAuthSystemPromptInjection bool
	claudeOAuthSystemPrompt          string
	claudeOAuthSystemPromptBlocks    string
	anthropicCacheTTL1hInjection     bool
	rewriteMessageCacheControl       bool
	clientDatelineNormalization      bool
	expiresAt                        int64 // unix nano
}

var gatewayForwardingCache atomic.Value // *cachedGatewayForwardingSettings
var gatewayForwardingSF singleflight.Group

const gatewayForwardingCacheTTL = 60 * time.Second
const gatewayForwardingErrorTTL = 5 * time.Second
const gatewayForwardingDBTimeout = 5 * time.Second

// cachedPayloadLoggingSettings 缓存报文审计配置（进程内缓存，60s TTL）
type cachedPayloadLoggingSettings struct {
	settings  *PayloadLoggingSettings
	expiresAt int64 // unix nano
}

var payloadLoggingCache atomic.Value // *cachedPayloadLoggingSettings
var payloadLoggingSF singleflight.Group

const payloadLoggingCacheTTL = 60 * time.Second
const payloadLoggingErrorTTL = 5 * time.Second
const payloadLoggingDBTimeout = 5 * time.Second
// cachedAntigravityUserAgentVersion 缓存 Antigravity UA 版本号（进程内缓存，60s TTL）
type cachedAntigravityUserAgentVersion struct {
	version   string
	expiresAt int64 // unix nano
}

const antigravityUserAgentVersionCacheTTL = 60 * time.Second
const antigravityUserAgentVersionErrorTTL = 5 * time.Second
const antigravityUserAgentVersionDBTimeout = 5 * time.Second

// DefaultOpenAICodexUserAgent OpenAI Codex 默认 User-Agent（用于规避 Cloudflare 对浏览器 UA 的质询）
const DefaultOpenAICodexUserAgent = "codex-tui/0.125.0 (Ubuntu 22.4.0; x86_64) xterm-256color (codex-tui; 0.125.0)"

// cachedOpenAICodexUserAgent 缓存 OpenAI Codex UA（进程内缓存，60s TTL）
type cachedOpenAICodexUserAgent struct {
	value     string
	expiresAt int64 // unix nano
}

type cachedOpenAIQuotaAutoPauseSettings struct {
	settings  OpsOpenAIAccountQuotaAutoPauseSettings
	expiresAt int64
}

const openAICodexUserAgentCacheTTL = 60 * time.Second
const openAICodexUserAgentErrorTTL = 5 * time.Second
const openAICodexUserAgentDBTimeout = 5 * time.Second

const codexRestrictionPolicyCacheTTL = 60 * time.Second
const codexRestrictionPolicyDBTimeout = 5 * time.Second

// cachedCodexRestrictionPolicy codex_cli_only 全局加固策略缓存（进程内，60s TTL）。
// GetCodexRestrictionPolicy 在每个 codex_cli_only 账号的网关请求热路径上被调用，避免每次访问 DB。
type cachedCodexRestrictionPolicy struct {
	value     CodexRestrictionPolicy
	expiresAt int64 // unix nano
}

// cachedCyberSessionBlockRuntime cyber 会话屏蔽开关+TTL 进程内缓存（60s TTL）。
// GetCyberSessionBlockRuntime 在网关请求热路径上被调用，避免每次访问 DB。
type cachedCyberSessionBlockRuntime struct {
	enabled   bool
	ttl       time.Duration
	expiresAt int64 // unix nano
}

const cyberSessionBlockRuntimeCacheTTL = 60 * time.Second
const cyberSessionBlockRuntimeErrorTTL = 5 * time.Second
const cyberSessionBlockRuntimeDBTimeout = 5 * time.Second

const openAIQuotaAutoPauseSettingsCacheTTL = 60 * time.Second
const openAIQuotaAutoPauseSettingsErrorTTL = 5 * time.Second
const openAIQuotaAutoPauseSettingsDBTimeout = 5 * time.Second

const openAIQuotaAutoPauseSettingsRefreshKey = "openai_quota_auto_pause_settings"

// DefaultSubscriptionGroupReader validates group references used by default subscriptions.
type DefaultSubscriptionGroupReader interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
}

// WebSearchManagerBuilder creates a websearch.Manager from config (injected by infra layer).
// proxyURLs maps proxy ID to resolved URL for provider-level proxy support.
type WebSearchManagerBuilder func(cfg *WebSearchEmulationConfig, proxyURLs map[int64]string)

// SettingService 系统设置服务
type SettingService struct {
	settingRepo                 SettingRepository
	defaultSubGroupReader       DefaultSubscriptionGroupReader
	proxyRepo                   ProxyRepository // for resolving websearch provider proxy URLs
	cfg                         *config.Config
	onUpdate                    func() // Callback when settings are updated (for cache invalidation)
	version                     string // Application version
	webSearchManagerBuilder     WebSearchManagerBuilder
	antigravityUAVersionCache   atomic.Value // *cachedAntigravityUserAgentVersion
	antigravityUAVersionSF      singleflight.Group
	openAICodexUACache          atomic.Value // *cachedOpenAICodexUserAgent
	openAICodexUASF             singleflight.Group
	codexRestrictionPolicyCache atomic.Value // *cachedCodexRestrictionPolicy
	codexRestrictionPolicySF    singleflight.Group

	cyberSessionBlockRuntimeCache atomic.Value // *cachedCyberSessionBlockRuntime
	cyberSessionBlockRuntimeSF    singleflight.Group

	// openAIQuotaAutoPauseSettingsCache holds the most recently observed quota auto-pause
	// settings. GetOpenAIQuotaAutoPauseSettings reads this atomic.Value on the request hot
	// path without ever blocking on the DB; when the cached entry expires, a background
	// goroutine refreshes it via openAIQuotaAutoPauseSettingsSF (stale-while-revalidate).
	// This per-service field also gives tests natural isolation — each SettingService
	// instance owns its own cache, no shared package-level state.
	openAIQuotaAutoPauseSettingsCache atomic.Value // *cachedOpenAIQuotaAutoPauseSettings
	openAIQuotaAutoPauseSettingsSF    singleflight.Group
}

// DefaultPlatformQuotaSetting 单 platform 三档限额（nil = 沿用上层；0 = 显式禁用；>0 = 上限）
type DefaultPlatformQuotaSetting struct {
	DailyLimitUSD   *float64 `json:"daily"`
	WeeklyLimitUSD  *float64 `json:"weekly"`
	MonthlyLimitUSD *float64 `json:"monthly"`
}

type ProviderDefaultGrantSettings struct {
	Balance          float64
	Concurrency      int
	Subscriptions    []DefaultSubscriptionSetting
	GrantOnSignup    bool
	GrantOnFirstBind bool
	PlatformQuotas   map[string]*DefaultPlatformQuotaSetting // key = platform name
}

type AuthSourceDefaultSettings struct {
	Email                        ProviderDefaultGrantSettings
	LinuxDo                      ProviderDefaultGrantSettings
	OIDC                         ProviderDefaultGrantSettings
	WeChat                       ProviderDefaultGrantSettings
	GitHub                       ProviderDefaultGrantSettings
	Google                       ProviderDefaultGrantSettings
	DingTalk                     ProviderDefaultGrantSettings
	ForceEmailOnThirdPartySignup bool
}

type authSourceDefaultKeySet struct {
	// source 是 auth source 标识（如 "email"、"github"），仅用于 parse 时
	// slog.Warn 诊断输出，不再参与 key 拼接（platformQuotas 字段已存完整 key）。
	source           string
	balance          string
	concurrency      string
	subscriptions    string
	grantOnSignup    string
	grantOnFirstBind string
	platformQuotas   string // SettingKeyAuthSourcePlatformQuotas(source)
}

var (
	emailAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "email",
		balance:          SettingKeyAuthSourceDefaultEmailBalance,
		concurrency:      SettingKeyAuthSourceDefaultEmailConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultEmailSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultEmailGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultEmailGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("email"),
	}
	linuxDoAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "linuxdo",
		balance:          SettingKeyAuthSourceDefaultLinuxDoBalance,
		concurrency:      SettingKeyAuthSourceDefaultLinuxDoConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultLinuxDoSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultLinuxDoGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("linuxdo"),
	}
	oidcAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "oidc",
		balance:          SettingKeyAuthSourceDefaultOIDCBalance,
		concurrency:      SettingKeyAuthSourceDefaultOIDCConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultOIDCSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultOIDCGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultOIDCGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("oidc"),
	}
	weChatAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "wechat",
		balance:          SettingKeyAuthSourceDefaultWeChatBalance,
		concurrency:      SettingKeyAuthSourceDefaultWeChatConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultWeChatSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultWeChatGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultWeChatGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("wechat"),
	}
	gitHubAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "github",
		balance:          SettingKeyAuthSourceDefaultGitHubBalance,
		concurrency:      SettingKeyAuthSourceDefaultGitHubConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultGitHubSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultGitHubGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultGitHubGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("github"),
	}
	googleAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "google",
		balance:          SettingKeyAuthSourceDefaultGoogleBalance,
		concurrency:      SettingKeyAuthSourceDefaultGoogleConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultGoogleSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultGoogleGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultGoogleGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("google"),
	}
	dingTalkAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source:           "dingtalk",
		balance:          SettingKeyAuthSourceDefaultDingTalkBalance,
		concurrency:      SettingKeyAuthSourceDefaultDingTalkConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultDingTalkSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultDingTalkGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultDingTalkGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("dingtalk"),
	}
)

const (
	defaultAuthSourceBalance     = 0
	defaultAuthSourceConcurrency = 5
	defaultWeChatConnectMode     = "open"
	defaultWeChatConnectScopes   = "snsapi_login"
	defaultWeChatConnectFrontend = "/auth/wechat/callback"
	defaultGitHubOAuthAuthorize  = "https://github.com/login/oauth/authorize"
	defaultGitHubOAuthToken      = "https://github.com/login/oauth/access_token"
	defaultGitHubOAuthUserInfo   = "https://api.github.com/user"
	defaultGitHubOAuthEmails     = "https://api.github.com/user/emails"
	defaultGitHubOAuthScopes     = "read:user user:email"
	defaultGitHubOAuthFrontend   = "/auth/oauth/callback"
	defaultGoogleOAuthAuthorize  = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultGoogleOAuthToken      = "https://oauth2.googleapis.com/token"
	defaultGoogleOAuthUserInfo   = "https://openidconnect.googleapis.com/v1/userinfo"
	defaultGoogleOAuthScopes     = "openid email profile"
	defaultGoogleOAuthFrontend   = "/auth/oauth/callback"
	defaultLoginAgreementMode    = "modal"
	defaultLoginAgreementDate    = "2026-03-31"
)

// NewSettingService 创建系统设置服务实例
func NewSettingService(settingRepo SettingRepository, cfg *config.Config) *SettingService {
	return &SettingService{
		settingRepo: settingRepo,
		cfg:         cfg,
	}
}

// SetDefaultSubscriptionGroupReader injects an optional group reader for default subscription validation.
func (s *SettingService) SetDefaultSubscriptionGroupReader(reader DefaultSubscriptionGroupReader) {
	s.defaultSubGroupReader = reader
}

// SetProxyRepository injects a proxy repo for resolving websearch provider proxy URLs.
func (s *SettingService) SetProxyRepository(repo ProxyRepository) {
	s.proxyRepo = repo
}

func (s *SettingService) LoadAPIKeyACLTrustForwardedIPSetting(ctx context.Context) error {
	if s == nil || s.cfg == nil || s.settingRepo == nil {
		return nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAPIKeyACLTrustForwardedIP)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			s.cfg.SetTrustForwardedIPForAPIKeyACL(s.cfg.Security.TrustForwardedIPForAPIKeyACL)
			return nil
		}
		return fmt.Errorf("get api key acl forwarded ip setting: %w", err)
	}
	enabled := value == "true"
	s.cfg.SetTrustForwardedIPForAPIKeyACL(enabled)
	return nil
}

// GetAllSettings 获取所有系统设置
func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	return s.parseSettings(settings), nil
}

// SetOnUpdateCallback sets a callback function to be called when settings are updated
// This is used for cache invalidation (e.g., HTML cache in frontend server)
func (s *SettingService) SetOnUpdateCallback(callback func()) {
	s.onUpdate = callback
}

// SetVersion sets the application version for injection into public settings
func (s *SettingService) SetVersion(version string) {
	s.version = version
}

// getStringOrDefault 获取字符串值或默认值
func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
	if value, ok := settings[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

// IsTurnstileEnabled 检查是否启用 Turnstile 验证
func (s *SettingService) IsTurnstileEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

// GetTurnstileSecretKey 获取 Turnstile Secret Key
func (s *SettingService) GetTurnstileSecretKey(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileSecretKey)
	if err != nil {
		return ""
	}
	return value
}

// IsIdentityPatchEnabled 检查是否启用身份补丁（Claude -> Gemini systemInstruction 注入）
func (s *SettingService) IsIdentityPatchEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableIdentityPatch)
	if err != nil {
		// 默认开启，保持兼容
		return true
	}
	return value == "true"
}

// GetIdentityPatchPrompt 获取自定义身份补丁提示词（为空表示使用内置默认模板）
func (s *SettingService) GetIdentityPatchPrompt(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyIdentityPatchPrompt)
	if err != nil {
		return ""
	}
	return value
}

// GenerateAdminAPIKey 生成新的管理员 API Key
func (s *SettingService) GenerateAdminAPIKey(ctx context.Context) (string, error) {
	// 生成 32 字节随机数 = 64 位十六进制字符
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	key := AdminAPIKeyPrefix + hex.EncodeToString(bytes)

	// 存储到 settings 表
	if err := s.settingRepo.Set(ctx, SettingKeyAdminAPIKey, key); err != nil {
		return "", fmt.Errorf("save admin api key: %w", err)
	}

	return key, nil
}

// GetAdminAPIKeyStatus 获取管理员 API Key 状态
// 返回脱敏的 key、是否存在、错误
func (s *SettingService) GetAdminAPIKeyStatus(ctx context.Context) (maskedKey string, exists bool, err error) {
	key, err := s.settingRepo.GetValue(ctx, SettingKeyAdminAPIKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if key == "" {
		return "", false, nil
	}

	// 脱敏：显示前 10 位和后 4 位
	if len(key) > 14 {
		maskedKey = key[:10] + "..." + key[len(key)-4:]
	} else {
		maskedKey = key
	}

	return maskedKey, true, nil
}

// GetAdminAPIKey 获取完整的管理员 API Key（仅供内部验证使用）
// 如果未配置返回空字符串和 nil 错误，只有数据库错误时才返回 error
func (s *SettingService) GetAdminAPIKey(ctx context.Context) (string, error) {
	key, err := s.settingRepo.GetValue(ctx, SettingKeyAdminAPIKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", nil // 未配置，返回空字符串
		}
		return "", err // 数据库错误
	}
	return key, nil
}

// DeleteAdminAPIKey 删除管理员 API Key
func (s *SettingService) DeleteAdminAPIKey(ctx context.Context) error {
	return s.settingRepo.Delete(ctx, SettingKeyAdminAPIKey)
}

// IsModelFallbackEnabled 检查是否启用模型兜底机制
func (s *SettingService) IsModelFallbackEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableModelFallback)
	if err != nil {
		return false // Default: disabled
	}
	return value == "true"
}

// GetFallbackModel 获取指定平台的兜底模型
func (s *SettingService) GetFallbackModel(ctx context.Context, platform string) string {
	var key string
	var defaultModel string

	switch platform {
	case PlatformAnthropic:
		key = SettingKeyFallbackModelAnthropic
		defaultModel = "claude-3-5-sonnet-20241022"
	case PlatformOpenAI:
		key = SettingKeyFallbackModelOpenAI
		defaultModel = "gpt-4o"
	case PlatformGemini:
		key = SettingKeyFallbackModelGemini
		defaultModel = "gemini-2.5-pro"
	case PlatformAntigravity:
		key = SettingKeyFallbackModelAntigravity
		defaultModel = "gemini-2.5-pro"
	default:
		return ""
	}

	value, err := s.settingRepo.GetValue(ctx, key)
	if err != nil || value == "" {
		return defaultModel
	}
	return value
}

// GetLinuxDoConnectOAuthConfig 返回用于登录的"最终生效" LinuxDo Connect 配置。
//
// 优先级：
// - 若对应系统设置键存在，则覆盖 config.yaml/env 的值
// - 否则回退到 config.yaml/env 的值
func (s *SettingService) GetLinuxDoConnectOAuthConfig(ctx context.Context) (config.LinuxDoConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.LinuxDoConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}

	effective := s.cfg.LinuxDo

	keys := []string{
		SettingKeyLinuxDoConnectEnabled,
		SettingKeyLinuxDoConnectClientID,
		SettingKeyLinuxDoConnectClientSecret,
		SettingKeyLinuxDoConnectRedirectURL,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.LinuxDoConnectConfig{}, fmt.Errorf("get linuxdo connect settings: %w", err)
	}

	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}
	if !effective.Enabled {
		return config.LinuxDoConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}

	// 基础健壮性校验（避免把用户重定向到一个必然失败或不安全的 OAuth 流程里）。
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url not configured")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}

	method := strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod))
	switch method {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
	default:
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}

	return effective, nil
}

// GetDingTalkConnectOAuthConfig 返回用于登录的"最终生效" DingTalk Connect 配置。
//
// 优先级：
// - 若对应系统设置键存在，则覆盖 config.yaml/env 的值
// - 否则回退到 config.yaml/env 的值
func (s *SettingService) GetDingTalkConnectOAuthConfig(ctx context.Context) (config.DingTalkConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.DingTalkConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}

	effective := s.cfg.DingTalk

	keys := []string{
		SettingKeyDingTalkConnectEnabled,
		SettingKeyDingTalkConnectClientID,
		SettingKeyDingTalkConnectClientSecret,
		SettingKeyDingTalkConnectRedirectURL,
		SettingKeyDingTalkConnectCorpRestrictionPolicy,
		SettingKeyDingTalkConnectInternalCorpID,
		SettingKeyDingTalkConnectBypassRegistration,
		SettingKeyDingTalkConnectSyncCorpEmail,
		SettingKeyDingTalkConnectSyncDisplayName,
		SettingKeyDingTalkConnectSyncDept,
		SettingKeyDingTalkConnectSyncCorpEmailAttrKey,
		SettingKeyDingTalkConnectSyncDisplayNameAttrKey,
		SettingKeyDingTalkConnectSyncDeptAttrKey,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.DingTalkConnectConfig{}, fmt.Errorf("get dingtalk connect settings: %w", err)
	}

	if raw, ok := settings[SettingKeyDingTalkConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyDingTalkConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyDingTalkConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyDingTalkConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyDingTalkConnectCorpRestrictionPolicy]; ok && strings.TrimSpace(v) != "" {
		effective.CorpRestrictionPolicy = strings.TrimSpace(v)
	}
	effective.CorpRestrictionPolicy = coerceDeprecatedDingTalkCorpPolicy(effective.CorpRestrictionPolicy)
	if v, ok := settings[SettingKeyDingTalkConnectInternalCorpID]; ok && strings.TrimSpace(v) != "" {
		effective.InternalCorpID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyDingTalkConnectBypassRegistration]; ok && strings.TrimSpace(v) != "" {
		effective.BypassRegistration = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	// bypass_registration 仅在 internal_only 模式下有意义；其它策略下强制 false，
	// 以保证 OAuth callback 看到的 effective config 永远是一致状态。
	if effective.CorpRestrictionPolicy != "internal_only" {
		effective.BypassRegistration = false
	}

	if v, ok := settings[SettingKeyDingTalkConnectSyncCorpEmail]; ok && strings.TrimSpace(v) != "" {
		effective.SyncCorpEmail = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	if v, ok := settings[SettingKeyDingTalkConnectSyncDisplayName]; ok && strings.TrimSpace(v) != "" {
		effective.SyncDisplayName = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	if v, ok := settings[SettingKeyDingTalkConnectSyncDept]; ok && strings.TrimSpace(v) != "" {
		effective.SyncDept = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	// 身份同步三开关仅在 internal_only 模式下有意义；其它策略强制 false。
	if effective.CorpRestrictionPolicy != "internal_only" {
		effective.SyncCorpEmail = false
		effective.SyncDisplayName = false
		effective.SyncDept = false
	}

	// 身份同步目标 attr key（DB 空 → fallback 默认值）
	if v := strings.TrimSpace(settings[SettingKeyDingTalkConnectSyncCorpEmailAttrKey]); v != "" {
		effective.SyncCorpEmailAttrKey = v
	}
	if effective.SyncCorpEmailAttrKey == "" {
		effective.SyncCorpEmailAttrKey = "dingtalk_email"
	}
	if v := strings.TrimSpace(settings[SettingKeyDingTalkConnectSyncDisplayNameAttrKey]); v != "" {
		effective.SyncDisplayNameAttrKey = v
	}
	if effective.SyncDisplayNameAttrKey == "" {
		effective.SyncDisplayNameAttrKey = "dingtalk_name"
	}
	if v := strings.TrimSpace(settings[SettingKeyDingTalkConnectSyncDeptAttrKey]); v != "" {
		effective.SyncDeptAttrKey = v
	}
	if effective.SyncDeptAttrKey == "" {
		effective.SyncDeptAttrKey = "dingtalk_department"
	}

	if !effective.Enabled {
		return config.DingTalkConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "dingtalk oauth login is disabled")
	}

	// 基础健壮性校验（避免把用户重定向到一个必然失败或不安全的 OAuth 流程里）。
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth frontend redirect url not configured")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth frontend redirect url invalid")
	}
	if strings.TrimSpace(effective.ClientSecret) == "" {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "dingtalk oauth client secret not configured")
	}

	// 镜像 admin handler 行为：internal_only policy 隐式要求 AppType=internal
	if effective.CorpRestrictionPolicy == "internal_only" {
		effective.AppType = "internal"
	}

	if err := config.ValidateDingTalkConfig(effective); err != nil {
		return config.DingTalkConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", err.Error())
	}

	return effective, nil
}

// GetWeChatConnectOAuthConfig 返回用于登录的最终生效 WeChat Connect 配置。
//
// WeChat Connect 已回归 DB 系统设置模型，不再回退到 config/env。
func (s *SettingService) GetWeChatConnectOAuthConfig(ctx context.Context) (WeChatConnectOAuthConfig, error) {
	keys := []string{
		SettingKeyWeChatConnectEnabled,
		SettingKeyWeChatConnectAppID,
		SettingKeyWeChatConnectAppSecret,
		SettingKeyWeChatConnectOpenAppID,
		SettingKeyWeChatConnectOpenAppSecret,
		SettingKeyWeChatConnectMPAppID,
		SettingKeyWeChatConnectMPAppSecret,
		SettingKeyWeChatConnectMobileAppID,
		SettingKeyWeChatConnectMobileAppSecret,
		SettingKeyWeChatConnectOpenEnabled,
		SettingKeyWeChatConnectMPEnabled,
		SettingKeyWeChatConnectMobileEnabled,
		SettingKeyWeChatConnectMode,
		SettingKeyWeChatConnectScopes,
		SettingKeyWeChatConnectRedirectURL,
		SettingKeyWeChatConnectFrontendRedirectURL,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return WeChatConnectOAuthConfig{}, fmt.Errorf("get wechat connect settings: %w", err)
	}
	return s.parseWeChatConnectOAuthConfig(settings)
}

// GetOverloadCooldownSettings 获取529过载冷却配置
func (s *SettingService) GetOverloadCooldownSettings(ctx context.Context) (*OverloadCooldownSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOverloadCooldownSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultOverloadCooldownSettings(), nil
		}
		return nil, fmt.Errorf("get overload cooldown settings: %w", err)
	}
	if value == "" {
		return DefaultOverloadCooldownSettings(), nil
	}

	var settings OverloadCooldownSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultOverloadCooldownSettings(), nil
	}

	// 修正配置值范围
	if settings.CooldownMinutes < 1 {
		settings.CooldownMinutes = 1
	}
	if settings.CooldownMinutes > 120 {
		settings.CooldownMinutes = 120
	}

	return &settings, nil
}

// SetOverloadCooldownSettings 设置529过载冷却配置
func (s *SettingService) SetOverloadCooldownSettings(ctx context.Context, settings *OverloadCooldownSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// 禁用时修正为合法值即可，不拒绝请求
	if settings.CooldownMinutes < 1 || settings.CooldownMinutes > 120 {
		if settings.Enabled {
			return fmt.Errorf("cooldown_minutes must be between 1-120")
		}
		settings.CooldownMinutes = 10 // 禁用状态下归一化为默认值
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal overload cooldown settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyOverloadCooldownSettings, string(data))
}

// GetRateLimit429CooldownSettings 获取429默认回避配置
func (s *SettingService) GetRateLimit429CooldownSettings(ctx context.Context) (*RateLimit429CooldownSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRateLimit429CooldownSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultRateLimit429CooldownSettings(), nil
		}
		return nil, fmt.Errorf("get 429 cooldown settings: %w", err)
	}
	if value == "" {
		return DefaultRateLimit429CooldownSettings(), nil
	}

	var settings RateLimit429CooldownSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultRateLimit429CooldownSettings(), nil
	}

	if settings.CooldownSeconds < 1 {
		settings.CooldownSeconds = 1
	}
	if settings.CooldownSeconds > 7200 {
		settings.CooldownSeconds = 7200
	}

	return &settings, nil
}

// SetRateLimit429CooldownSettings 设置429默认回避配置
func (s *SettingService) SetRateLimit429CooldownSettings(ctx context.Context, settings *RateLimit429CooldownSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	if settings.CooldownSeconds < 1 || settings.CooldownSeconds > 7200 {
		if settings.Enabled {
			return fmt.Errorf("cooldown_seconds must be between 1-7200")
		}
		settings.CooldownSeconds = 5
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal 429 cooldown settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyRateLimit429CooldownSettings, string(data))
}

// GetOIDCConnectOAuthConfig 返回用于登录的“最终生效” OIDC 配置。
//
// 优先级：
// - 若对应系统设置键存在，则覆盖 config.yaml/env 的值
// - 否则回退到 config.yaml/env 的值
func (s *SettingService) GetOIDCConnectOAuthConfig(ctx context.Context) (config.OIDCConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.OIDCConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}

	effective := s.cfg.OIDC

	keys := []string{
		SettingKeyOIDCConnectEnabled,
		SettingKeyOIDCConnectProviderName,
		SettingKeyOIDCConnectClientID,
		SettingKeyOIDCConnectClientSecret,
		SettingKeyOIDCConnectIssuerURL,
		SettingKeyOIDCConnectDiscoveryURL,
		SettingKeyOIDCConnectAuthorizeURL,
		SettingKeyOIDCConnectTokenURL,
		SettingKeyOIDCConnectUserInfoURL,
		SettingKeyOIDCConnectJWKSURL,
		SettingKeyOIDCConnectScopes,
		SettingKeyOIDCConnectRedirectURL,
		SettingKeyOIDCConnectFrontendRedirectURL,
		SettingKeyOIDCConnectTokenAuthMethod,
		SettingKeyOIDCConnectUsePKCE,
		SettingKeyOIDCConnectValidateIDToken,
		SettingKeyOIDCConnectAllowedSigningAlgs,
		SettingKeyOIDCConnectClockSkewSeconds,
		SettingKeyOIDCConnectRequireEmailVerified,
		SettingKeyOIDCConnectUserInfoEmailPath,
		SettingKeyOIDCConnectUserInfoIDPath,
		SettingKeyOIDCConnectUserInfoUsernamePath,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.OIDCConnectConfig{}, fmt.Errorf("get oidc connect settings: %w", err)
	}

	if raw, ok := settings[SettingKeyOIDCConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyOIDCConnectProviderName]; ok && strings.TrimSpace(v) != "" {
		effective.ProviderName = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectIssuerURL]; ok && strings.TrimSpace(v) != "" {
		effective.IssuerURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectDiscoveryURL]; ok && strings.TrimSpace(v) != "" {
		effective.DiscoveryURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectAuthorizeURL]; ok && strings.TrimSpace(v) != "" {
		effective.AuthorizeURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectTokenURL]; ok && strings.TrimSpace(v) != "" {
		effective.TokenURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectUserInfoURL]; ok && strings.TrimSpace(v) != "" {
		effective.UserInfoURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectJWKSURL]; ok && strings.TrimSpace(v) != "" {
		effective.JWKSURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectScopes]; ok && strings.TrimSpace(v) != "" {
		effective.Scopes = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectFrontendRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.FrontendRedirectURL = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectTokenAuthMethod]; ok && strings.TrimSpace(v) != "" {
		effective.TokenAuthMethod = strings.ToLower(strings.TrimSpace(v))
	}
	if raw, ok := settings[SettingKeyOIDCConnectUsePKCE]; ok {
		effective.UsePKCE = raw == "true"
	} else {
		effective.UsePKCE = oidcUsePKCECompatibilityDefault(effective)
	}
	if raw, ok := settings[SettingKeyOIDCConnectValidateIDToken]; ok {
		effective.ValidateIDToken = raw == "true"
	} else {
		effective.ValidateIDToken = oidcValidateIDTokenCompatibilityDefault(effective)
	}
	if v, ok := settings[SettingKeyOIDCConnectAllowedSigningAlgs]; ok && strings.TrimSpace(v) != "" {
		effective.AllowedSigningAlgs = strings.TrimSpace(v)
	}
	if raw, ok := settings[SettingKeyOIDCConnectClockSkewSeconds]; ok && strings.TrimSpace(raw) != "" {
		if parsed, parseErr := strconv.Atoi(strings.TrimSpace(raw)); parseErr == nil {
			effective.ClockSkewSeconds = parsed
		}
	}
	if raw, ok := settings[SettingKeyOIDCConnectRequireEmailVerified]; ok {
		effective.RequireEmailVerified = raw == "true"
	}
	if v, ok := settings[SettingKeyOIDCConnectUserInfoEmailPath]; ok {
		effective.UserInfoEmailPath = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectUserInfoIDPath]; ok {
		effective.UserInfoIDPath = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyOIDCConnectUserInfoUsernamePath]; ok {
		effective.UserInfoUsernamePath = strings.TrimSpace(v)
	}

	if !effective.Enabled {
		return config.OIDCConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if strings.TrimSpace(effective.ProviderName) == "" {
		effective.ProviderName = "OIDC"
	}
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.IssuerURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth issuer url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url not configured")
	}
	if !scopesContainOpenID(effective.Scopes) {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth scopes must contain openid")
	}
	if effective.ClockSkewSeconds < 0 || effective.ClockSkewSeconds > 600 {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth clock skew must be between 0 and 600")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.IssuerURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth issuer url invalid")
	}

	discoveryURL := strings.TrimSpace(effective.DiscoveryURL)
	if discoveryURL == "" {
		discoveryURL = oidcDefaultDiscoveryURL(effective.IssuerURL)
		effective.DiscoveryURL = discoveryURL
	}
	if discoveryURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(discoveryURL); err != nil {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth discovery url invalid")
		}
	}

	needsDiscovery := strings.TrimSpace(effective.AuthorizeURL) == "" ||
		strings.TrimSpace(effective.TokenURL) == "" ||
		(effective.ValidateIDToken && strings.TrimSpace(effective.JWKSURL) == "")
	if needsDiscovery && discoveryURL != "" {
		metadata, resolveErr := oidcResolveProviderMetadata(ctx, discoveryURL)
		if resolveErr != nil {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth discovery resolve failed").WithCause(resolveErr)
		}
		if strings.TrimSpace(effective.AuthorizeURL) == "" {
			effective.AuthorizeURL = strings.TrimSpace(metadata.AuthorizationEndpoint)
		}
		if strings.TrimSpace(effective.TokenURL) == "" {
			effective.TokenURL = strings.TrimSpace(metadata.TokenEndpoint)
		}
		if strings.TrimSpace(effective.UserInfoURL) == "" {
			effective.UserInfoURL = strings.TrimSpace(metadata.UserInfoEndpoint)
		}
		if strings.TrimSpace(effective.JWKSURL) == "" {
			effective.JWKSURL = strings.TrimSpace(metadata.JWKSURI)
		}
	}

	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if v := strings.TrimSpace(effective.UserInfoURL); v != "" {
		if err := config.ValidateAbsoluteHTTPURL(v); err != nil {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
		}
	}
	if effective.ValidateIDToken {
		if strings.TrimSpace(effective.JWKSURL) == "" {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth jwks url not configured")
		}
		if strings.TrimSpace(effective.AllowedSigningAlgs) == "" {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth signing algs not configured")
		}
	}
	if v := strings.TrimSpace(effective.JWKSURL); v != "" {
		if err := config.ValidateAbsoluteHTTPURL(v); err != nil {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth jwks url invalid")
		}
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}

	method := strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod))
	switch method {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
	default:
		return config.OIDCConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}

	return effective, nil
}

func scopesContainOpenID(scopes string) bool {
	for _, scope := range strings.Fields(strings.ToLower(strings.TrimSpace(scopes))) {
		if scope == "openid" {
			return true
		}
	}
	return false
}

type oidcProviderMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func oidcDefaultDiscoveryURL(issuerURL string) string {
	issuerURL = strings.TrimSpace(issuerURL)
	if issuerURL == "" {
		return ""
	}
	return strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
}

func oidcResolveProviderMetadata(ctx context.Context, discoveryURL string) (*oidcProviderMetadata, error) {
	discoveryURL = strings.TrimSpace(discoveryURL)
	if discoveryURL == "" {
		return nil, fmt.Errorf("discovery url is empty")
	}

	resp, err := req.C().
		SetTimeout(15*time.Second).
		R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("request discovery document: %w", err)
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("discovery request failed: status=%d", resp.StatusCode)
	}

	metadata := &oidcProviderMetadata{}
	if err := json.Unmarshal(resp.Bytes(), metadata); err != nil {
		return nil, fmt.Errorf("parse discovery document: %w", err)
	}
	return metadata, nil
}

// GetStreamTimeoutSettings 获取流超时处理配置
func (s *SettingService) GetStreamTimeoutSettings(ctx context.Context) (*StreamTimeoutSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyStreamTimeoutSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultStreamTimeoutSettings(), nil
		}
		return nil, fmt.Errorf("get stream timeout settings: %w", err)
	}
	if value == "" {
		return DefaultStreamTimeoutSettings(), nil
	}

	var settings StreamTimeoutSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultStreamTimeoutSettings(), nil
	}

	// 验证并修正配置值
	if settings.TempUnschedMinutes < 1 {
		settings.TempUnschedMinutes = 1
	}
	if settings.TempUnschedMinutes > 60 {
		settings.TempUnschedMinutes = 60
	}
	if settings.ThresholdCount < 1 {
		settings.ThresholdCount = 1
	}
	if settings.ThresholdCount > 10 {
		settings.ThresholdCount = 10
	}
	if settings.ThresholdWindowMinutes < 1 {
		settings.ThresholdWindowMinutes = 1
	}
	if settings.ThresholdWindowMinutes > 60 {
		settings.ThresholdWindowMinutes = 60
	}

	// 验证 action
	switch settings.Action {
	case StreamTimeoutActionTempUnsched, StreamTimeoutActionError, StreamTimeoutActionNone:
		// valid
	default:
		settings.Action = StreamTimeoutActionTempUnsched
	}

	return &settings, nil
}

// IsUngroupedKeySchedulingAllowed 查询是否允许未分组 Key 调度
func (s *SettingService) IsUngroupedKeySchedulingAllowed(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAllowUngroupedKeyScheduling)
	if err != nil {
		return false // fail-closed: 查询失败时默认不允许
	}
	return value == "true"
}

// GetClaudeCodeVersionBounds 获取 Claude Code 版本号上下限要求
// 使用进程内 atomic.Value 缓存，60 秒 TTL，热路径零锁开销
// singleflight 防止缓存过期时 thundering herd
// 返回空字符串表示不做对应方向的版本检查
func (s *SettingService) GetClaudeCodeVersionBounds(ctx context.Context) (min, max string) {
	if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.min, cached.max
		}
	}
	// singleflight: 同一时刻只有一个 goroutine 查询 DB，其余复用结果
	type bounds struct{ min, max string }
	result, err, _ := versionBoundsSF.Do("version_bounds", func() (any, error) {
		// 二次检查，避免排队的 goroutine 重复查询
		if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok {
			if time.Now().UnixNano() < cached.expiresAt {
				return bounds{cached.min, cached.max}, nil
			}
		}
		// 使用独立 context：断开请求取消链，避免客户端断连导致空值被长期缓存
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), versionBoundsDBTimeout)
		defer cancel()
		values, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyMinClaudeCodeVersion,
			SettingKeyMaxClaudeCodeVersion,
		})
		if err != nil {
			// fail-open: DB 错误时不阻塞请求，但记录日志并使用短 TTL 快速重试
			slog.Warn("failed to get claude code version bounds setting, skipping version check", "error", err)
			versionBoundsCache.Store(&cachedVersionBounds{
				min:       "",
				max:       "",
				expiresAt: time.Now().Add(versionBoundsErrorTTL).UnixNano(),
			})
			return bounds{"", ""}, nil
		}
		b := bounds{
			min: values[SettingKeyMinClaudeCodeVersion],
			max: values[SettingKeyMaxClaudeCodeVersion],
		}
		versionBoundsCache.Store(&cachedVersionBounds{
			min:       b.min,
			max:       b.max,
			expiresAt: time.Now().Add(versionBoundsCacheTTL).UnixNano(),
		})
		return b, nil
	})
	if err != nil {
		return "", ""
	}
	b, ok := result.(bounds)
	if !ok {
		return "", ""
	}
	return b.min, b.max
}

// GetOpenAIQuotaAutoPauseSettings returns the current global default quota auto-pause
// settings. It is invoked on the OpenAI scheduling hot path (once per request) and is
// therefore designed to never block on the DB:
//
//   - Fresh cached value → returned immediately.
//   - Stale or empty cache → the last known value is returned, and a background
//     goroutine refreshes the cache via singleflight (stale-while-revalidate).
//   - First call with no cache yet → zero defaults are returned and the same async
//     refresh is kicked off; the next call gets the freshly populated value.
//
// Callers that need the freshly persisted value synchronously (tests, post-update
// confirmation, optional startup warm-up) should call WarmOpenAIQuotaAutoPauseSettings.
func (s *SettingService) GetOpenAIQuotaAutoPauseSettings(ctx context.Context) OpsOpenAIAccountQuotaAutoPauseSettings {
	if s == nil {
		return OpsOpenAIAccountQuotaAutoPauseSettings{}
	}
	cached, _ := s.openAIQuotaAutoPauseSettingsCache.Load().(*cachedOpenAIQuotaAutoPauseSettings)
	now := time.Now().UnixNano()
	if cached != nil && now < cached.expiresAt {
		return cached.settings
	}
	// Stale or unset: trigger background refresh without blocking this request.
	// singleflight.DoChan dedupes concurrent refreshes; we deliberately ignore the
	// returned channel — the result is observable via the atomic cache.
	s.openAIQuotaAutoPauseSettingsSF.DoChan(openAIQuotaAutoPauseSettingsRefreshKey, func() (any, error) {
		s.refreshOpenAIQuotaAutoPauseSettings(context.Background())
		return nil, nil
	})
	if cached != nil {
		return cached.settings // serve stale value while revalidating
	}
	return OpsOpenAIAccountQuotaAutoPauseSettings{}
}

// WarmOpenAIQuotaAutoPauseSettings synchronously loads the quota auto-pause settings
// into the in-memory cache. Useful for application startup (so the first request hits
// a warm cache) and for tests that need deterministic reads immediately after
// constructing the service.
func (s *SettingService) WarmOpenAIQuotaAutoPauseSettings(ctx context.Context) OpsOpenAIAccountQuotaAutoPauseSettings {
	if s == nil {
		return OpsOpenAIAccountQuotaAutoPauseSettings{}
	}
	s.refreshOpenAIQuotaAutoPauseSettings(ctx)
	cached, _ := s.openAIQuotaAutoPauseSettingsCache.Load().(*cachedOpenAIQuotaAutoPauseSettings)
	if cached == nil {
		return OpsOpenAIAccountQuotaAutoPauseSettings{}
	}
	return cached.settings
}

// refreshOpenAIQuotaAutoPauseSettings reads the latest settings from the DB and stores
// them into the in-memory cache. On error it stores the prior value (or zero defaults
// if nothing is cached yet) with the shorter error TTL so the next refresh comes
// sooner. Always uses its own timeout-bounded context to keep refresh latency
// predictable regardless of the caller.
func (s *SettingService) refreshOpenAIQuotaAutoPauseSettings(ctx context.Context) {
	if s == nil || s.settingRepo == nil {
		return
	}
	dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIQuotaAutoPauseSettingsDBTimeout)
	defer cancel()

	settings := OpsOpenAIAccountQuotaAutoPauseSettings{}
	ttl := openAIQuotaAutoPauseSettingsCacheTTL
	raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyOpsAdvancedSettings)
	if err == nil {
		cfg := defaultOpsAdvancedSettings()
		if strings.TrimSpace(raw) != "" {
			if jsonErr := json.Unmarshal([]byte(raw), cfg); jsonErr == nil {
				normalizeOpsAdvancedSettings(cfg)
			}
		}
		settings = cfg.OpenAIAccountQuotaAutoPause
	} else if !errors.Is(err, ErrSettingNotFound) {
		// Real error: keep serving prior value but refresh sooner.
		if prior, _ := s.openAIQuotaAutoPauseSettingsCache.Load().(*cachedOpenAIQuotaAutoPauseSettings); prior != nil {
			settings = prior.settings
		}
		ttl = openAIQuotaAutoPauseSettingsErrorTTL
	}

	s.openAIQuotaAutoPauseSettingsCache.Store(&cachedOpenAIQuotaAutoPauseSettings{
		settings:  settings,
		expiresAt: time.Now().Add(ttl).UnixNano(),
	})
}

// SetOpenAIQuotaAutoPauseSettings writes the given settings directly into the in-memory
// cache. Called from settings-write code paths so that the next read reflects the new
// value immediately, without waiting for the background refresh.
func (s *SettingService) SetOpenAIQuotaAutoPauseSettings(settings OpsOpenAIAccountQuotaAutoPauseSettings) {
	if s == nil {
		return
	}
	s.openAIQuotaAutoPauseSettingsCache.Store(&cachedOpenAIQuotaAutoPauseSettings{
		settings:  settings,
		expiresAt: time.Now().Add(openAIQuotaAutoPauseSettingsCacheTTL).UnixNano(),
	})
}

// GetRectifierSettings 获取请求整流器配置
func (s *SettingService) GetRectifierSettings(ctx context.Context) (*RectifierSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRectifierSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultRectifierSettings(), nil
		}
		return nil, fmt.Errorf("get rectifier settings: %w", err)
	}
	if value == "" {
		return DefaultRectifierSettings(), nil
	}

	var settings RectifierSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultRectifierSettings(), nil
	}

	return &settings, nil
}

// SetRectifierSettings 设置请求整流器配置
func (s *SettingService) SetRectifierSettings(ctx context.Context, settings *RectifierSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal rectifier settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyRectifierSettings, string(data))
}

// IsSignatureRectifierEnabled 判断签名整流是否启用（总开关 && 签名子开关）
func (s *SettingService) IsSignatureRectifierEnabled(ctx context.Context) bool {
	settings, err := s.GetRectifierSettings(ctx)
	if err != nil {
		return true // fail-open: 查询失败时默认启用
	}
	return settings.Enabled && settings.ThinkingSignatureEnabled
}

// IsBudgetRectifierEnabled 判断 Budget 整流是否启用（总开关 && Budget 子开关）
func (s *SettingService) IsBudgetRectifierEnabled(ctx context.Context) bool {
	settings, err := s.GetRectifierSettings(ctx)
	if err != nil {
		return true // fail-open: 查询失败时默认启用
	}
	return settings.Enabled && settings.ThinkingBudgetEnabled
}

// GetBetaPolicySettings 获取 Beta 策略配置
func (s *SettingService) GetBetaPolicySettings(ctx context.Context) (*BetaPolicySettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyBetaPolicySettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultBetaPolicySettings(), nil
		}
		return nil, fmt.Errorf("get beta policy settings: %w", err)
	}
	if value == "" {
		return DefaultBetaPolicySettings(), nil
	}

	var settings BetaPolicySettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultBetaPolicySettings(), nil
	}

	return &settings, nil
}

// SetBetaPolicySettings 设置 Beta 策略配置
func (s *SettingService) SetBetaPolicySettings(ctx context.Context, settings *BetaPolicySettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	validActions := map[string]bool{
		BetaPolicyActionPass: true, BetaPolicyActionFilter: true, BetaPolicyActionBlock: true,
	}
	validScopes := map[string]bool{
		BetaPolicyScopeAll: true, BetaPolicyScopeOAuth: true, BetaPolicyScopeAPIKey: true, BetaPolicyScopeBedrock: true,
	}

	for i, rule := range settings.Rules {
		if rule.BetaToken == "" {
			return fmt.Errorf("rule[%d]: beta_token cannot be empty", i)
		}
		if !validActions[rule.Action] {
			return fmt.Errorf("rule[%d]: invalid action %q", i, rule.Action)
		}
		if !validScopes[rule.Scope] {
			return fmt.Errorf("rule[%d]: invalid scope %q", i, rule.Scope)
		}
		// Validate model_whitelist patterns
		for j, pattern := range rule.ModelWhitelist {
			trimmed := strings.TrimSpace(pattern)
			if trimmed == "" {
				return fmt.Errorf("rule[%d]: model_whitelist[%d] cannot be empty", i, j)
			}
			settings.Rules[i].ModelWhitelist[j] = trimmed
		}
		// Validate fallback_action
		if rule.FallbackAction != "" && !validActions[rule.FallbackAction] {
			return fmt.Errorf("rule[%d]: invalid fallback_action %q", i, rule.FallbackAction)
		}
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal beta policy settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyBetaPolicySettings, string(data))
}

// GetOpenAIFastPolicySettings 获取 OpenAI fast 策略配置
func (s *SettingService) GetOpenAIFastPolicySettings(ctx context.Context) (*OpenAIFastPolicySettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIFastPolicySettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultOpenAIFastPolicySettings(), nil
		}
		return nil, fmt.Errorf("get openai fast policy settings: %w", err)
	}
	if value == "" {
		return DefaultOpenAIFastPolicySettings(), nil
	}

	var settings OpenAIFastPolicySettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		// JSON 损坏时静默 fallback 到默认配置会让策略意外失效（管理员配
		// 置的 block/filter 规则被忽略）。记录 Warn 让运维能在出现异常
		// 行为时定位到 settings 表里的脏数据。
		slog.Warn("failed to unmarshal openai fast policy settings, falling back to defaults",
			"error", err,
			"key", SettingKeyOpenAIFastPolicySettings)
		return DefaultOpenAIFastPolicySettings(), nil
	}

	return &settings, nil
}

// SetOpenAIFastPolicySettings 设置 OpenAI fast 策略配置
func (s *SettingService) SetOpenAIFastPolicySettings(ctx context.Context, settings *OpenAIFastPolicySettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	validActions := map[string]bool{
		BetaPolicyActionPass: true, BetaPolicyActionFilter: true, BetaPolicyActionBlock: true,
	}
	validScopes := map[string]bool{
		BetaPolicyScopeAll: true, BetaPolicyScopeOAuth: true, BetaPolicyScopeAPIKey: true, BetaPolicyScopeBedrock: true,
	}
	validTiers := map[string]bool{
		OpenAIFastTierAny: true, OpenAIFastTierPriority: true, OpenAIFastTierFlex: true,
	}

	for i, rule := range settings.Rules {
		tier := strings.ToLower(strings.TrimSpace(rule.ServiceTier))
		if tier == "" {
			tier = OpenAIFastTierAny
		}
		if !validTiers[tier] {
			return fmt.Errorf("rule[%d]: invalid service_tier %q", i, rule.ServiceTier)
		}
		settings.Rules[i].ServiceTier = tier
		if !validActions[rule.Action] {
			return fmt.Errorf("rule[%d]: invalid action %q", i, rule.Action)
		}
		if !validScopes[rule.Scope] {
			return fmt.Errorf("rule[%d]: invalid scope %q", i, rule.Scope)
		}
		for j, pattern := range rule.ModelWhitelist {
			trimmed := strings.TrimSpace(pattern)
			if trimmed == "" {
				return fmt.Errorf("rule[%d]: model_whitelist[%d] cannot be empty", i, j)
			}
			settings.Rules[i].ModelWhitelist[j] = trimmed
		}
		if rule.FallbackAction != "" && !validActions[rule.FallbackAction] {
			return fmt.Errorf("rule[%d]: invalid fallback_action %q", i, rule.FallbackAction)
		}
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal openai fast policy settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyOpenAIFastPolicySettings, string(data))
}

// SetStreamTimeoutSettings 设置流超时处理配置
func (s *SettingService) SetStreamTimeoutSettings(ctx context.Context, settings *StreamTimeoutSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	// 验证配置值
	if settings.TempUnschedMinutes < 1 || settings.TempUnschedMinutes > 60 {
		return fmt.Errorf("temp_unsched_minutes must be between 1-60")
	}
	if settings.ThresholdCount < 1 || settings.ThresholdCount > 10 {
		return fmt.Errorf("threshold_count must be between 1-10")
	}
	if settings.ThresholdWindowMinutes < 1 || settings.ThresholdWindowMinutes > 60 {
		return fmt.Errorf("threshold_window_minutes must be between 1-60")
	}

	switch settings.Action {
	case StreamTimeoutActionTempUnsched, StreamTimeoutActionError, StreamTimeoutActionNone:
		// valid
	default:
		return fmt.Errorf("invalid action: %s", settings.Action)
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal stream timeout settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyStreamTimeoutSettings, string(data))
}

// PayloadLoggingSettings 报文审计记录配置
type PayloadLoggingSettings struct {
	Enabled         bool  `json:"enabled"`           // 是否开启报文记录
	MaxRequestSize  int64 `json:"max_request_size"`  // 请求体最大保存字节
	MaxResponseSize int64 `json:"max_response_size"` // 响应体最大保存字节
	RetentionDays   int   `json:"retention_days"`    // 保留天数，0=不自动清理
}

// DefaultPayloadLoggingSettings 返回默认配置
func DefaultPayloadLoggingSettings() *PayloadLoggingSettings {
	return &PayloadLoggingSettings{
		Enabled:         false,
		MaxRequestSize:  65536,
		MaxResponseSize: 65536,
		RetentionDays:   7,
	}
}

// GetPayloadLoggingSettings 获取报文记录配置
// 使用进程内 atomic.Value 缓存，60 秒 TTL，热路径零锁开销
// singleflight 防止缓存过期时 thundering herd
func (s *SettingService) GetPayloadLoggingSettings(ctx context.Context) (*PayloadLoggingSettings, error) {
	if cached, ok := payloadLoggingCache.Load().(*cachedPayloadLoggingSettings); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			if cached.settings != nil {
				slog.Debug("payload logging settings resolved",
					"source", "cache",
					"enabled", cached.settings.Enabled,
					"max_request_size", cached.settings.MaxRequestSize,
					"max_response_size", cached.settings.MaxResponseSize,
					"retention_days", cached.settings.RetentionDays,
					"expires_at_unix_nano", cached.expiresAt,
				)
			}
			return cached.settings, nil
		}
	}

	val, _, _ := payloadLoggingSF.Do("payload_logging", func() (any, error) {
		if cached, ok := payloadLoggingCache.Load().(*cachedPayloadLoggingSettings); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.settings, nil
			}
		}

		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), payloadLoggingDBTimeout)
		defer cancel()

		settings, err := s.loadPayloadLoggingSettingsFromDB(dbCtx)
		if err != nil {
			slog.Warn("failed to get payload logging settings", "error", err)
			defaults := DefaultPayloadLoggingSettings()
			payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
				settings:  defaults,
				expiresAt: time.Now().Add(payloadLoggingErrorTTL).UnixNano(),
			})
			return defaults, nil
		}

		payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
			settings:  settings,
			expiresAt: time.Now().Add(payloadLoggingCacheTTL).UnixNano(),
		})
		slog.Info("payload logging settings resolved",
			"source", "db",
			"enabled", settings.Enabled,
			"max_request_size", settings.MaxRequestSize,
			"max_response_size", settings.MaxResponseSize,
			"retention_days", settings.RetentionDays,
			"cache_ttl_seconds", payloadLoggingCacheTTL.Seconds(),
		)
		return settings, nil
	})

	if s, ok := val.(*PayloadLoggingSettings); ok {
		return s, nil
	}
	return DefaultPayloadLoggingSettings(), nil
}

// loadPayloadLoggingSettingsFromDB 从数据库加载报文记录配置（内部方法，供缓存刷新使用）
func (s *SettingService) loadPayloadLoggingSettingsFromDB(ctx context.Context) (*PayloadLoggingSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPayloadLoggingSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultPayloadLoggingSettings(), nil
		}
		return nil, fmt.Errorf("get payload logging settings: %w", err)
	}
	if value == "" {
		return DefaultPayloadLoggingSettings(), nil
	}

	var settings PayloadLoggingSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultPayloadLoggingSettings(), nil
	}

	if settings.MaxRequestSize < 1024 {
		settings.MaxRequestSize = 1024
	}
	if settings.MaxRequestSize > 524288 {
		settings.MaxRequestSize = 524288
	}
	if settings.MaxResponseSize < 1024 {
		settings.MaxResponseSize = 1024
	}
	if settings.MaxResponseSize > 524288 {
		settings.MaxResponseSize = 524288
	}
	if settings.RetentionDays < 0 {
		settings.RetentionDays = 0
	}
	if settings.RetentionDays > 365 {
		settings.RetentionDays = 365
	}

	return &settings, nil
}

// SetPayloadLoggingSettings 保存报文记录配置
func (s *SettingService) SetPayloadLoggingSettings(ctx context.Context, settings *PayloadLoggingSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	if settings.MaxRequestSize < 1024 || settings.MaxRequestSize > 524288 {
		return fmt.Errorf("max_request_size must be between 1024-524288")
	}
	if settings.MaxResponseSize < 1024 || settings.MaxResponseSize > 524288 {
		return fmt.Errorf("max_response_size must be between 1024-524288")
	}
	if settings.RetentionDays < 0 || settings.RetentionDays > 365 {
		return fmt.Errorf("retention_days must be between 0-365")
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal payload logging settings: %w", err)
	}

	if err := s.settingRepo.Set(ctx, SettingKeyPayloadLoggingSettings, string(data)); err != nil {
		return err
	}

	payloadLoggingSF.Forget("payload_logging")
	payloadLoggingCache.Store(&cachedPayloadLoggingSettings{
		settings:  settings,
		expiresAt: time.Now().Add(payloadLoggingCacheTTL).UnixNano(),
	})
	slog.Info("payload logging settings updated",
		"enabled", settings.Enabled,
		"max_request_size", settings.MaxRequestSize,
		"max_response_size", settings.MaxResponseSize,
		"retention_days", settings.RetentionDays,
		"cache_scope", "local_process_only",
		"cache_ttl_seconds", payloadLoggingCacheTTL.Seconds(),
	)
	return nil
}

// IsPayloadLoggingEnabled 快捷判断（用于网关热路径）
func (s *SettingService) IsPayloadLoggingEnabled(ctx context.Context) bool {
	settings, err := s.GetPayloadLoggingSettings(ctx)
	if err != nil {
		return false
	}
	return settings.Enabled
}
// GetDefaultPlatformQuotas 读取系统全局 platform quota JSON key，返回全部允许平台 x 3 window 的设置。
// 永远返回包含全部允许 platform key 的 map（值可能为零值/nil 字段，表示"上层未配置 = 不限制"）。
//
// 使用单个 JSON key（default_platform_quotas），一次 DB roundtrip，消除旧 12-KV 格式的 N+1 问题。
// 容错语义：取值失败或 unmarshal 失败 → 返回补齐全部允许平台 key 的空 map（fail-open，注册不被阻断）。
func (s *SettingService) GetDefaultPlatformQuotas(ctx context.Context) (map[string]*DefaultPlatformQuotaSetting, error) {
	out := make(map[string]*DefaultPlatformQuotaSetting, len(AllowedQuotaPlatforms))
	for _, platform := range AllowedQuotaPlatforms {
		out[platform] = &DefaultPlatformQuotaSetting{}
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultPlatformQuotas)
	if err != nil || raw == "" {
		return out, nil // 无配置 = 全部不限制
	}
	parsed := map[string]*DefaultPlatformQuotaSetting{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		slog.Warn("[Setting] unmarshal default_platform_quotas failed (fail-open)", "error", err)
		return out, nil
	}
	for _, platform := range AllowedQuotaPlatforms {
		if v := parsed[platform]; v != nil {
			out[platform] = v
		}
	}
	return out, nil // 补齐全部允许 platform key，保持与旧实现一致的下游契约
}

// GetAuthSourcePlatformQuotas 读取指定 auth source 的 platform quota 覆盖（仅返回有配置的平台，override 语义）。
func (s *SettingService) GetAuthSourcePlatformQuotas(ctx context.Context, source string) map[string]*DefaultPlatformQuotaSetting {
	out := map[string]*DefaultPlatformQuotaSetting{}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAuthSourcePlatformQuotas(source))
	if err != nil || raw == "" {
		return out // 无 override
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		slog.Warn("[Setting] unmarshal auth source platform quotas failed (fail-open)", "source", source, "error", err)
		return map[string]*DefaultPlatformQuotaSetting{}
	}
	return out // 仅含已配置平台，保持 override 语义
}

// mergePlatformQuotaDefaults 按字段级 patch：src 中非 nil 字段覆盖 dst。
// 区分 nil（"未配置"，保留 dst）vs &0.0（"显式禁用"，覆盖 dst 为 0）
func mergePlatformQuotaDefaults(dst, src *DefaultPlatformQuotaSetting) {
	if src == nil || dst == nil {
		return
	}
	if src.DailyLimitUSD != nil {
		dst.DailyLimitUSD = src.DailyLimitUSD
	}
	if src.WeeklyLimitUSD != nil {
		dst.WeeklyLimitUSD = src.WeeklyLimitUSD
	}
	if src.MonthlyLimitUSD != nil {
		dst.MonthlyLimitUSD = src.MonthlyLimitUSD
	}
}
