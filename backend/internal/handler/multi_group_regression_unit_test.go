//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandlerModels_MultiGroupFiltersInactiveAndForcePlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	activeSoraGroup := &service.Group{ID: 101, Platform: service.PlatformSora, Status: service.StatusActive}
	activeOpenAIGroup := &service.Group{ID: 102, Platform: service.PlatformOpenAI, Status: service.StatusActive}
	inactiveSoraGroup := &service.Group{ID: 103, Platform: service.PlatformSora, Status: service.StatusDisabled}

	accountRepo := &stubAccountRepo{
		accounts: map[int64]*service.Account{
			1: {
				ID:          1,
				Platform:    service.PlatformSora,
				Status:      service.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"sora-fast": "sora-fast"},
				},
				AccountGroups: []service.AccountGroup{{AccountID: 1, GroupID: activeSoraGroup.ID}},
			},
			2: {
				ID:          2,
				Platform:    service.PlatformOpenAI,
				Status:      service.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"gpt-4o": "gpt-4o"},
				},
				AccountGroups: []service.AccountGroup{{AccountID: 2, GroupID: activeOpenAIGroup.ID}},
			},
			3: {
				ID:          3,
				Platform:    service.PlatformSora,
				Status:      service.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"sora-disabled": "sora-disabled"},
				},
				AccountGroups: []service.AccountGroup{{AccountID: 3, GroupID: inactiveSoraGroup.ID}},
			},
		},
	}

	gatewayService := service.NewGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	handler := &GatewayHandler{gatewayService: gatewayService}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/sora/v1/models", nil)
	c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformSora)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		BoundGroups: []service.APIKeyGroup{
			{GroupID: activeSoraGroup.ID, Group: activeSoraGroup},
			{GroupID: activeOpenAIGroup.ID, Group: activeOpenAIGroup},
			{GroupID: inactiveSoraGroup.ID, Group: inactiveSoraGroup},
		},
	})

	handler.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "sora-fast", resp.Data[0].ID)
}

func TestGatewayHandlerCountTokens_MultiGroupNoMatchReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"sora-preview","messages":[{"role":"user","content":"hello"}]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.MultiGroupDeferred, true))

	apiKey := &service.APIKey{
		ID:     1,
		Status: service.StatusActive,
		BoundGroups: []service.APIKeyGroup{
			{
				GroupID: 1,
				Group:   &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive},
			},
			{
				GroupID: 2,
				Group:   &service.Group{ID: 2, Platform: service.PlatformAnthropic, Status: service.StatusActive},
			},
		},
	}
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})

	handler := &GatewayHandler{}
	handler.CountTokens(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "No group matches the requested model: sora-preview")
}

func TestSoraGatewayHandlerChatCompletions_MultiGroupNoMatchReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/sora/v1/chat/completions", strings.NewReader(`{"model":"sora-preview","messages":[{"role":"user","content":"hello"}],"stream":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.MultiGroupDeferred, true))

	apiKey := &service.APIKey{
		ID:     1,
		Status: service.StatusActive,
		User:   &service.User{ID: 1, Concurrency: 1, Status: service.StatusActive},
		BoundGroups: []service.APIKeyGroup{
			{
				GroupID: 1,
				Group:   &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive},
			},
			{
				GroupID: 2,
				Group:   &service.Group{ID: 2, Platform: service.PlatformAnthropic, Status: service.StatusActive},
			},
		},
	}
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})

	handler := &SoraGatewayHandler{streamMode: "force"}
	handler.ChatCompletions(c)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "No group matches the requested model: sora-preview")
}
