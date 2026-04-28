-- Migration: 131_promote_initial_super_admin
-- Introduce a single super administrator by promoting the earliest existing admin.

UPDATE users
SET role = 'super_admin',
    updated_at = NOW()
WHERE id = (
    SELECT id
    FROM users
    WHERE role = 'admin'
      AND deleted_at IS NULL
    ORDER BY id ASC
    LIMIT 1
)
AND NOT EXISTS (
    SELECT 1
    FROM users
    WHERE role = 'super_admin'
      AND deleted_at IS NULL
);
