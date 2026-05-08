package service

import (
	"time"
)

// Department 部门实体，与计费分组 (Group) 解耦
type Department struct {
	ID          int64
	Name        string
	Code        string
	Description string
	ParentID    *int64
	SortOrder   int
	Status      string // "active" | "disabled"
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
