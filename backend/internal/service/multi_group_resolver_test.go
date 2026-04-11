package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveGroupForModel_SingleBoundGroupSkipsInactive(t *testing.T) {
	group := &Group{
		ID:     101,
		Status: StatusDisabled,
	}
	apiKey := &APIKey{
		BoundGroups: []APIKeyGroup{
			{
				GroupID: group.ID,
				Group:   group,
			},
		},
	}

	resolved, err := ResolveGroupForModel(apiKey, "claude-3-5-sonnet")
	require.ErrorIs(t, err, ErrNoGroupMatchForModel)
	require.Nil(t, resolved)
}

func TestResolveGroupForModel_EmptyModelSkipsInactiveGroups(t *testing.T) {
	inactive := &Group{
		ID:       101,
		Status:   StatusDisabled,
		Platform: PlatformOpenAI,
	}
	active := &Group{
		ID:       102,
		Status:   StatusActive,
		Platform: PlatformGemini,
	}
	apiKey := &APIKey{
		BoundGroups: []APIKeyGroup{
			{
				GroupID:  inactive.ID,
				Priority: 1,
				Group:    inactive,
			},
			{
				GroupID:  active.ID,
				Priority: 2,
				Group:    active,
			},
		},
	}

	resolved, err := ResolveGroupForModel(apiKey, "")
	require.NoError(t, err)
	require.Same(t, active, resolved)
}

func TestResolveGroupForModel_EmptyModelReturnsNoMatchWhenAllGroupsInactive(t *testing.T) {
	apiKey := &APIKey{
		BoundGroups: []APIKeyGroup{
			{
				GroupID:  101,
				Priority: 1,
				Group: &Group{
					ID:       101,
					Status:   StatusDisabled,
					Platform: PlatformOpenAI,
				},
			},
			{
				GroupID:  102,
				Priority: 2,
				Group: &Group{
					ID:       102,
					Status:   StatusDisabled,
					Platform: PlatformAnthropic,
				},
			},
		},
	}

	resolved, err := ResolveGroupForModel(apiKey, "")
	require.ErrorIs(t, err, ErrNoGroupMatchForModel)
	require.Nil(t, resolved)
}
