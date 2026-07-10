package admin

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// semverPattern 预编译 semver 格式校验正则
var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// menuItemIDPattern validates custom menu item IDs: alphanumeric, hyphens, underscores only.
var menuItemIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// generateMenuItemID generates a short random hex ID for a custom menu item.
func generateMenuItemID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate menu item ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func scopesContainOpenID(scopes string) bool {
	for _, scope := range strings.Fields(strings.ToLower(strings.TrimSpace(scopes))) {
		if scope == "openid" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// SettingHandler 系统设置处理器
type SettingHandler struct {
	settingService           *service.SettingService
	emailService             *service.EmailService
	turnstileService         *service.TurnstileService
	opsService               *service.OpsService
	paymentConfigService     *service.PaymentConfigService
	paymentService           *service.PaymentService
	userAttributeService     *service.UserAttributeService
	notificationEmailService *service.NotificationEmailService
}

// NewSettingHandler 创建系统设置处理器
func NewSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, opsService *service.OpsService, paymentConfigService *service.PaymentConfigService, paymentService *service.PaymentService, userAttributeService *service.UserAttributeService) *SettingHandler {
	return &SettingHandler{
		settingService:       settingService,
		emailService:         emailService,
		turnstileService:     turnstileService,
		opsService:           opsService,
		paymentConfigService: paymentConfigService,
		paymentService:       paymentService,
		userAttributeService: userAttributeService,
	}
}

// SetNotificationEmailService attaches the notification template service without changing
// the constructor signature used by existing unit tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// GetSettings 获取所有系统设置
// GET /api/v1/admin/settings
func (h *SettingHandler) GetSettings(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	authSourceDefaults, err := h.settingService.GetAuthSourceDefaultSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Check if ops monitoring is enabled (respects config.ops.enabled)
	opsEnabled := h.opsService != nil && h.opsService.IsMonitoringEnabled(c.Request.Context())
	defaultSubscriptions := make([]dto.DefaultSubscriptionSetting, 0, len(settings.DefaultSubscriptions))
	for _, sub := range settings.DefaultSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, dto.DefaultSubscriptionSetting{
			GroupID:      sub.GroupID,
			ValidityDays: sub.ValidityDays,
		})
	}

	// Load payment config
	var paymentCfg *service.PaymentConfig
	if h.paymentConfigService != nil {
		paymentCfg, _ = h.paymentConfigService.GetPaymentConfig(c.Request.Context())
	}
	if paymentCfg == nil {
		paymentCfg = &service.PaymentConfig{}
	}

	payload := dto.SystemSettings{
		RegistrationEnabled:                                    settings.RegistrationEnabled,
		EmailVerifyEnabled:                                     settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:                       settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                                       settings.PromoCodeEnabled,
		PasswordResetEnabled:                                   settings.PasswordResetEnabled,
		FrontendURL:                                            settings.FrontendURL,
		InvitationCodeEnabled:                                  settings.InvitationCodeEnabled,
		TotpEnabled:                                            settings.TotpEnabled,
		TotpEncryptionKeyConfigured:                            h.settingService.IsTotpEncryptionKeyConfigured(),
		LoginAgreementEnabled:                                  settings.LoginAgreementEnabled,
		LoginAgreementMode:                                     settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:                                settings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments:                                loginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		SMTPHost:                                               settings.SMTPHost,
		SMTPPort:                                               settings.SMTPPort,
		SMTPUsername:                                           settings.SMTPUsername,
		SMTPPasswordConfigured:                                 settings.SMTPPasswordConfigured,
		SMTPFrom:                                               settings.SMTPFrom,
		SMTPFromName:                                           settings.SMTPFromName,
		SMTPUseTLS:                                             settings.SMTPUseTLS,
		TurnstileEnabled:                                       settings.TurnstileEnabled,
		TurnstileSiteKey:                                       settings.TurnstileSiteKey,
		TurnstileSecretKeyConfigured:                           settings.TurnstileSecretKeyConfigured,
		APIKeyACLTrustForwardedIP:                              settings.APIKeyACLTrustForwardedIP,
		LinuxDoConnectEnabled:                                  settings.LinuxDoConnectEnabled,
		LinuxDoConnectClientID:                                 settings.LinuxDoConnectClientID,
		LinuxDoConnectClientSecretConfigured:                   settings.LinuxDoConnectClientSecretConfigured,
		LinuxDoConnectRedirectURL:                              settings.LinuxDoConnectRedirectURL,
		DingTalkConnectEnabled:                                 settings.DingTalkConnectEnabled,
		DingTalkConnectClientID:                                settings.DingTalkConnectClientID,
		DingTalkConnectClientSecretConfigured:                  settings.DingTalkConnectClientSecretConfigured,
		DingTalkConnectRedirectURL:                             settings.DingTalkConnectRedirectURL,
		DingTalkConnectCorpRestrictionPolicy:                   settings.DingTalkConnectCorpRestrictionPolicy,
		DingTalkConnectInternalCorpID:                          settings.DingTalkConnectInternalCorpID,
		DingTalkConnectBypassRegistration:                      settings.DingTalkConnectBypassRegistration,
		DingTalkConnectSyncCorpEmail:                           settings.DingTalkConnectSyncCorpEmail,
		DingTalkConnectSyncDisplayName:                         settings.DingTalkConnectSyncDisplayName,
		DingTalkConnectSyncDept:                                settings.DingTalkConnectSyncDept,
		DingTalkConnectSyncCorpEmailAttrKey:                    settings.DingTalkConnectSyncCorpEmailAttrKey,
		DingTalkConnectSyncDisplayNameAttrKey:                  settings.DingTalkConnectSyncDisplayNameAttrKey,
		DingTalkConnectSyncDeptAttrKey:                         settings.DingTalkConnectSyncDeptAttrKey,
		DingTalkConnectSyncCorpEmailAttrName:                   settings.DingTalkConnectSyncCorpEmailAttrName,
		DingTalkConnectSyncDisplayNameAttrName:                 settings.DingTalkConnectSyncDisplayNameAttrName,
		DingTalkConnectSyncDeptAttrName:                        settings.DingTalkConnectSyncDeptAttrName,
		WeChatConnectEnabled:                                   settings.WeChatConnectEnabled,
		WeChatConnectAppID:                                     settings.WeChatConnectAppID,
		WeChatConnectAppSecretConfigured:                       settings.WeChatConnectAppSecretConfigured,
		WeChatConnectOpenAppID:                                 settings.WeChatConnectOpenAppID,
		WeChatConnectOpenAppSecretConfigured:                   settings.WeChatConnectOpenAppSecretConfigured,
		WeChatConnectMPAppID:                                   settings.WeChatConnectMPAppID,
		WeChatConnectMPAppSecretConfigured:                     settings.WeChatConnectMPAppSecretConfigured,
		WeChatConnectMobileAppID:                               settings.WeChatConnectMobileAppID,
		WeChatConnectMobileAppSecretConfigured:                 settings.WeChatConnectMobileAppSecretConfigured,
		WeChatConnectOpenEnabled:                               settings.WeChatConnectOpenEnabled,
		WeChatConnectMPEnabled:                                 settings.WeChatConnectMPEnabled,
		WeChatConnectMobileEnabled:                             settings.WeChatConnectMobileEnabled,
		WeChatConnectMode:                                      settings.WeChatConnectMode,
		WeChatConnectScopes:                                    settings.WeChatConnectScopes,
		WeChatConnectRedirectURL:                               settings.WeChatConnectRedirectURL,
		WeChatConnectFrontendRedirectURL:                       settings.WeChatConnectFrontendRedirectURL,
		OIDCConnectEnabled:                                     settings.OIDCConnectEnabled,
		OIDCConnectProviderName:                                settings.OIDCConnectProviderName,
		OIDCConnectClientID:                                    settings.OIDCConnectClientID,
		OIDCConnectClientSecretConfigured:                      settings.OIDCConnectClientSecretConfigured,
		OIDCConnectIssuerURL:                                   settings.OIDCConnectIssuerURL,
		OIDCConnectDiscoveryURL:                                settings.OIDCConnectDiscoveryURL,
		OIDCConnectAuthorizeURL:                                settings.OIDCConnectAuthorizeURL,
		OIDCConnectTokenURL:                                    settings.OIDCConnectTokenURL,
		OIDCConnectUserInfoURL:                                 settings.OIDCConnectUserInfoURL,
		OIDCConnectJWKSURL:                                     settings.OIDCConnectJWKSURL,
		OIDCConnectScopes:                                      settings.OIDCConnectScopes,
		OIDCConnectRedirectURL:                                 settings.OIDCConnectRedirectURL,
		OIDCConnectFrontendRedirectURL:                         settings.OIDCConnectFrontendRedirectURL,
		OIDCConnectTokenAuthMethod:                             settings.OIDCConnectTokenAuthMethod,
		OIDCConnectUsePKCE:                                     settings.OIDCConnectUsePKCE,
		OIDCConnectValidateIDToken:                             settings.OIDCConnectValidateIDToken,
		OIDCConnectAllowedSigningAlgs:                          settings.OIDCConnectAllowedSigningAlgs,
		OIDCConnectClockSkewSeconds:                            settings.OIDCConnectClockSkewSeconds,
		OIDCConnectRequireEmailVerified:                        settings.OIDCConnectRequireEmailVerified,
		OIDCConnectUserInfoEmailPath:                           settings.OIDCConnectUserInfoEmailPath,
		OIDCConnectUserInfoIDPath:                              settings.OIDCConnectUserInfoIDPath,
		OIDCConnectUserInfoUsernamePath:                        settings.OIDCConnectUserInfoUsernamePath,
		GitHubOAuthEnabled:                                     settings.GitHubOAuthEnabled,
		GitHubOAuthClientID:                                    settings.GitHubOAuthClientID,
		GitHubOAuthClientSecretConfigured:                      settings.GitHubOAuthClientSecretConfigured,
		GitHubOAuthRedirectURL:                                 settings.GitHubOAuthRedirectURL,
		GitHubOAuthFrontendRedirectURL:                         settings.GitHubOAuthFrontendRedirectURL,
		GoogleOAuthEnabled:                                     settings.GoogleOAuthEnabled,
		GoogleOAuthClientID:                                    settings.GoogleOAuthClientID,
		GoogleOAuthClientSecretConfigured:                      settings.GoogleOAuthClientSecretConfigured,
		GoogleOAuthRedirectURL:                                 settings.GoogleOAuthRedirectURL,
		GoogleOAuthFrontendRedirectURL:                         settings.GoogleOAuthFrontendRedirectURL,
		SiteName:                                               settings.SiteName,
		SiteLogo:                                               settings.SiteLogo,
		SiteSubtitle:                                           settings.SiteSubtitle,
		APIBaseURL:                                             settings.APIBaseURL,
		ContactInfo:                                            settings.ContactInfo,
		DocURL:                                                 settings.DocURL,
		HomeContent:                                            settings.HomeContent,
		HideCcsImportButton:                                    settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:                            settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:                                settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                                   settings.TableDefaultPageSize,
		TablePageSizeOptions:                                   settings.TablePageSizeOptions,
		CustomMenuItems:                                        dto.ParseCustomMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                                        dto.ParseCustomEndpoints(settings.CustomEndpoints),
		DefaultConcurrency:                                     settings.DefaultConcurrency,
		DefaultBalance:                                         settings.DefaultBalance,
		RiskControlEnabled:                                     settings.RiskControlEnabled,
		CyberSessionBlockEnabled:                               settings.CyberSessionBlockEnabled,
		CyberSessionBlockTTLSeconds:                            settings.CyberSessionBlockTTLSeconds,
		AffiliateRebateRate:                                    settings.AffiliateRebateRate,
		AffiliateRebateFreezeHours:                             settings.AffiliateRebateFreezeHours,
		AffiliateRebateDurationDays:                            settings.AffiliateRebateDurationDays,
		AffiliateRebatePerInviteeCap:                           settings.AffiliateRebatePerInviteeCap,
		DefaultUserRPMLimit:                                    settings.DefaultUserRPMLimit,
		DefaultSubscriptions:                                   defaultSubscriptions,
		EnableModelFallback:                                    settings.EnableModelFallback,
		FallbackModelAnthropic:                                 settings.FallbackModelAnthropic,
		FallbackModelOpenAI:                                    settings.FallbackModelOpenAI,
		FallbackModelGemini:                                    settings.FallbackModelGemini,
		FallbackModelAntigravity:                               settings.FallbackModelAntigravity,
		EnableIdentityPatch:                                    settings.EnableIdentityPatch,
		IdentityPatchPrompt:                                    settings.IdentityPatchPrompt,
		OpsMonitoringEnabled:                                   opsEnabled && settings.OpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:                           settings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                                    settings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:                              settings.OpsMetricsIntervalSeconds,
		MinClaudeCodeVersion:                                   settings.MinClaudeCodeVersion,
		MaxClaudeCodeVersion:                                   settings.MaxClaudeCodeVersion,
		AllowUngroupedKeyScheduling:                            settings.AllowUngroupedKeyScheduling,
		BackendModeEnabled:                                     settings.BackendModeEnabled,
		EnableFingerprintUnification:                           settings.EnableFingerprintUnification,
		EnableMetadataPassthrough:                              settings.EnableMetadataPassthrough,
		EnableCCHSigning:                                       settings.EnableCCHSigning,
		EnableClaudeOAuthSystemPromptInjection:                 settings.EnableClaudeOAuthSystemPromptInjection,
		ClaudeOAuthSystemPrompt:                                settings.ClaudeOAuthSystemPrompt,
		ClaudeOAuthSystemPromptBlocks:                          settings.ClaudeOAuthSystemPromptBlocks,
		EnableAnthropicCacheTTL1hInjection:                     settings.EnableAnthropicCacheTTL1hInjection,
		RewriteMessageCacheControl:                             settings.RewriteMessageCacheControl,
		EnableClientDatelineNormalization:                      settings.EnableClientDatelineNormalization,
		AntigravityUserAgentVersion:                            settings.AntigravityUserAgentVersion,
		OpenAICodexUserAgent:                                   settings.OpenAICodexUserAgent,
		MinCodexVersion:                                        settings.MinCodexVersion,
		MaxCodexVersion:                                        settings.MaxCodexVersion,
		CodexCLIOnlyBlacklist:                                  settings.CodexCLIOnlyBlacklist,
		CodexCLIOnlyWhitelist:                                  settings.CodexCLIOnlyWhitelist,
		CodexCLIOnlyAllowAppServerClients:                      settings.CodexCLIOnlyAllowAppServerClients,
		CodexCLIOnlyEngineFingerprintSignals:                   settings.CodexCLIOnlyEngineFingerprintSignals,
		WebSearchEmulationEnabled:                              settings.WebSearchEmulationEnabled,
		PaymentVisibleMethodAlipaySource:                       settings.PaymentVisibleMethodAlipaySource,
		PaymentVisibleMethodWxpaySource:                        settings.PaymentVisibleMethodWxpaySource,
		PaymentVisibleMethodAlipayEnabled:                      settings.PaymentVisibleMethodAlipayEnabled,
		PaymentVisibleMethodWxpayEnabled:                       settings.PaymentVisibleMethodWxpayEnabled,
		OpenAIAdvancedSchedulerEnabled:                         settings.OpenAIAdvancedSchedulerEnabled,
		OpenAIAdvancedSchedulerStickyWeightedEnabled:           settings.OpenAIAdvancedSchedulerStickyWeightedEnabled,
		OpenAIAdvancedSchedulerSubscriptionPriorityEnabled:     settings.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled,
		OpenAIAdvancedSchedulerLBTopK:                          settings.OpenAIAdvancedSchedulerLBTopK,
		OpenAIAdvancedSchedulerWeightPriority:                  settings.OpenAIAdvancedSchedulerWeightPriority,
		OpenAIAdvancedSchedulerWeightLoad:                      settings.OpenAIAdvancedSchedulerWeightLoad,
		OpenAIAdvancedSchedulerWeightQueue:                     settings.OpenAIAdvancedSchedulerWeightQueue,
		OpenAIAdvancedSchedulerWeightErrorRate:                 settings.OpenAIAdvancedSchedulerWeightErrorRate,
		OpenAIAdvancedSchedulerWeightTTFT:                      settings.OpenAIAdvancedSchedulerWeightTTFT,
		OpenAIAdvancedSchedulerWeightReset:                     settings.OpenAIAdvancedSchedulerWeightReset,
		OpenAIAdvancedSchedulerWeightQuotaHeadroom:             settings.OpenAIAdvancedSchedulerWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerWeightPreviousResponse:          settings.OpenAIAdvancedSchedulerWeightPreviousResponse,
		OpenAIAdvancedSchedulerWeightSessionSticky:             settings.OpenAIAdvancedSchedulerWeightSessionSticky,
		OpenAIAdvancedSchedulerEffectiveLBTopK:                 settings.OpenAIAdvancedSchedulerEffectiveLBTopK,
		OpenAIAdvancedSchedulerEffectiveWeightPriority:         settings.OpenAIAdvancedSchedulerEffectiveWeightPriority,
		OpenAIAdvancedSchedulerEffectiveWeightLoad:             settings.OpenAIAdvancedSchedulerEffectiveWeightLoad,
		OpenAIAdvancedSchedulerEffectiveWeightQueue:            settings.OpenAIAdvancedSchedulerEffectiveWeightQueue,
		OpenAIAdvancedSchedulerEffectiveWeightErrorRate:        settings.OpenAIAdvancedSchedulerEffectiveWeightErrorRate,
		OpenAIAdvancedSchedulerEffectiveWeightTTFT:             settings.OpenAIAdvancedSchedulerEffectiveWeightTTFT,
		OpenAIAdvancedSchedulerEffectiveWeightReset:            settings.OpenAIAdvancedSchedulerEffectiveWeightReset,
		OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom:    settings.OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse: settings.OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse,
		OpenAIAdvancedSchedulerEffectiveWeightSessionSticky:    settings.OpenAIAdvancedSchedulerEffectiveWeightSessionSticky,
		BalanceLowNotifyEnabled:                                settings.BalanceLowNotifyEnabled,
		BalanceLowNotifyThreshold:                              settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:                            settings.BalanceLowNotifyRechargeURL,
		SubscriptionExpiryNotifyEnabled:                        settings.SubscriptionExpiryNotifyEnabled,
		AccountQuotaNotifyEnabled:                              settings.AccountQuotaNotifyEnabled,
		AccountQuotaNotifyEmails:                               dto.NotifyEmailEntriesFromService(settings.AccountQuotaNotifyEmails),
		PaymentEnabled:                                         paymentCfg.Enabled,
		PaymentMinAmount:                                       paymentCfg.MinAmount,
		PaymentMaxAmount:                                       paymentCfg.MaxAmount,
		PaymentDailyLimit:                                      paymentCfg.DailyLimit,
		PaymentOrderTimeoutMin:                                 paymentCfg.OrderTimeoutMin,
		PaymentMaxPendingOrders:                                paymentCfg.MaxPendingOrders,
		PaymentEnabledTypes:                                    paymentCfg.EnabledTypes,
		PaymentBalanceDisabled:                                 paymentCfg.BalanceDisabled,
		PaymentBalanceRechargeMultiplier:                       paymentCfg.BalanceRechargeMultiplier,
		PaymentSubscriptionUSDToCNYRate:                        paymentCfg.SubscriptionUSDToCNYRate,
		PaymentRechargeFeeRate:                                 paymentCfg.RechargeFeeRate,
		PaymentLoadBalanceStrat:                                paymentCfg.LoadBalanceStrategy,
		PaymentProductNamePrefix:                               paymentCfg.ProductNamePrefix,
		PaymentProductNameSuffix:                               paymentCfg.ProductNameSuffix,
		PaymentHelpImageURL:                                    paymentCfg.HelpImageURL,
		PaymentHelpText:                                        paymentCfg.HelpText,
		PaymentCancelRateLimitEnabled:                          paymentCfg.CancelRateLimitEnabled,
		PaymentCancelRateLimitMax:                              paymentCfg.CancelRateLimitMax,
		PaymentCancelRateLimitWindow:                           paymentCfg.CancelRateLimitWindow,
		PaymentCancelRateLimitUnit:                             paymentCfg.CancelRateLimitUnit,
		PaymentCancelRateLimitMode:                             paymentCfg.CancelRateLimitMode,
		PaymentAlipayForceQRCode:                               paymentCfg.AlipayForceQRCode,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		AffiliateEnabled: settings.AffiliateEnabled,

		AllowUserViewErrorRequests: settings.AllowUserViewErrorRequests,
	}

	// OpenAI fast policy (stored under a dedicated setting key)
	if fastPolicy, err := h.settingService.GetOpenAIFastPolicySettings(c.Request.Context()); err != nil {
		slog.Error("openai_fast_policy_settings_get_failed", "error", err)
	} else if fastPolicy != nil {
		payload.OpenAIFastPolicySettings = openaiFastPolicySettingsToDTO(fastPolicy)
	}

	// Default platform quotas（JSON map）
	if platformQuotas, err := h.settingService.GetDefaultPlatformQuotas(c.Request.Context()); err != nil {
		slog.Error("default_platform_quotas_get_failed", "error", err)
	} else {
		payload.DefaultPlatformQuotas = platformQuotas
	}

	response.Success(c, systemSettingsResponseData(payload, authSourceDefaults))
}

