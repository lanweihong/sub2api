//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserRoleHelpers(t *testing.T) {
	require.True(t, (&User{Role: RoleSuperAdmin}).IsAdmin())
	require.True(t, (&User{Role: RoleAdmin}).IsAdmin())
	require.False(t, (&User{Role: RoleUser}).IsAdmin())

	require.True(t, (&User{Role: RoleSuperAdmin}).IsSuperAdmin())
	require.False(t, (&User{Role: RoleAdmin}).IsSuperAdmin())
}
