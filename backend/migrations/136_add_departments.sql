-- Add departments table and bind every user to a department.
-- Each user belongs to exactly one department; the migration creates a default
-- department and backfills all existing users so the (NOT NULL) invariant holds
-- the first time the column is added.

-- 1) departments definition table
CREATE TABLE IF NOT EXISTS departments (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(50)  NOT NULL DEFAULT '',
    description TEXT         NOT NULL DEFAULT '',
    parent_id   BIGINT       NULL REFERENCES departments(id) ON DELETE RESTRICT,
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    is_default  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_departments_parent_id
    ON departments(parent_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_departments_status
    ON departments(status);
CREATE INDEX IF NOT EXISTS idx_departments_sort_order
    ON departments(sort_order);
CREATE INDEX IF NOT EXISTS idx_departments_deleted_at
    ON departments(deleted_at);

-- 同级同名互斥（软删除友好）：parent_id 为 NULL 时折算为 0 参与唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_parent_name
    ON departments(COALESCE(parent_id, 0), name) WHERE deleted_at IS NULL;

-- code 非空时全表唯一
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_code
    ON departments(code) WHERE code <> '' AND deleted_at IS NULL;

-- 全局只能存在一个未删除的默认部门
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_is_default
    ON departments((1)) WHERE is_default = TRUE AND deleted_at IS NULL;

COMMENT ON TABLE  departments               IS '组织架构部门，与计费分组 (groups) 解耦';
COMMENT ON COLUMN departments.code          IS '部门短代码，可选；非空时全表唯一';
COMMENT ON COLUMN departments.parent_id     IS '父部门 ID，NULL 表示顶层部门';
COMMENT ON COLUMN departments.status        IS 'active 表示可被新分配；disabled 仅保留旧绑定';
COMMENT ON COLUMN departments.is_default    IS '系统默认部门标记，全局唯一且禁止删除';

-- 2) 创建『默认部门』，作为所有用户的兜底归属
INSERT INTO departments (name, code, is_default, sort_order, status, description)
SELECT '默认', 'DEFAULT', TRUE, 0, 'active', '系统默认部门，不可删除'
WHERE NOT EXISTS (
    SELECT 1 FROM departments WHERE is_default = TRUE AND deleted_at IS NULL
);

-- 3) users 表新增 department_id 列：先 NULL 回填，再设为 NOT NULL
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS department_id BIGINT
    REFERENCES departments(id) ON DELETE RESTRICT;

UPDATE users
SET department_id = (SELECT id FROM departments WHERE is_default = TRUE AND deleted_at IS NULL LIMIT 1)
WHERE department_id IS NULL;

ALTER TABLE users ALTER COLUMN department_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_department_id
    ON users(department_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN users.department_id IS '所属部门 ID，每个用户必须归属唯一部门';