// openaiFastPolicySettingsToDTO converts service -> dto for OpenAI fast policy.
func openaiFastPolicySettingsToDTO(s *service.OpenAIFastPolicySettings) *dto.OpenAIFastPolicySettings {
	if s == nil {
		return nil
	}
	rules := make([]dto.OpenAIFastPolicyRule, len(s.Rules))
	for i, r := range s.Rules {
		rules[i] = dto.OpenAIFastPolicyRule(r)
	}
	return &dto.OpenAIFastPolicySettings{Rules: rules}
}

// openaiFastPolicySettingsFromDTO converts dto -> service for OpenAI fast policy.
//
// 规范化 ServiceTier：在 DTO 进入 service 层之前统一把空字符串归一为
// service.OpenAIFastTierAny ("all")，避免管理员保存时空串与 "all" 同时
// 表达"匹配任意 tier"造成数据库取值的二义性。其它非空值原样透传，由
// service.SetOpenAIFastPolicySettings 负责合法值校验。
func openaiFastPolicySettingsFromDTO(s *dto.OpenAIFastPolicySettings) *service.OpenAIFastPolicySettings {
	if s == nil {
		return nil
	}
	rules := make([]service.OpenAIFastPolicyRule, len(s.Rules))
	for i, r := range s.Rules {
		rules[i] = service.OpenAIFastPolicyRule(r)
		tier := strings.ToLower(strings.TrimSpace(rules[i].ServiceTier))
		if tier == "" {
			tier = service.OpenAIFastTierAny
		}
		rules[i].ServiceTier = tier
	}
	return &service.OpenAIFastPolicySettings{Rules: rules}
}

func loginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}

func loginAgreementDocumentsToService(items []dto.LoginAgreementDocument) []service.LoginAgreementDocument {
	result := make([]service.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(item.ContentMD)
		if title == "" && content == "" {
			continue
		}
		result = append(result, service.LoginAgreementDocument{
			ID:        strings.TrimSpace(item.ID),
			Title:     title,
			ContentMD: content,
		})
	}
	return result
}

func systemSettingsResponseData(settings dto.SystemSettings, authSourceDefaults *service.AuthSourceDefaultSettings) map[string]any {
	data := make(map[string]any)
	raw, err := json.Marshal(settings)
	if err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	if authSourceDefaults == nil {
		authSourceDefaults = &service.AuthSourceDefaultSettings{}
	}

	data["auth_source_default_email_balance"] = authSourceDefaults.Email.Balance
	data["auth_source_default_email_concurrency"] = authSourceDefaults.Email.Concurrency
	data["auth_source_default_email_subscriptions"] = authSourceDefaults.Email.Subscriptions
	data["auth_source_default_email_grant_on_signup"] = authSourceDefaults.Email.GrantOnSignup
	data["auth_source_default_email_grant_on_first_bind"] = authSourceDefaults.Email.GrantOnFirstBind
	data["auth_source_default_linuxdo_balance"] = authSourceDefaults.LinuxDo.Balance
	data["auth_source_default_linuxdo_concurrency"] = authSourceDefaults.LinuxDo.Concurrency
	data["auth_source_default_linuxdo_subscriptions"] = authSourceDefaults.LinuxDo.Subscriptions
	data["auth_source_default_linuxdo_grant_on_signup"] = authSourceDefaults.LinuxDo.GrantOnSignup
	data["auth_source_default_linuxdo_grant_on_first_bind"] = authSourceDefaults.LinuxDo.GrantOnFirstBind
	data["auth_source_default_dingtalk_balance"] = authSourceDefaults.DingTalk.Balance
	data["auth_source_default_dingtalk_concurrency"] = authSourceDefaults.DingTalk.Concurrency
	data["auth_source_default_dingtalk_subscriptions"] = authSourceDefaults.DingTalk.Subscriptions
	data["auth_source_default_dingtalk_grant_on_signup"] = authSourceDefaults.DingTalk.GrantOnSignup
	data["auth_source_default_dingtalk_grant_on_first_bind"] = authSourceDefaults.DingTalk.GrantOnFirstBind
	data["auth_source_default_oidc_balance"] = authSourceDefaults.OIDC.Balance
	data["auth_source_default_oidc_concurrency"] = authSourceDefaults.OIDC.Concurrency
	data["auth_source_default_oidc_subscriptions"] = authSourceDefaults.OIDC.Subscriptions
	data["auth_source_default_oidc_grant_on_signup"] = authSourceDefaults.OIDC.GrantOnSignup
	data["auth_source_default_oidc_grant_on_first_bind"] = authSourceDefaults.OIDC.GrantOnFirstBind
	data["auth_source_default_wechat_balance"] = authSourceDefaults.WeChat.Balance
	data["auth_source_default_wechat_concurrency"] = authSourceDefaults.WeChat.Concurrency
	data["auth_source_default_wechat_subscriptions"] = authSourceDefaults.WeChat.Subscriptions
	data["auth_source_default_wechat_grant_on_signup"] = authSourceDefaults.WeChat.GrantOnSignup
	data["auth_source_default_wechat_grant_on_first_bind"] = authSourceDefaults.WeChat.GrantOnFirstBind
	data["auth_source_default_github_balance"] = authSourceDefaults.GitHub.Balance
	data["auth_source_default_github_concurrency"] = authSourceDefaults.GitHub.Concurrency
	data["auth_source_default_github_subscriptions"] = authSourceDefaults.GitHub.Subscriptions
	data["auth_source_default_github_grant_on_signup"] = authSourceDefaults.GitHub.GrantOnSignup
	data["auth_source_default_github_grant_on_first_bind"] = authSourceDefaults.GitHub.GrantOnFirstBind
	data["auth_source_default_google_balance"] = authSourceDefaults.Google.Balance
	data["auth_source_default_google_concurrency"] = authSourceDefaults.Google.Concurrency
	data["auth_source_default_google_subscriptions"] = authSourceDefaults.Google.Subscriptions
	data["auth_source_default_google_grant_on_signup"] = authSourceDefaults.Google.GrantOnSignup
	data["auth_source_default_google_grant_on_first_bind"] = authSourceDefaults.Google.GrantOnFirstBind
	data["auth_source_default_email_platform_quotas"] = authSourceDefaults.Email.PlatformQuotas
	data["auth_source_default_linuxdo_platform_quotas"] = authSourceDefaults.LinuxDo.PlatformQuotas
	data["auth_source_default_oidc_platform_quotas"] = authSourceDefaults.OIDC.PlatformQuotas
	data["auth_source_default_wechat_platform_quotas"] = authSourceDefaults.WeChat.PlatformQuotas
	data["auth_source_default_github_platform_quotas"] = authSourceDefaults.GitHub.PlatformQuotas
	data["auth_source_default_google_platform_quotas"] = authSourceDefaults.Google.PlatformQuotas
	data["auth_source_default_dingtalk_platform_quotas"] = authSourceDefaults.DingTalk.PlatformQuotas
	data["force_email_on_third_party_signup"] = authSourceDefaults.ForceEmailOnThirdPartySignup

	return data
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalDefaultSubscriptions(a, b []service.DefaultSubscriptionSetting) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].GroupID != b[i].GroupID || a[i].ValidityDays != b[i].ValidityDays {
			return false
		}
	}
	return true
}

