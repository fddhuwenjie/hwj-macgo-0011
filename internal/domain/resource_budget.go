package domain

import "time"

// ResourceBudget 资源预算
type ResourceBudget struct {
	ID           string        `json:"id"`
	ExperimentID string        `json:"experiment_id"`
	CPU          int           `json:"cpu"`
	MemoryMB     int           `json:"memory_mb"`
	DiskMB       int           `json:"disk_mb"`
	MaxRetries   int           `json:"max_retries"`
	Timeout      time.Duration `json:"timeout"`
	Version      int           `json:"version"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

func NewResourceBudget(id, experimentID string, cpu, memoryMB, diskMB, maxRetries int, timeout time.Duration) *ResourceBudget {
	now := time.Now().UTC()
	return &ResourceBudget{
		ID:           id,
		ExperimentID: experimentID,
		CPU:          cpu,
		MemoryMB:     memoryMB,
		DiskMB:       diskMB,
		MaxRetries:   maxRetries,
		Timeout:      timeout,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
