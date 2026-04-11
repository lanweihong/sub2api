package middleware

import (
	"context"
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