func equalLoginAgreementDocuments(a, b []service.LoginAgreementDocument) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Title != b[i].Title || a[i].ContentMD != b[i].ContentMD {
			return false
		}
	}
	return true
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalNotifyEmailEntries(a, b []service.NotifyEmailEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Email != b[i].Email || a[i].Verified != b[i].Verified || a[i].Disabled != b[i].Disabled {
			return false
		}
	}
	return true
}

// TestSMTPRequest 测试SMTP连接请求
type TestSMTPRequest struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
}

// TestSMTPConnection 测试SMTP连接
// POST /api/v1/admin/settings/test-smtp
func (h *SettingHandler) TestSMTPConnection(c *gin.Context) {
	var req TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.SMTPHost = strings.TrimSpace(req.SMTPHost)
	req.SMTPUsername = strings.TrimSpace(req.SMTPUsername)

	var savedConfig *service.SMTPConfig
	if cfg, err := h.emailService.GetSMTPConfig(c.Request.Context()); err == nil && cfg != nil {
		savedConfig = cfg
	}

	if req.SMTPHost == "" && savedConfig != nil {
		req.SMTPHost = savedConfig.Host
	}
	if req.SMTPPort <= 0 {
		if savedConfig != nil && savedConfig.Port > 0 {
			req.SMTPPort = savedConfig.Port
		} else {
			req.SMTPPort = 587
		}
	}
	if req.SMTPUsername == "" && savedConfig != nil {
		req.SMTPUsername = savedConfig.Username
	}
	password := strings.TrimSpace(req.SMTPPassword)
	if password == "" && savedConfig != nil {
		password = savedConfig.Password
	}
	if req.SMTPHost == "" {
		response.BadRequest(c, "SMTP host is required")
		return
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		UseTLS:   req.SMTPUseTLS,
	}

	err := h.emailService.TestSMTPConnectionWithConfig(config)
	if err != nil {
		response.BadRequest(c, "SMTP connection test failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "SMTP connection successful"})
}

// SendTestEmailRequest 发送测试邮件请求
type SendTestEmailRequest struct {
	Email        string `json:"email" binding:"required,email"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from_email"`
	SMTPFromName string `json:"smtp_from_name"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
}

// SendTestEmail 发送测试邮件
// POST /api/v1/admin/settings/send-test-email
func (h *SettingHandler) SendTestEmail(c *gin.Context) {
	var req SendTestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	req.SMTPHost = strings.TrimSpace(req.SMTPHost)
	req.SMTPUsername = strings.TrimSpace(req.SMTPUsername)
	req.SMTPFrom = strings.TrimSpace(req.SMTPFrom)
	req.SMTPFromName = strings.TrimSpace(req.SMTPFromName)

	var savedConfig *service.SMTPConfig
	if cfg, err := h.emailService.GetSMTPConfig(c.Request.Context()); err == nil && cfg != nil {
		savedConfig = cfg
	}

	if req.SMTPHost == "" && savedConfig != nil {
		req.SMTPHost = savedConfig.Host
	}
	if req.SMTPPort <= 0 {
		if savedConfig != nil && savedConfig.Port > 0 {
			req.SMTPPort = savedConfig.Port
		} else {
			req.SMTPPort = 587
		}
	}
	if req.SMTPUsername == "" && savedConfig != nil {
		req.SMTPUsername = savedConfig.Username
	}
	password := strings.TrimSpace(req.SMTPPassword)
	if password == "" && savedConfig != nil {
		password = savedConfig.Password
	}
	if req.SMTPFrom == "" && savedConfig != nil {
		req.SMTPFrom = savedConfig.From
	}
	if req.SMTPFromName == "" && savedConfig != nil {
		req.SMTPFromName = savedConfig.FromName
	}
	if req.SMTPHost == "" {
		response.BadRequest(c, "SMTP host is required")
		return
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		From:     req.SMTPFrom,
		FromName: req.SMTPFromName,
		UseTLS:   req.SMTPUseTLS,
	}

	siteName := h.settingService.GetSiteName(c.Request.Context())
	subject := "[" + siteName + "] Test Email"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
        .content { padding: 40px 30px; text-align: center; }
        .success { color: #10b981; font-size: 48px; margin-bottom: 20px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + siteName + `</h1>
        </div>
        <div class="content">
            <div class="success">✓</div>
            <h2>Email Configuration Successful!</h2>
            <p>This is a test email to verify your SMTP settings are working correctly.</p>
        </div>
        <div class="footer">
            <p>This is an automated test message.</p>
        </div>
    </div>
</body>
</html>
`

	if err := h.emailService.SendEmailWithConfig(config, req.Email, subject, body); err != nil {
		response.BadRequest(c, "Failed to send test email: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Test email sent successfully"})
}

// GetAdminAPIKey 获取管理员 API Key 状态
// GET /api/v1/admin/settings/admin-api-key
func (h *SettingHandler) GetAdminAPIKey(c *gin.Context) {
	maskedKey, exists, err := h.settingService.GetAdminAPIKeyStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"exists":     exists,
		"masked_key": maskedKey,
	})
}

// RegenerateAdminAPIKey 生成/重新生成管理员 API Key
// POST /api/v1/admin/settings/admin-api-key/regenerate
func (h *SettingHandler) RegenerateAdminAPIKey(c *gin.Context) {
	key, err := h.settingService.GenerateAdminAPIKey(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"key": key, // 完整 key 只在生成时返回一次
	})
}

// DeleteAdminAPIKey 删除管理员 API Key
// DELETE /api/v1/admin/settings/admin-api-key
func (h *SettingHandler) DeleteAdminAPIKey(c *gin.Context) {
	if err := h.settingService.DeleteAdminAPIKey(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Admin API key deleted"})
}

// GetOverloadCooldownSettings 获取529过载冷却配置
// GET /api/v1/admin/settings/overload-cooldown
func (h *SettingHandler) GetOverloadCooldownSettings(c *gin.Context) {
	settings, err := h.settingService.GetOverloadCooldownSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.OverloadCooldownSettings{
		Enabled:         settings.Enabled,
		CooldownMinutes: settings.CooldownMinutes,
	})
}

// UpdateOverloadCooldownSettingsRequest 更新529过载冷却配置请求
type UpdateOverloadCooldownSettingsRequest struct {
	Enabled         bool `json:"enabled"`
	CooldownMinutes int  `json:"cooldown_minutes"`
}

// UpdateOverloadCooldownSettings 更新529过载冷却配置
// PUT /api/v1/admin/settings/overload-cooldown
func (h *SettingHandler) UpdateOverloadCooldownSettings(c *gin.Context) {
	var req UpdateOverloadCooldownSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.OverloadCooldownSettings{
		Enabled:         req.Enabled,
		CooldownMinutes: req.CooldownMinutes,
	}

	if err := h.settingService.SetOverloadCooldownSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updatedSettings, err := h.settingService.GetOverloadCooldownSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.OverloadCooldownSettings{
		Enabled:         updatedSettings.Enabled,
		CooldownMinutes: updatedSettings.CooldownMinutes,
	})
}

// GetRateLimit429CooldownSettings 获取429默认回避配置
// GET /api/v1/admin/settings/rate-limit-429-cooldown
func (h *SettingHandler) GetRateLimit429CooldownSettings(c *gin.Context) {
	settings, err := h.settingService.GetRateLimit429CooldownSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.RateLimit429CooldownSettings{
		Enabled:         settings.Enabled,
		CooldownSeconds: settings.CooldownSeconds,
	})
}

// UpdateRateLimit429CooldownSettingsRequest 更新429默认回避配置请求
type UpdateRateLimit429CooldownSettingsRequest struct {
	Enabled         bool `json:"enabled"`
	CooldownSeconds int  `json:"cooldown_seconds"`
}

// UpdateRateLimit429CooldownSettings 更新429默认回避配置
// PUT /api/v1/admin/settings/rate-limit-429-cooldown
func (h *SettingHandler) UpdateRateLimit429CooldownSettings(c *gin.Context) {
	var req UpdateRateLimit429CooldownSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.RateLimit429CooldownSettings{
		Enabled:         req.Enabled,
		CooldownSeconds: req.CooldownSeconds,
	}

	if err := h.settingService.SetRateLimit429CooldownSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updatedSettings, err := h.settingService.GetRateLimit429CooldownSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.RateLimit429CooldownSettings{
		Enabled:         updatedSettings.Enabled,
		CooldownSeconds: updatedSettings.CooldownSeconds,
	})
}

// GetStreamTimeoutSettings 获取流超时处理配置
// GET /api/v1/admin/settings/stream-timeout
func (h *SettingHandler) GetStreamTimeoutSettings(c *gin.Context) {
	settings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.StreamTimeoutSettings{
		Enabled:                settings.Enabled,
		Action:                 settings.Action,
		TempUnschedMinutes:     settings.TempUnschedMinutes,
		ThresholdCount:         settings.ThresholdCount,
		ThresholdWindowMinutes: settings.ThresholdWindowMinutes,
	})
}

// GetRectifierSettings 获取请求整流器配置
// GET /api/v1/admin/settings/rectifier
func (h *SettingHandler) GetRectifierSettings(c *gin.Context) {
	settings, err := h.settingService.GetRectifierSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	patterns := settings.APIKeySignaturePatterns
	if patterns == nil {
		patterns = []string{}
	}
	response.Success(c, dto.RectifierSettings{
		Enabled:                  settings.Enabled,
		ThinkingSignatureEnabled: settings.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    settings.ThinkingBudgetEnabled,
		APIKeySignatureEnabled:   settings.APIKeySignatureEnabled,
		APIKeySignaturePatterns:  patterns,
	})
}

// UpdateRectifierSettingsRequest 更新整流器配置请求
type UpdateRectifierSettingsRequest struct {
	Enabled                  bool     `json:"enabled"`
	ThinkingSignatureEnabled bool     `json:"thinking_signature_enabled"`
	ThinkingBudgetEnabled    bool     `json:"thinking_budget_enabled"`
	APIKeySignatureEnabled   bool     `json:"apikey_signature_enabled"`
	APIKeySignaturePatterns  []string `json:"apikey_signature_patterns"`
}

// UpdateRectifierSettings 更新请求整流器配置
// PUT /api/v1/admin/settings/rectifier
func (h *SettingHandler) UpdateRectifierSettings(c *gin.Context) {
	var req UpdateRectifierSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 校验并清理自定义匹配关键词
	const maxPatterns = 50
	const maxPatternLen = 500
	if len(req.APIKeySignaturePatterns) > maxPatterns {
		response.BadRequest(c, "Too many signature patterns (max 50)")
		return
	}
	var cleanedPatterns []string
	for _, p := range req.APIKeySignaturePatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) > maxPatternLen {
			response.BadRequest(c, "Signature pattern too long (max 500 characters)")
			return
		}
		cleanedPatterns = append(cleanedPatterns, p)
	}

	settings := &service.RectifierSettings{
		Enabled:                  req.Enabled,
		ThinkingSignatureEnabled: req.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    req.ThinkingBudgetEnabled,
		APIKeySignatureEnabled:   req.APIKeySignatureEnabled,
		APIKeySignaturePatterns:  cleanedPatterns,
	}

	if err := h.settingService.SetRectifierSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 重新获取设置返回
	updatedSettings, err := h.settingService.GetRectifierSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updatedPatterns := updatedSettings.APIKeySignaturePatterns
	if updatedPatterns == nil {
		updatedPatterns = []string{}
	}
	response.Success(c, dto.RectifierSettings{
		Enabled:                  updatedSettings.Enabled,
		ThinkingSignatureEnabled: updatedSettings.ThinkingSignatureEnabled,
		ThinkingBudgetEnabled:    updatedSettings.ThinkingBudgetEnabled,
		APIKeySignatureEnabled:   updatedSettings.APIKeySignatureEnabled,
		APIKeySignaturePatterns:  updatedPatterns,
	})
}

// GetBetaPolicySettings 获取 Beta 策略配置
// GET /api/v1/admin/settings/beta-policy
func (h *SettingHandler) GetBetaPolicySettings(c *gin.Context) {
	settings, err := h.settingService.GetBetaPolicySettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	rules := make([]dto.BetaPolicyRule, len(settings.Rules))
	for i, r := range settings.Rules {
		rules[i] = dto.BetaPolicyRule(r)
	}
	response.Success(c, dto.BetaPolicySettings{Rules: rules})
}

// UpdateBetaPolicySettingsRequest 更新 Beta 策略配置请求
type UpdateBetaPolicySettingsRequest struct {
	Rules []dto.BetaPolicyRule `json:"rules"`
}

// UpdateBetaPolicySettings 更新 Beta 策略配置
// PUT /api/v1/admin/settings/beta-policy
func (h *SettingHandler) UpdateBetaPolicySettings(c *gin.Context) {
	var req UpdateBetaPolicySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	rules := make([]service.BetaPolicyRule, len(req.Rules))
	for i, r := range req.Rules {
		rules[i] = service.BetaPolicyRule(r)
	}

	settings := &service.BetaPolicySettings{Rules: rules}
	if err := h.settingService.SetBetaPolicySettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Re-fetch to return updated settings
	updated, err := h.settingService.GetBetaPolicySettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	outRules := make([]dto.BetaPolicyRule, len(updated.Rules))
	for i, r := range updated.Rules {
		outRules[i] = dto.BetaPolicyRule(r)
	}
	response.Success(c, dto.BetaPolicySettings{Rules: outRules})
}

// UpdateStreamTimeoutSettingsRequest 更新流超时配置请求
type UpdateStreamTimeoutSettingsRequest struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}

// UpdateStreamTimeoutSettings 更新流超时处理配置
// PUT /api/v1/admin/settings/stream-timeout
func (h *SettingHandler) UpdateStreamTimeoutSettings(c *gin.Context) {
	var req UpdateStreamTimeoutSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.StreamTimeoutSettings{
		Enabled:                req.Enabled,
		Action:                 req.Action,
		TempUnschedMinutes:     req.TempUnschedMinutes,
		ThresholdCount:         req.ThresholdCount,
		ThresholdWindowMinutes: req.ThresholdWindowMinutes,
	}

	if err := h.settingService.SetStreamTimeoutSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 重新获取设置返回
	updatedSettings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.StreamTimeoutSettings{
		Enabled:                updatedSettings.Enabled,
		Action:                 updatedSettings.Action,
		TempUnschedMinutes:     updatedSettings.TempUnschedMinutes,
		ThresholdCount:         updatedSettings.ThresholdCount,
		ThresholdWindowMinutes: updatedSettings.ThresholdWindowMinutes,
	})
}

// GetPayloadLoggingSettings 获取报文审计记录配置
// GET /api/v1/admin/settings/payload-logging
func (h *SettingHandler) GetPayloadLoggingSettings(c *gin.Context) {
	settings, err := h.settingService.GetPayloadLoggingSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.PayloadLoggingSettings{
		Enabled:         settings.Enabled,
		MaxRequestSize:  settings.MaxRequestSize,
		MaxResponseSize: settings.MaxResponseSize,
		RetentionDays:   settings.RetentionDays,
	})
}

// UpdatePayloadLoggingSettingsRequest 更新报文审计记录配置请求
type UpdatePayloadLoggingSettingsRequest struct {
	Enabled         bool  `json:"enabled"`
	MaxRequestSize  int64 `json:"max_request_size"`
	MaxResponseSize int64 `json:"max_response_size"`
	RetentionDays   int   `json:"retention_days"`
}

// UpdatePayloadLoggingSettings 更新报文审计记录配置
// PUT /api/v1/admin/settings/payload-logging
func (h *SettingHandler) UpdatePayloadLoggingSettings(c *gin.Context) {
	var req UpdatePayloadLoggingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.PayloadLoggingSettings{
		Enabled:         req.Enabled,
		MaxRequestSize:  req.MaxRequestSize,
		MaxResponseSize: req.MaxResponseSize,
		RetentionDays:   req.RetentionDays,
	}

	if err := h.settingService.SetPayloadLoggingSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, err := h.settingService.GetPayloadLoggingSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PayloadLoggingSettings{
		Enabled:         updated.Enabled,
		MaxRequestSize:  updated.MaxRequestSize,
		MaxResponseSize: updated.MaxResponseSize,
		RetentionDays:   updated.RetentionDays,
	})
}

// GetWebSearchEmulationConfig 获取 Web Search 模拟配置
// GET /api/v1/admin/settings/web-search-emulation
func (h *SettingHandler) GetWebSearchEmulationConfig(c *gin.Context) {
	cfg, err := h.settingService.GetWebSearchEmulationConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, service.PopulateWebSearchUsage(c.Request.Context(), cfg))
}

// UpdateWebSearchEmulationConfig 更新 Web Search 模拟配置
// PUT /api/v1/admin/settings/web-search-emulation
func (h *SettingHandler) UpdateWebSearchEmulationConfig(c *gin.Context) {
	var cfg service.WebSearchEmulationConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.settingService.SaveWebSearchEmulationConfig(c.Request.Context(), &cfg); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Re-read (with sanitized api keys) to return current state
	updated, err := h.settingService.GetWebSearchEmulationConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, service.PopulateWebSearchUsage(c.Request.Context(), updated))
}

// ResetWebSearchUsage 重置指定 provider 的配额用量
// POST /api/v1/admin/settings/web-search-emulation/reset-usage
func (h *SettingHandler) ResetWebSearchUsage(c *gin.Context) {
	var req struct {
		ProviderType string `json:"provider_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.ProviderType == "" {
		response.BadRequest(c, "provider_type is required")
		return
	}
	if err := service.ResetWebSearchUsage(c.Request.Context(), req.ProviderType); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

// TestWebSearchEmulation 测试 Web Search 搜索
// POST /api/v1/admin/settings/web-search-emulation/test
func (h *SettingHandler) TestWebSearchEmulation(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		req.Query = "搜索今年世界大事件"
	}

	result, err := service.TestWebSearch(c.Request.Context(), req.Query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ensureDingTalkSyncAttributes 在保存 settings 后，按 admin 配置的 (attr key, attr name)
// 兜底 upsert 对应 user attribute definition：不存在则创建；存在但 name 不同则更新 name
// （type/options/required 不变）。仅 internal_only + 对应 sync 开关开启时执行。
// 失败仅记录日志，不阻塞 settings 保存。
func (h *SettingHandler) ensureDingTalkSyncAttributes(ctx context.Context, settings *service.SystemSettings) {
	if h.userAttributeService == nil || settings == nil {
		return
	}
	if settings.DingTalkConnectCorpRestrictionPolicy != "internal_only" {
		return
	}
	if settings.DingTalkConnectSyncDisplayName {
		h.ensureUserAttributeDefinition(ctx, settings.DingTalkConnectSyncDisplayNameAttrKey, settings.DingTalkConnectSyncDisplayNameAttrName, "钉钉 internal_only 登录时同步的钉钉姓名", service.AttributeTypeText)
	}
	if settings.DingTalkConnectSyncCorpEmail {
		h.ensureUserAttributeDefinition(ctx, settings.DingTalkConnectSyncCorpEmailAttrKey, settings.DingTalkConnectSyncCorpEmailAttrName, "钉钉 internal_only 登录时同步的企业邮箱", service.AttributeTypeEmail)
	}
	if settings.DingTalkConnectSyncDept {
		h.ensureUserAttributeDefinition(ctx, settings.DingTalkConnectSyncDeptAttrKey, settings.DingTalkConnectSyncDeptAttrName, "钉钉 internal_only 登录时同步的完整部门路径（如：公司/研发部）", service.AttributeTypeText)
	}
}

func (h *SettingHandler) ensureUserAttributeDefinition(ctx context.Context, key, name, description string, attrType service.UserAttributeType) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	existing, err := h.userAttributeService.GetDefinitionByKey(ctx, key)
	if err == nil && existing != nil {
		if strings.TrimSpace(name) != "" && existing.Name != name {
			if _, err := h.userAttributeService.UpdateDefinition(ctx, existing.ID, service.UpdateAttributeDefinitionInput{
				Name: &name,
			}); err != nil {
				slog.Warn("dingtalk: update user attribute definition name failed", "key", key, "err", err.Error())
				return
			}
			slog.Info("dingtalk: updated user attribute definition name", "key", key, "name", name)
		}
		return
	}
	if _, err := h.userAttributeService.CreateDefinition(ctx, service.CreateAttributeDefinitionInput{
		Key:         key,
		Name:        name,
		Description: description,
		Type:        attrType,
		Enabled:     true,
	}); err != nil {
		slog.Warn("dingtalk: ensure user attribute definition failed", "key", key, "err", err.Error())
		return
	}
	slog.Info("dingtalk: created user attribute definition", "key", key, "name", name, "type", attrType)
}

// ListEmailTemplates returns all editable notification email templates.
// GET /api/v1/admin/settings/email-templates
func (h *SettingHandler) ListEmailTemplates(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	events := h.notificationEmailService.ListEventInfos()
	templates, err := h.notificationEmailService.ListTemplates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.EmailTemplateListResponse{
		Events:       emailTemplateEventOptionsToDTO(events),
		Locales:      h.notificationEmailService.SupportedLocales(),
		Templates:    emailTemplateSummariesToDTO(templates),
		Placeholders: emailTemplatePlaceholderUnion(events),
	})
}

// GetEmailTemplate returns one editable notification email template.
// GET /api/v1/admin/settings/email-templates/:event/:locale
func (h *SettingHandler) GetEmailTemplate(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	tmpl, err := h.notificationEmailService.GetTemplate(c.Request.Context(), c.Param("event"), c.Param("locale"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, emailTemplateDetailToDTO(tmpl))
}

// UpdateEmailTemplate saves an override for one event/locale template.
// PUT /api/v1/admin/settings/email-templates/:event/:locale
func (h *SettingHandler) UpdateEmailTemplate(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	var req dto.UpdateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tmpl, err := h.notificationEmailService.UpdateTemplate(c.Request.Context(), c.Param("event"), c.Param("locale"), req.Subject, req.HTML)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, emailTemplateDetailToDTO(tmpl))
}

// RestoreOfficialEmailTemplate removes an override and returns the built-in template.
// POST /api/v1/admin/settings/email-templates/:event/:locale/restore-official
func (h *SettingHandler) RestoreOfficialEmailTemplate(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	tmpl, err := h.notificationEmailService.RestoreOfficialTemplate(c.Request.Context(), c.Param("event"), c.Param("locale"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, emailTemplateDetailToDTO(tmpl))
}

// PreviewEmailTemplate renders a template with safe sample variables without saving it.
// POST /api/v1/admin/settings/email-templates/preview
func (h *SettingHandler) PreviewEmailTemplate(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	var req dto.PreviewEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	preview, err := h.notificationEmailService.PreviewTemplate(c.Request.Context(), service.NotificationEmailPreviewInput{
		Event:     req.Event,
		Locale:    req.Locale,
		Subject:   req.Subject,
		HTML:      req.HTML,
		Variables: req.Variables,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, dto.EmailTemplatePreviewResponse{Subject: preview.Subject, HTML: preview.HTML})
}

func emailTemplateEventOptionsToDTO(events []service.NotificationEmailEventInfo) []dto.EmailTemplateEventOption {
	items := make([]dto.EmailTemplateEventOption, 0, len(events))
	for _, event := range events {
		items = append(items, dto.EmailTemplateEventOption{
			Value:       event.Event,
			Label:       event.Label,
			Description: event.Description,
			Category:    event.Category,
			Optional:    event.Optional,
		})
	}
	return items
}

func emailTemplateSummariesToDTO(templates []service.NotificationEmailTemplate) []dto.EmailTemplateSummary {
	items := make([]dto.EmailTemplateSummary, 0, len(templates))
	for _, tmpl := range templates {
		items = append(items, dto.EmailTemplateSummary{
			Event:     tmpl.Event,
			Locale:    tmpl.Locale,
			Subject:   tmpl.Subject,
			IsCustom:  tmpl.IsCustom,
			UpdatedAt: emailTemplateUpdatedAt(tmpl),
		})
	}
	return items
}

func emailTemplateDetailToDTO(tmpl service.NotificationEmailTemplate) dto.EmailTemplateDetail {
	return dto.EmailTemplateDetail{
		Event:        tmpl.Event,
		Locale:       tmpl.Locale,
		Subject:      tmpl.Subject,
		HTML:         tmpl.HTML,
		IsCustom:     tmpl.IsCustom,
		UpdatedAt:    emailTemplateUpdatedAt(tmpl),
		Placeholders: tmpl.Placeholders,
	}
}

func emailTemplateUpdatedAt(tmpl service.NotificationEmailTemplate) string {
	if tmpl.UpdatedAt == nil {
		return ""
	}
	return tmpl.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
}

func emailTemplatePlaceholderUnion(events []service.NotificationEmailEventInfo) []string {
	seen := make(map[string]struct{})
	placeholders := make([]string, 0)
	for _, event := range events {
		for _, placeholder := range event.Placeholders {
			if _, ok := seen[placeholder]; ok {
				continue
			}
			seen[placeholder] = struct{}{}
			placeholders = append(placeholders, placeholder)
		}
	}
	return placeholders
}

// equalNullableFloat compares two *float64 values treating nil as a distinct case.
func equalNullableFloat(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// slotOf returns the *float64 for the given window from a DefaultPlatformQuotaSetting.
func slotOf(s *service.DefaultPlatformQuotaSetting, win string) *float64 {
	if s == nil {
		return nil
	}
	switch win {
	case "daily":
		return s.DailyLimitUSD
	case "weekly":
		return s.WeeklyLimitUSD
	case "monthly":
		return s.MonthlyLimitUSD
	}
	return nil
}

// equalPlatformQuotaSettings reports whether two platform-quota maps are identical across all allowed slots.
func equalPlatformQuotaSettings(before, after map[string]*service.DefaultPlatformQuotaSetting) bool {
	for _, platform := range service.AllowedQuotaPlatforms {
		b := before[platform]
		a := after[platform]
		if !equalNullableFloat(slotOf(b, "daily"), slotOf(a, "daily")) {
			return false
		}
		if !equalNullableFloat(slotOf(b, "weekly"), slotOf(a, "weekly")) {
			return false
		}
		if !equalNullableFloat(slotOf(b, "monthly"), slotOf(a, "monthly")) {
			return false
		}
	}
	return true
}
