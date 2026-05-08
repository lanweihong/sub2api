package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 部门相关业务错误
var (
	ErrDepartmentNotFound             = infraerrors.NotFound("DEPARTMENT_NOT_FOUND", "department not found")
	ErrDepartmentNameExists           = infraerrors.Conflict("DEPARTMENT_NAME_EXISTS", "department name already exists under the same parent")
	ErrDepartmentCodeExists           = infraerrors.Conflict("DEPARTMENT_CODE_EXISTS", "department code already exists")
	ErrCannotDeleteDefaultDepartment  = infraerrors.Forbidden("CANNOT_DELETE_DEFAULT_DEPARTMENT", "cannot delete the default department")
	ErrDepartmentHasChildren          = infraerrors.BadRequest("DEPARTMENT_HAS_CHILDREN", "department has child departments, delete them first")
	ErrDepartmentHasUsers             = infraerrors.BadRequest("DEPARTMENT_HAS_USERS", "department has users, use force to move them to default department")
	ErrDepartmentDisabled             = infraerrors.BadRequest("DEPARTMENT_DISABLED", "cannot assign users to a disabled department")
	ErrDepartmentCircularRef          = infraerrors.BadRequest("DEPARTMENT_CIRCULAR_REF", "moving department would create a circular reference")
	ErrDepartmentDefaultConflict      = infraerrors.Conflict("DEPARTMENT_DEFAULT_CONFLICT", "only one default department is allowed")
)

// DepartmentRepository 部门持久化接口（定义于 service 层，实现于 repository 层）
type DepartmentRepository interface {
	Create(ctx context.Context, dept *Department) error
	GetByID(ctx context.Context, id int64) (*Department, error)
	Update(ctx context.Context, dept *Department) error
	Delete(ctx context.Context, id int64) error

	// List 返回所有未删除部门（树形构建由上层完成）
	List(ctx context.Context) ([]Department, error)

	// GetDefault 返回系统默认部门
	GetDefault(ctx context.Context) (*Department, error)

	// ListDescendantIDs 返回指定部门的所有后代部门 ID（递归 CTE）
	ListDescendantIDs(ctx context.Context, parentID int64) ([]int64, error)

	// CountChildren 返回直接子部门数量
	CountChildren(ctx context.Context, parentID int64) (int, error)

	// CountUsers 返回部门下直接用户数量
	CountUsers(ctx context.Context, departmentID int64) (int, error)

	// MoveUsersToDepartment 将指定部门的所有用户迁移到目标部门
	MoveUsersToDepartment(ctx context.Context, fromDepartmentID, toDepartmentID int64) (int, error)

	// GetByIDForUpdate 读取部门并加 FOR UPDATE 行锁（仅在事务内使用）
	GetByIDForUpdate(ctx context.Context, id int64) (*Department, error)
}

// DepartmentService 部门管理业务接口
type DepartmentService interface {
	List(ctx context.Context) ([]Department, error)
	GetByID(ctx context.Context, id int64) (*Department, error)
	Create(ctx context.Context, input *CreateDepartmentInput) (*Department, error)
	Update(ctx context.Context, id int64, input *UpdateDepartmentInput) (*Department, error)
	Delete(ctx context.Context, id int64, force bool) error
	GetDefaultDepartmentID(ctx context.Context) (int64, error)
	// ListDescendantIDs returns all descendant department IDs (recursive)
	ListDescendantIDs(ctx context.Context, parentID int64) ([]int64, error)
}

type CreateDepartmentInput struct {
	Name        string
	Code        string
	Description string
	ParentID    *int64
	SortOrder   int
	Status      string
}

type UpdateDepartmentInput struct {
	Name        *string
	Code        *string
	Description *string
	ParentID    *int64 // 具体父部门 ID，nil 表示顶级
	ParentIDSet bool   // 请求体中是否显式携带了 parent_id 字段
	SortOrder   *int
	Status      *string
}
