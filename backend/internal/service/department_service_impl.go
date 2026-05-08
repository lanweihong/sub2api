package service

import (
	"context"
	"strings"
	"sync"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type departmentServiceImpl struct {
	deptRepo  DepartmentRepository
	entClient *dbent.Client

	// 默认部门 ID 缓存（进程内，按需加载）
	defaultOnce sync.Once
	defaultID   int64
}

func NewDepartmentService(deptRepo DepartmentRepository, entClient *dbent.Client) DepartmentService {
	return &departmentServiceImpl{deptRepo: deptRepo, entClient: entClient}
}

func (s *departmentServiceImpl) List(ctx context.Context) ([]Department, error) {
	return s.deptRepo.List(ctx)
}

func (s *departmentServiceImpl) GetByID(ctx context.Context, id int64) (*Department, error) {
	return s.deptRepo.GetByID(ctx, id)
}

func (s *departmentServiceImpl) Create(ctx context.Context, input *CreateDepartmentInput) (*Department, error) {
	status := input.Status
	if status == "" {
		status = "active"
	}

	dept := &Department{
		Name:        input.Name,
		Code:        strings.TrimSpace(input.Code),
		Description: input.Description,
		ParentID:    input.ParentID,
		SortOrder:   input.SortOrder,
		Status:      status,
		IsDefault:   false,
	}

	// 校验父部门存在性
	if input.ParentID != nil {
		parent, err := s.deptRepo.GetByID(ctx, *input.ParentID)
		if err != nil {
			return nil, ErrDepartmentNotFound
		}
		if parent.Status == "disabled" {
			return nil, ErrDepartmentDisabled
		}
	}

	if err := s.deptRepo.Create(ctx, dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (s *departmentServiceImpl) Update(ctx context.Context, id int64, input *UpdateDepartmentInput) (*Department, error) {
	dept, err := s.deptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		dept.Name = *input.Name
	}
	if input.Code != nil {
		dept.Code = strings.TrimSpace(*input.Code)
	}
	if input.Description != nil {
		dept.Description = *input.Description
	}
	if input.SortOrder != nil {
		dept.SortOrder = *input.SortOrder
	}
	if input.Status != nil {
		dept.Status = *input.Status
	}
	if input.ParentIDSet {
		dept.ParentID = input.ParentID

		// 防止自环：不能将自己或后代设为自己的父
		if input.ParentID != nil {
			if *input.ParentID == id {
				return nil, ErrDepartmentCircularRef
			}
			descendants, err := s.deptRepo.ListDescendantIDs(ctx, id)
			if err != nil {
				return nil, err
			}
			for _, descID := range descendants {
				if descID == *input.ParentID {
					return nil, ErrDepartmentCircularRef
				}
			}
		}
	}

	if err := s.deptRepo.Update(ctx, dept); err != nil {
		return nil, err
	}
	return dept, nil
}

func (s *departmentServiceImpl) Delete(ctx context.Context, id int64, force bool) error {
	if s.entClient == nil {
		return infraerrors.InternalServer("ENT_CLIENT_UNAVAILABLE", "ent client not configured")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	// 1. 事务内读取目标部门（行锁）
	dept, err := s.deptRepo.GetByIDForUpdate(txCtx, id)
	if err != nil {
		return err
	}
	if dept.IsDefault {
		return ErrCannotDeleteDefaultDepartment
	}

	// 2. 事务内检查子部门
	childCount, err := s.deptRepo.CountChildren(txCtx, id)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrDepartmentHasChildren
	}

	// 3. 事务内检查用户
	userCount, err := s.deptRepo.CountUsers(txCtx, id)
	if err != nil {
		return err
	}
	if userCount > 0 {
		if !force {
			return ErrDepartmentHasUsers
		}
		defaultID, err := s.resolveDefaultDepartmentIDInTx(txCtx)
		if err != nil {
			return err
		}
		if _, err := s.deptRepo.MoveUsersToDepartment(txCtx, id, defaultID); err != nil {
			return err
		}
	}

	// 4. 事务内删除
	if err := s.deptRepo.Delete(txCtx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// resolveDefaultDepartmentIDInTx 在事务内直接查 DB，绕开 sync.Once 进程缓存，
// 避免事务回滚后缓存写入不一致。
func (s *departmentServiceImpl) resolveDefaultDepartmentIDInTx(ctx context.Context) (int64, error) {
	dept, err := s.deptRepo.GetDefault(ctx)
	if err != nil {
		return 0, err
	}
	if dept == nil || dept.ID <= 0 {
		return 0, ErrServiceUnavailable
	}
	return dept.ID, nil
}

func (s *departmentServiceImpl) GetDefaultDepartmentID(ctx context.Context) (int64, error) {
	if s.defaultID > 0 {
		return s.defaultID, nil
	}
	dept, err := s.deptRepo.GetDefault(ctx)
	if err != nil {
		return 0, err
	}
	s.defaultOnce.Do(func() {
		s.defaultID = dept.ID
	})
	return dept.ID, nil
}

func (s *departmentServiceImpl) ListDescendantIDs(ctx context.Context, parentID int64) ([]int64, error) {
	return s.deptRepo.ListDescendantIDs(ctx, parentID)
}
