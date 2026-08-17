package domain

import "time"

// ExperimentFilter 实验过滤条件
type ExperimentFilter struct {
	Status   ExperimentStatus
	NameLike string
	Limit    int
	Offset   int
}

// RunPlanFilter 运行计划过滤条件
type RunPlanFilter struct {
	ExperimentID string
	Status       []RunPlanStatus
	MinRetries   int
	MaxRetries   int
	CompletedBefore time.Time
	CompletedAfter  time.Time
	SortBy      string // "duration", "retries", "completed_at"
	SortOrder   string // "asc", "desc"
	Limit       int
	Offset      int
}
