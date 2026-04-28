-- Migration: 132_ensure_active_super_admin
-- If the super_admin promoted by 131 is disabled/inactive, re-promote the earliest active admin.

UPDATE users
SET role = 'super_admin',
    updated_at = NOW()
WHERE id = (
    SELECT id
    FROM users
    WHERE role = 'admin'
      AND status = 'active'
      AND deleted_at IS NULL
    ORDER BY id ASC
    LIMIT 1
)
AND EXISTS (
    SELECT 1
    FROM users
    WHERE role = 'super_admin'
      AND status != 'active'
      AND deleted_at IS NULL
)
AND NOT EXISTS (
    SELECT 1
    FROM users
    WHERE role = 'super_admin'
      AND status = 'active'
      AND deleted_at IS NULL
);
