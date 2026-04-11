package middleware

import (
	"bytes"
	"context"
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
		if resolvedGroup.IsSubscriptionType() && subscriptionService != nil && apiKey.User != nil {
			sub, subErr := subscriptionService.GetActiveSubscription(
				c.Request.Context(),
				apiKey.User.ID,
				resolvedGroup.ID,
			)
			if subErr == nil && sub != nil {
				c.Set(string(ContextKeySubscription), sub)
			}
		}

		// Clear the deferred flag so downstream handlers skip re-resolution
		ctx := context.WithValue(c.Request.Context(), ctxkey.MultiGroupDeferred, false)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
