package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// MultiGroupPreResolveGemini is a middleware that pre-resolves the target group
// for multi-group API keys on Gemini native endpoints (/v1beta).
//
// Unlike MultiGroupPreResolve (which reads the model from the JSON body),
// Gemini endpoints encode the model name in URL path parameters:
//   - GET  /v1beta/models/:model
//   - POST /v1beta/models/*modelAction  (e.g. "gemini-pro:generateContent")
//
// For list endpoints (GET /v1beta/models) with no model parameter, it falls
// back to the highest-priority bound group via ResolveGroupForModel("").
func MultiGroupPreResolveGemini(subscriptionService *service.SubscriptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsMultiGroupDeferred(c) {
			c.Next()
			return
		}

		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok {
			c.Next()
			return
		}

		// Extract model name from URL path parameters
		modelName := extractGeminiModelFromParams(c)

		resolvedGroup, err := service.ResolveGroupForModel(apiKey, modelName)
		if err != nil || resolvedGroup == nil {
			// Let the handler deal with the error after full request parsing
			c.Next()
			return
		}

		// Update the apiKey's group reference
		apiKey.Group = resolvedGroup
		apiKey.GroupID = &resolvedGroup.ID

		// Set the group context for downstream middleware/handlers
		SetGroupContext(c, resolvedGroup)

		// Load subscription if needed (deferred from auth middleware for multi-group keys)
		// 订阅型分组必须强制校验订阅，失败则中止请求（不允许回退到余额模式）
		if resolvedGroup.IsSubscriptionType() && subscriptionService != nil && apiKey.User != nil {
			sub, subErr := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				resolvedGroup.ID,
			)
			if subErr != nil || sub == nil {
				GoogleErrorWriter(c, 403, "No active subscription found for this group")
				c.Abort()
				return
			}
			// 校验订阅限额
			needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(sub, resolvedGroup)
			if validateErr != nil {
				status := 403
				if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
					errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
					errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
					status = 429
				}
				GoogleErrorWriter(c, status, validateErr.Error())
				c.Abort()
				return
			}
			c.Set(string(ContextKeySubscription), sub)
			// 窗口维护异步化
			if needsMaintenance {
				maintenanceCopy := *sub
				subscriptionService.DoWindowMaintenance(&maintenanceCopy)
			}
		} else if apiKey.User != nil {
			// 非订阅型分组：检查余额
			if apiKey.User.Balance <= 0 {
				GoogleErrorWriter(c, 403, "Insufficient account balance")
				c.Abort()
				return
			}
		}

		// Clear the deferred flag so downstream handlers skip re-resolution
		ctx := context.WithValue(c.Request.Context(), ctxkey.MultiGroupDeferred, false)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// extractGeminiModelFromParams extracts the model name from Gin route parameters.
// Handles both :model param (GET /v1beta/models/:model) and
// *modelAction param (POST /v1beta/models/*modelAction like "/gemini-pro:generateContent").
func extractGeminiModelFromParams(c *gin.Context) string {
	// Try :model parameter first (GET /v1beta/models/:model)
	if model := strings.TrimSpace(c.Param("model")); model != "" {
		return model
	}

	// Try *modelAction parameter (POST /v1beta/models/*modelAction)
	modelAction := strings.TrimPrefix(c.Param("modelAction"), "/")
	if modelAction == "" {
		return ""
	}

	// Extract model name: "gemini-pro:generateContent" → "gemini-pro"
	if idx := strings.Index(modelAction, ":"); idx > 0 {
		return modelAction[:idx]
	}
	return modelAction
}
