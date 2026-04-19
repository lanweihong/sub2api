//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type batchUserRepoStub struct {
	userRepoStub
	existingEmails map[string]bool
	createByEmail  map[string]error
}

func (s *batchUserRepoStub) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if s.existsErr != nil {
		return false, s.existsErr
	}
	return s.existingEmails[email], nil
}

func (s *batchUserRepoStub) Create(ctx context.Context, user *User) error {
	if err := s.createByEmail[user.Email]; err != nil {
		return err
	}
	return s.userRepoStub.Create(ctx, user)
}

func TestAdminService_PreviewBatchUsers_AutoSuffixesDuplicatePinyin(t *testing.T) {
	repo := &batchUserRepoStub{existingEmails: map[string]bool{}}
	svc := &adminServiceImpl{userRepo: repo}

	items, err := svc.PreviewBatchUsers(context.Background(), []string{"张三", "张三", "李四"})
	require.NoError(t, err)
	require.Len(t, items, 3)

	require.Equal(t, "zhangsan", items[0].Username)
	require.Equal(t, "zhangsan@xssio.com", items[0].Email)
	require.Empty(t, items[0].Errors)

	require.Equal(t, "zhangsan2", items[1].Username)
	require.Equal(t, "zhangsan2@xssio.com", items[1].Email)
	require.Empty(t, items[1].Errors)

	require.Equal(t, "lisi", items[2].Username)
	require.Equal(t, 9999.0, items[2].Balance)
	require.Equal(t, 3, items[2].Concurrency)
}

func TestAdminService_PreviewBatchUsers_MarksExistingEmailConflicts(t *testing.T) {
	repo := &batchUserRepoStub{existingEmails: map[string]bool{
		"zhangsan@xssio.com": true,
	}}
	svc := &adminServiceImpl{userRepo: repo}

	items, err := svc.PreviewBatchUsers(context.Background(), []string{"张三"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotEmpty(t, items[0].Errors)
	require.Contains(t, items[0].Errors, BatchUserFieldError{
		Field:   "email",
		Code:    "EMAIL_EXISTS",
		Message: "email already exists",
	})
}

func TestAdminService_CreateUsersBatch_Success(t *testing.T) {
	repo := &batchUserRepoStub{
		userRepoStub:   userRepoStub{nextID: 101},
		existingEmails: map[string]bool{},
	}
	svc := &adminServiceImpl{userRepo: repo}

	result, err := svc.CreateUsersBatch(context.Background(), []BatchCreateUserInput{
		{
			RowNo:       1,
			SourceName:  "张三",
			Email:       "zhangsan@xssio.com",
			Password:    "pass1234",
			Username:    "zhangsan",
			Balance:     9999,
			Concurrency: 3,
		},
		{
			RowNo:       2,
			SourceName:  "李四",
			Email:       "lisi@xssio.com",
			Password:    "pass1234",
			Username:    "lisi",
			Balance:     9999,
			Concurrency: 3,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, result.CreatedCount)
	require.Zero(t, result.FailedCount)
	require.Len(t, result.Users, 2)
	require.Empty(t, result.Errors)
	require.Len(t, repo.created, 2)
}

func TestAdminService_CreateUsersBatch_ReturnsRowErrorsWithoutCreating(t *testing.T) {
	repo := &batchUserRepoStub{
		existingEmails: map[string]bool{
			"existing@xssio.com": true,
		},
	}
	svc := &adminServiceImpl{userRepo: repo}

	result, err := svc.CreateUsersBatch(context.Background(), []BatchCreateUserInput{
		{
			RowNo:       1,
			SourceName:  "张三",
			Email:       "existing@xssio.com",
			Password:    "pass1234",
			Username:    "zhangsan",
			Balance:     9999,
			Concurrency: 3,
		},
		{
			RowNo:       2,
			SourceName:  "李四",
			Email:       "existing@xssio.com",
			Password:    "123",
			Username:    "zhangsan",
			Balance:     -1,
			Concurrency: 0,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Zero(t, result.CreatedCount)
	require.Equal(t, 2, result.FailedCount)
	require.NotEmpty(t, result.Errors)
	require.Empty(t, repo.created)
}
