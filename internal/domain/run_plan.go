package domain

import (
	"time"
)

// RunPlanStatus 运行计划状态
type RunPlanStatus string

const (
	RunPlanDraft     RunPlanStatus = "draft"
	RunPlanFrozen    RunPlanStatus = "frozen"
	RunPlanQueued    RunPlanStatus = "queued"
	RunPlanClaimed   RunPlanStatus = "claimed"
	RunPlanExecuting RunPlanStatus = "executing"
	RunPlanRetryWait RunPlanStatus = "retry_wait"
	RunPlanSealed    RunPlanStatus = "sealed"
	RunPlanFailed    RunPlanStatus = "failed"
)

// RunPlan 运行计划
type RunPlan struct {
	ID                string        `json:"id"`
	ExperimentID      string        `json:"experiment_id"`
	ParamVersionID    string        `json:"param_version_id"`
	InputSnapshotID   string        `json:"input_snapshot_id"`
	Status            RunPlanStatus `json:"status"`
	RetryCount        int           `json:"retry_count"`
	MaxRetries        int           `json:"max_retries"`
	Timeout           time.Duration `json:"timeout"`
	CurrentAttemptNo  int           `json:"current_attempt_no"`
	Version           int           `json:"version"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
}

func NewRunPlan(id, experimentID, paramVersionID, inputSnapshotID string, maxRetries int, timeout time.Duration) *RunPlan {
	now := time.Now().UTC()
	return &RunPlan{
		ID:               id,
		ExperimentID:     experimentID,
		ParamVersionID:   paramVersionID,
		InputSnapshotID:  inputSnapshotID,
		Status:           RunPlanDraft,
		RetryCount:       0,
		MaxRetries:       maxRetries,
		Timeout:          timeout,
		CurrentAttemptNo: 0,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (p *RunPlan) CanTransition(target RunPlanStatus) bool {
	switch p.Status {
	case RunPlanDraft:
		return target == RunPlanFrozen
	case RunPlanFrozen:
		return target == RunPlanQueued
	case RunPlanQueued:
		return target == RunPlanClaimed
	case RunPlanClaimed:
		return target == RunPlanExecuting || target == RunPlanRetryWait
	case RunPlanExecuting:
		return target == RunPlanSealed || target == RunPlanFailed || target == RunPlanRetryWait
	case RunPlanRetryWait:
		return target == RunPlanQueued
	case RunPlanSealed:
		return false // 终态
	case RunPlanFailed:
		return false // 终态
	default:
		return false
	}
}

func (p *RunPlan) Freeze() error {
	if p.Status != RunPlanDraft {
		return ErrInvalidStatus
	}
	p.Status = RunPlanFrozen
	p.UpdatedAt = time.Now().UTC()
	p.Version++
	return nil
}

func (p *RunPlan) Queue() error {
	if p.Status != RunPlanFrozen {
		return ErrInvalidStatus
	}
	p.Status = RunPlanQueued
	p.UpdatedAt = time.Now().UTC()
	p.Version++
	return nil
}

func (p *RunPlan) Claim() error {
	if p.Status != RunPlanQueued {
		return ErrInvalidStatus
	}
	p.Status = RunPlanClaimed
	p.UpdatedAt = time.Now().UTC()
	p.Version++
	return nil
}

func (p *RunPlan) StartExecution() error {
	if p.Status != RunPlanClaimed {
		return ErrInvalidStatus
	}
	p.Status = RunPlanExecuting
	p.UpdatedAt = time.Now().UTC()
	p.Version++
	return nil
}

func (p *RunPlan) Succeed() error {
	if p.Status != RunPlanExecuting {
		return ErrInvalidStatus
	}
	p.Status = RunPlanSealed
	now := time.Now().UTC()
	p.CompletedAt = &now
	p.UpdatedAt = now
	p.Version++
	return nil
}

func (p *RunPlan) FailWithRetry() error {
	if p.Status != RunPlanExecuting {
		return ErrInvalidStatus
	}
	p.RetryCount++
	if p.RetryCount >= p.MaxRetries {
		p.Status = RunPlanFailed
		now := time.Now().UTC()
		p.CompletedAt = &now
		p.UpdatedAt = now
	} else {
		p.Status = RunPlanRetryWait
		p.UpdatedAt = time.Now().UTC()
	}
	p.Version++
	return nil
}

func (p *RunPlan) FailFinal() error {
	if p.Status != RunPlanExecuting {
		return ErrInvalidStatus
	}
	p.Status = RunPlanFailed
	now := time.Now().UTC()
	p.CompletedAt = &now
	p.UpdatedAt = now
	p.Version++
	return nil
}
