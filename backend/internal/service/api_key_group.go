package service

import "time"

// APIKeyGroup represents the association between an API key and a group,
// including priority and optional model pattern overrides for multi-group routing.
type APIKeyGroup struct {
	APIKeyID      int64
	GroupID       int64
	Priority      int
	ModelPatterns []string
	CreatedAt     time.Time

	Group *Group
}

// APIKeyGroupBinding is the input for setting bound groups on an API key.
type APIKeyGroupBinding struct {
	GroupID       int64    `json:"group_id"`
	Priority      int      `json:"priority"`
	ModelPatterns []string `json:"model_patterns,omitempty"`
}
