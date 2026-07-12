package service

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// resolveDefaultDepartmentID 安全地获取默认部门 ID。
// 若 deptService 未注入或默认部门不存在，返回错误。
func (s *adminServiceImpl) resolveDefaultDepartmentID(ctx context.Context) (int64, error) {
	if s.deptService == nil {
		return 0, ErrServiceUnavailable
	}
	id, err := s.deptService.GetDefaultDepartmentID(ctx)
	if err != nil {
		return 0, err
	}
	if id <= 0 {
		return 0, ErrServiceUnavailable
	}
	return id, nil
}

func (s *adminServiceImpl) GetUserAvailableGroups(ctx context.Context, userID int64) ([]Group, error) {
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}

	allGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	subscribedGroupIDs := make(map[int64]struct{})
	if s.userSubRepo != nil {
		activeSubscriptions, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, sub := range activeSubscriptions {
			subscribedGroupIDs[sub.GroupID] = struct{}{}
		}
	}

	availableGroups := make([]Group, 0, len(allGroups))
	for _, group := range allGroups {
		if group.IsSubscriptionType() {
			if _, ok := subscribedGroupIDs[group.ID]; ok {
				availableGroups = append(availableGroups, group)
			}
			continue
		}
		// 管理员允许为用户配置任意标准分组；专属标准分组在保存时自动补 allowed_groups。
		availableGroups = append(availableGroups, group)
	}

	return availableGroups, nil
}

func (s *adminServiceImpl) AdminUpdateAPIKey(ctx context.Context, keyID int64, input *AdminUpdateAPIKeyInput) (*AdminUpdateAPIKeyResult, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return &AdminUpdateAPIKeyResult{APIKey: apiKey}, nil
	}

	normalized := *input
	if normalized.GroupID != nil {
		switch {
		case *normalized.GroupID < 0:
			return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be non-negative")
		case *normalized.GroupID == 0:
			normalized.GroupID = nil
			normalized.ClearGroupID = true
		}
	}
	if normalized.BoundGroups != nil {
		for _, binding := range *normalized.BoundGroups {
			if binding.GroupID <= 0 {
				return nil, infraerrors.BadRequest("INVALID_GROUP_ID", "bound_groups.group_id must be positive")
			}
		}
	}
	if normalized.GroupID == nil && !normalized.ClearGroupID && normalized.BoundGroups == nil {
		return &AdminUpdateAPIKeyResult{APIKey: apiKey}, nil
	}

	targetGroupIDs := make([]int64, 0, 1)
	seen := make(map[int64]struct{})
	if normalized.GroupID != nil {
		targetGroupIDs = append(targetGroupIDs, *normalized.GroupID)
		seen[*normalized.GroupID] = struct{}{}
	}
	if normalized.BoundGroups != nil {
		for _, binding := range *normalized.BoundGroups {
			if _, ok := seen[binding.GroupID]; ok {
				continue
			}
			seen[binding.GroupID] = struct{}{}
			targetGroupIDs = append(targetGroupIDs, binding.GroupID)
		}
	}

	autoGrantGroups := make([]*Group, 0, len(targetGroupIDs))
	for _, groupID := range targetGroupIDs {
		group, err := s.groupRepo.GetByID(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if group.Status != StatusActive {
			return nil, infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
		}
		if group.IsSubscriptionType() && s.userSubRepo == nil {
			return nil, infraerrors.InternalServer("SUBSCRIPTION_REPOSITORY_UNAVAILABLE", "subscription repository is not configured")
		}
		if group.IsExclusive && !group.IsSubscriptionType() {
			autoGrantGroups = append(autoGrantGroups, group)
		}
	}

	opCtx := ctx
	var tx *dbent.Tx
	if len(autoGrantGroups) > 0 {
		if s.entClient == nil {
			logger.LegacyPrintf("service.admin", "Warning: entClient is nil, skipping transaction protection for admin api key update")
		} else {
			tx, err = s.entClient.Tx(ctx)
			if err != nil {
				return nil, fmt.Errorf("begin transaction: %w", err)
			}
			defer func() { _ = tx.Rollback() }()
			opCtx = dbent.NewTxContext(ctx, tx)
		}
	}

	grantedGroupIDs := make([]int64, 0, len(autoGrantGroups))
	grantedGroupNames := make([]string, 0, len(autoGrantGroups))
	for _, group := range autoGrantGroups {
		if err := s.userRepo.AddGroupToAllowedGroups(opCtx, apiKey.UserID, group.ID); err != nil {
			return nil, fmt.Errorf("add group to user allowed groups: %w", err)
		}
		grantedGroupIDs = append(grantedGroupIDs, group.ID)
		grantedGroupNames = append(grantedGroupNames, group.Name)
	}

	apiKeyService := &APIKeyService{
		apiKeyRepo:        s.apiKeyRepo,
		userRepo:          s.userRepo,
		groupRepo:         s.groupRepo,
		userSubRepo:       s.userSubRepo,
		userGroupRateRepo: s.userGroupRateRepo,
	}
	updateReq := UpdateAPIKeyRequest{
		GroupID:      normalized.GroupID,
		ClearGroupID: normalized.ClearGroupID,
		BoundGroups:  normalized.BoundGroups,
		IPWhitelist:  apiKey.IPWhitelist,
		IPBlacklist:  apiKey.IPBlacklist,
	}
	updatedKey, err := apiKeyService.Update(opCtx, keyID, apiKey.UserID, updateReq)
	if err != nil {
		return nil, err
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
	}

	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, updatedKey.Key)
	}

	result := &AdminUpdateAPIKeyResult{
		APIKey:            updatedKey,
		GrantedGroupIDs:   grantedGroupIDs,
		GrantedGroupNames: grantedGroupNames,
	}
	if len(grantedGroupIDs) > 0 {
		result.AutoGrantedGroupAccess = true
		result.GrantedGroupID = &grantedGroupIDs[0]
		if len(grantedGroupNames) == 1 {
			result.GrantedGroupName = grantedGroupNames[0]
		} else {
			result.GrantedGroupName = strings.Join(grantedGroupNames, ", ")
		}
	}

	return result, nil
}
