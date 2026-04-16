//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type updateResponseRepoStub struct {
	authRepoStub
	keys          []*APIKey
	getByIDCalls  int
	updateCalled  bool
	updateBinding *[]APIKeyGroupBinding
}

func (s *updateResponseRepoStub) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	if s.getByIDCalls >= len(s.keys) {
		panic("unexpected GetByID call")
	}
	key := s.keys[s.getByIDCalls]
	s.getByIDCalls++
	clone := *key
	return &clone, nil
}

func (s *updateResponseRepoStub) UpdateWithBoundGroups(ctx context.Context, key *APIKey, bindings *[]APIKeyGroupBinding) error {
	s.updateCalled = true
	s.updateBinding = bindings
	return nil
}

type updateResponseUserRepoStub struct {
	mockUserRepo
	user *User
}

func (s *updateResponseUserRepoStub) GetByID(ctx context.Context, id int64) (*User, error) {
	clone := *s.user
	return &clone, nil
}

type updateResponseGroupRepoStub struct {
	groupRepoNoop
	group *Group
}

func (s *updateResponseGroupRepoStub) GetByID(ctx context.Context, id int64) (*Group, error) {
	clone := *s.group
	return &clone, nil
}

func TestAPIKeyService_Update_ReturnsRefreshedKeyState(t *testing.T) {
	oldGroupID := int64(1)
	newGroupID := int64(2)
	repo := &updateResponseRepoStub{
		keys: []*APIKey{
			{
				ID:      99,
				UserID:  7,
				Key:     "sk-test",
				Name:    "key",
				Status:  StatusActive,
				GroupID: &oldGroupID,
				Group:   &Group{ID: oldGroupID, Name: "old-group", Status: StatusActive},
			},
			{
				ID:      99,
				UserID:  7,
				Key:     "sk-test",
				Name:    "key",
				Status:  StatusActive,
				GroupID: &newGroupID,
				Group:   &Group{ID: newGroupID, Name: "new-group", Status: StatusActive},
			},
		},
	}
	userRepo := &updateResponseUserRepoStub{
		user: &User{ID: 7, AllowedGroups: []int64{newGroupID}},
	}
	groupRepo := &updateResponseGroupRepoStub{
		group: &Group{ID: newGroupID, Name: "new-group", Status: StatusActive},
	}
	svc := NewAPIKeyService(repo, userRepo, groupRepo, nil, nil, nil, nil)

	updated, err := svc.Update(context.Background(), 99, 7, UpdateAPIKeyRequest{
		GroupID: &newGroupID,
	})
	require.NoError(t, err)
	require.True(t, repo.updateCalled)
	require.Equal(t, 2, repo.getByIDCalls, "expected refresh load after update")
	require.NotNil(t, updated.GroupID)
	require.Equal(t, newGroupID, *updated.GroupID)
	require.NotNil(t, updated.Group)
	require.Equal(t, "new-group", updated.Group.Name)
}
