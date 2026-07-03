package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func validateAPIKeyGroupAvailable(apiKey *service.APIKey) (code string, message string, ok bool) {
	if apiKey == nil || apiKey.HasBoundGroups() {
		return "", "", true
	}
	if apiKey.GroupID == nil {
		return "", "", true
	}
	if apiKey.Group == nil || apiKey.Group.Status == "deleted" {
		return "GROUP_DELETED", "API Key 所属分组已删除", false
	}
	if !apiKey.Group.IsActive() {
		return "GROUP_DISABLED", "API Key 所属分组已停用", false
	}
	return "", "", true
}

func abortIfAPIKeyGroupUnavailable(c *gin.Context, apiKey *service.APIKey) bool {
	code, message, ok := validateAPIKeyGroupAvailable(apiKey)
	if ok {
		return false
	}
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	AbortWithError(c, 403, code, message)
	return true
}

func abortIfAPIKeyGroupNotAllowed(c *gin.Context, apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.User == nil || apiKey.Group == nil || apiKey.HasBoundGroups() {
		return false
	}
	if apiKey.User.CanBindGroup(apiKey.Group.ID, apiKey.Group.IsExclusive) {
		return false
	}
	AbortWithError(c, 403, "GROUP_NOT_ALLOWED", "User is not allowed to use this group")
	return true
}
