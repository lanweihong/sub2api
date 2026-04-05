//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type compatAdminAccountRepoStub struct {
	accountRepoStub
	account   *Account
	created   *Account
	updated   *Account
	createErr error
	updateErr error
}

func (s *compatAdminAccountRepoStub) Create(_ context.Context, account *Account) error {
	s.created = account
	return s.createErr
}

func (s *compatAdminAccountRepoStub) GetByID(_ context.Context, _ int64) (*Account, error) {
	if s.account == nil {
		return nil, ErrAccountNotFound
	}
	return s.account, nil
}

func (s *compatAdminAccountRepoStub) Update(_ context.Context, account *Account) error {
	s.updated = account
	return s.updateErr
}

func TestAdminService_CreateAccountAnthropicCompatibleRequiresBaseURL(t *testing.T) {
	repo := &compatAdminAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "other-compat",
		Platform:             PlatformAnthropicCompatible,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test"},
		Concurrency:          1,
		Priority:             1,
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.ErrorContains(t, err, "必须设置 base_url")
	require.Nil(t, repo.created)
}

func TestAdminService_CreateAccountAnthropicCompatibleOnlySupportsAPIKey(t *testing.T) {
	repo := &compatAdminAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "other-compat",
		Platform:             PlatformAnthropicCompatible,
		Type:                 AccountTypeOAuth,
		Credentials:          map[string]any{"base_url": "https://relay.example.com", "access_token": "tok"},
		Concurrency:          1,
		Priority:             1,
		SkipDefaultGroupBind: true,
	})

	require.Nil(t, account)
	require.ErrorContains(t, err, "仅支持 API Key")
	require.Nil(t, repo.created)
}

func TestAdminService_UpdateAccountAnthropicCompatibleRequiresBaseURL(t *testing.T) {
	repo := &compatAdminAccountRepoStub{
		account: &Account{
			ID:       1,
			Name:     "other-compat",
			Platform: PlatformAnthropicCompatible,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "sk-test",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		Name: "other-compat-updated",
	})

	require.Nil(t, account)
	require.ErrorContains(t, err, "必须设置 base_url")
	require.Nil(t, repo.updated)
}
