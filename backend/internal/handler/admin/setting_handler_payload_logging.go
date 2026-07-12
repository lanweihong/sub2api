package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

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
