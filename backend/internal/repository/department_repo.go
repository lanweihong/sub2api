package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/department"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// deptConstraintMap 将 PostgreSQL 唯一约束名映射到精确的业务错误。
var deptConstraintMap = map[string]*infraerrors.ApplicationError{
	"uq_departments_code":        service.ErrDepartmentCodeExists,
	"uq_departments_parent_name": service.ErrDepartmentNameExists,
	"uq_departments_is_default":  service.ErrDepartmentDefaultConflict,
}

type departmentRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewDepartmentRepository(client *dbent.Client, sqlDB *sql.DB) service.DepartmentRepository {
	return &departmentRepository{client: client, sql: sqlDB}
}

func (r *departmentRepository) Create(ctx context.Context, dept *service.Department) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Department.Create().
		SetName(dept.Name).
		SetCode(dept.Code).
		SetDescription(dept.Description).
		SetSortOrder(dept.SortOrder).
		SetStatus(dept.Status).
		SetIsDefault(dept.IsDefault)

	if dept.ParentID != nil {
		builder = builder.SetParentID(*dept.ParentID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		if mapped := translateConstraintConflict(err, deptConstraintMap); mapped != nil {
			return mapped
		}
		return translatePersistenceError(err, nil, service.ErrDepartmentNameExists)
	}

	dept.ID = created.ID
	dept.CreatedAt = created.CreatedAt
	dept.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *departmentRepository) GetByID(ctx context.Context, id int64) (*service.Department, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.Department.Query().Where(department.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrDepartmentNotFound, nil)
	}
	return departmentEntityToService(m), nil
}

func (r *departmentRepository) Update(ctx context.Context, dept *service.Department) error {
	client := clientFromContext(ctx, r.client)
	builder := client.Department.UpdateOneID(dept.ID).
		SetName(dept.Name).
		SetCode(dept.Code).
		SetDescription(dept.Description).
		SetSortOrder(dept.SortOrder).
		SetStatus(dept.Status)

	if dept.ParentID != nil {
		builder = builder.SetParentID(*dept.ParentID)
	} else {
		builder = builder.ClearParentID()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if mapped := translateConstraintConflict(err, deptConstraintMap); mapped != nil {
			return mapped
		}
		return translatePersistenceError(err, service.ErrDepartmentNotFound, service.ErrDepartmentNameExists)
	}
	dept.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *departmentRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	affected, err := client.Department.Delete().Where(department.IDEQ(id)).Exec(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrDepartmentNotFound, nil)
	}
	if affected == 0 {
		return service.ErrDepartmentNotFound
	}
	return nil
}

func (r *departmentRepository) List(ctx context.Context) ([]service.Department, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.Department.Query().
		Order(dbent.Asc(department.FieldSortOrder), dbent.Asc(department.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.Department, 0, len(rows))
	for _, m := range rows {
		out = append(out, *departmentEntityToService(m))
	}
	return out, nil
}

func (r *departmentRepository) GetDefault(ctx context.Context) (*service.Department, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.Department.Query().
		Where(department.IsDefaultEQ(true)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrDepartmentNotFound, nil)
	}
	return departmentEntityToService(m), nil
}

func (r *departmentRepository) ListDescendantIDs(ctx context.Context, parentID int64) ([]int64, error) {
	// 递归 CTE 查询所有后代部门 ID
	const query = `
		WITH RECURSIVE descendants AS (
			SELECT id FROM departments WHERE parent_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT d.id FROM departments d
			INNER JOIN descendants ds ON d.parent_id = ds.id
			WHERE d.deleted_at IS NULL
		)
		SELECT id FROM descendants
	`
	rows, err := r.sql.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *departmentRepository) CountChildren(ctx context.Context, parentID int64) (int, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.Department.Query().
		Where(department.ParentIDEQ(parentID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *departmentRepository) CountUsers(ctx context.Context, departmentID int64) (int, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.User.Query().
		Where(dbuser.DepartmentIDEQ(departmentID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *departmentRepository) GetByIDForUpdate(ctx context.Context, id int64) (*service.Department, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.Department.Query().
		Where(department.IDEQ(id)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrDepartmentNotFound, nil)
	}
	return departmentEntityToService(m), nil
}

func (r *departmentRepository) MoveUsersToDepartment(ctx context.Context, fromDepartmentID, toDepartmentID int64) (int, error) {
	client := clientFromContext(ctx, r.client)
	affected, err := client.User.Update().
		Where(dbuser.DepartmentIDEQ(fromDepartmentID)).
		SetDepartmentID(toDepartmentID).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func departmentEntityToService(m *dbent.Department) *service.Department {
	if m == nil {
		return nil
	}
	return &service.Department{
		ID:          m.ID,
		Name:        m.Name,
		Code:        m.Code,
		Description: m.Description,
		ParentID:    m.ParentID,
		SortOrder:   m.SortOrder,
		Status:      m.Status,
		IsDefault:   m.IsDefault,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
