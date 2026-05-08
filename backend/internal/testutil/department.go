//go:build unit

package testutil

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 编译期接口断言
var _ service.DepartmentService = StubDepartmentService{}

// StubDepartmentService 是 DepartmentService 的空实现，用于单元测试中满足依赖注入。
type StubDepartmentService struct {
	DefaultID int64
}

func (s StubDepartmentService) List(_ context.Context) ([]service.Department, error) {
	return nil, nil
}
func (s StubDepartmentService) GetByID(_ context.Context, _ int64) (*service.Department, error) {
	return nil, nil
}
func (s StubDepartmentService) Create(_ context.Context, _ *service.CreateDepartmentInput) (*service.Department, error) {
	return nil, nil
}
func (s StubDepartmentService) Update(_ context.Context, _ int64, _ *service.UpdateDepartmentInput) (*service.Department, error) {
	return nil, nil
}
func (s StubDepartmentService) Delete(_ context.Context, _ int64, _ bool) error {
	return nil
}
func (s StubDepartmentService) GetDefaultDepartmentID(_ context.Context) (int64, error) {
	if s.DefaultID > 0 {
		return s.DefaultID, nil
	}
	return 1, nil
}
func (s StubDepartmentService) ListDescendantIDs(_ context.Context, _ int64) ([]int64, error) {
	return nil, nil
}

// StubDepartmentRepository 是 DepartmentRepository 的空实现，用于单元测试。
type StubDepartmentRepository struct{}

func (r StubDepartmentRepository) Create(_ context.Context, _ *service.Department) error             { return nil }
func (r StubDepartmentRepository) GetByID(_ context.Context, _ int64) (*service.Department, error)   { return nil, nil }
func (r StubDepartmentRepository) Update(_ context.Context, _ *service.Department) error             { return nil }
func (r StubDepartmentRepository) Delete(_ context.Context, _ int64) error                           { return nil }
func (r StubDepartmentRepository) List(_ context.Context) ([]service.Department, error)               { return nil, nil }
func (r StubDepartmentRepository) GetDefault(_ context.Context) (*service.Department, error)          { return nil, nil }
func (r StubDepartmentRepository) ListDescendantIDs(_ context.Context, _ int64) ([]int64, error)      { return nil, nil }
func (r StubDepartmentRepository) CountChildren(_ context.Context, _ int64) (int, error)              { return 0, nil }
func (r StubDepartmentRepository) CountUsers(_ context.Context, _ int64) (int, error)                 { return 0, nil }
func (r StubDepartmentRepository) MoveUsersToDepartment(_ context.Context, _, _ int64) (int, error)   { return 0, nil }
func (r StubDepartmentRepository) GetByIDForUpdate(_ context.Context, _ int64) (*service.Department, error) { return nil, nil }
