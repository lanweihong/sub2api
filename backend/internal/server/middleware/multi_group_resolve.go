package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// MultiGroupPreResolve is a middleware that pre-resolves the target group for
// multi-group API keys before route dispatch. It peeks at the request body to
// extract the "model" field, resolves the matching group, and sets the group
// context so that downstream route closures (e.g. getGroupPlatform) see the
// correct platform. The request body is restored for subsequent handlers.
//
// If the resolved group is subscription-type, the subscription is loaded and
// set in context for downstream billing checks.
//
// For single-group keys or keys without bound groups, this is a no-op.
func MultiGroupPreResolve(subscriptionService *service.SubscriptionService) gin.HandlerFunc {
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

		// Peek at the body to extract the model field
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// On error, restore empty body and let the handler handle it
			c.Request.Body = http.NoBody
			c.Next()
			return
		}
		// Restore the body so downstream handlers can read it
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		model := gjson.GetBytes(bodyBytes, "model").String()
		if model == "" {
			c.Next()
			return
		}

		resolvedGroup, err := service.ResolveGroupForModel(apiKey, model)
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
				AbortWithError(c, 403, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group")
				return
			}
			// 校验订阅限额
			needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(sub, resolvedGroup)
			if validateErr != nil {
				code := "SUBSCRIPTION_INVALID"
				status := 403
				if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
					errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
					errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
					code = "USAGE_LIMIT_EXCEEDED"
					status = 429
				}
				AbortWithError(c, status, code, validateErr.Error())
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
				AbortWithError(c, 403, "INSUFFICIENT_BALANCE", "Insufficient account balance")
				return
			}
		}

		// Clear the deferred flag so downstream handlers skip re-resolution
		ctx := context.WithValue(c.Request.Context(), ctxkey.MultiGroupDeferred, false)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
