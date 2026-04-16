package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
	}
}

// AdminUpdateAPIKeyGroupRequest represents the request to update an API key's group
type AdminUpdateAPIKeyGroupRequest struct {
	GroupID      *int64                        `json:"group_id"` // nil=不修改, 0=解绑, >0=绑定到目标分组
	ClearGroupID *bool                         `json:"clear_group_id"`
	BoundGroups  *[]service.APIKeyGroupBinding `json:"bound_groups,omitempty"`
}

// UpdateGroup handles updating an API key's group binding
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	useLegacyGroupOnlyPath := req.BoundGroups == nil && (req.ClearGroupID == nil || !*req.ClearGroupID)

	var result *service.AdminUpdateAPIKeyResult
	if useLegacyGroupOnlyPath {
		result, err = h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID)
	} else {
		input := &service.AdminUpdateAPIKeyInput{
			GroupID:     req.GroupID,
			BoundGroups: req.BoundGroups,
		}
		if req.ClearGroupID != nil && *req.ClearGroupID {
			input.ClearGroupID = true
		}
		result, err = h.adminService.AdminUpdateAPIKey(c.Request.Context(), keyID, input)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	resp := struct {
		APIKey                 *dto.APIKey `json:"api_key"`
		AutoGrantedGroupAccess bool        `json:"auto_granted_group_access"`
		GrantedGroupID         *int64      `json:"granted_group_id,omitempty"`
		GrantedGroupName       string      `json:"granted_group_name,omitempty"`
		GrantedGroupIDs        []int64     `json:"granted_group_ids,omitempty"`
		GrantedGroupNames      []string    `json:"granted_group_names,omitempty"`
	}{
		APIKey:                 dto.APIKeyFromService(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
		GrantedGroupIDs:        result.GrantedGroupIDs,
		GrantedGroupNames:      result.GrantedGroupNames,
	}
	response.Success(c, resp)
}
