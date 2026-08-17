package domain

import "time"

// AttemptStatus 尝试状态
type AttemptStatus string

const (
	AttemptQueued    AttemptStatus = "queued"
	AttemptClaimed   AttemptStatus = "claimed"
	AttemptExecuting AttemptStatus = "executing"
	AttemptSucceeded AttemptStatus = "succeeded"
	AttemptFailed    AttemptStatus = "failed"
)

// ExecutionAttempt 执行尝试
type ExecutionAttempt struct {
	ID            string        `json:"id"`
	RunPlanID     string        `json:"run_plan_id"`
	AttemptNo     int           `json:"attempt_no"`
	Status        AttemptStatus `json:"status"`
	WorkerLeaseID string        `json:"worker_lease_id,omitempty"`
	StartTime     *time.Time    `json:"start_time,omitempty"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	Version       int           `json:"version"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func NewExecutionAttempt(id, runPlanID string, attemptNo int) *ExecutionAttempt {
	now := time.Now().UTC()
	return &ExecutionAttempt{
		ID:        id,
		RunPlanID: runPlanID,
		AttemptNo: attemptNo,
		Status:    AttemptQueued,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (a *ExecutionAttempt) Claim(workerLeaseID string) error {
	if a.Status != AttemptQueued {
		return ErrInvalidStatus
	}
	a.Status = AttemptClaimed
	a.WorkerLeaseID = workerLeaseID
	a.UpdatedAt = time.Now().UTC()
	a.Version++
	return nil
}

func (a *ExecutionAttempt) Start() error {
	if a.Status != AttemptClaimed {
		return ErrInvalidStatus
	}
	a.Status = AttemptExecuting
	now := time.Now().UTC()
	a.StartTime = &now
	a.UpdatedAt = now
	a.Version++
	return nil
}

func (a *ExecutionAttempt) Succeed() error {
	if a.Status != AttemptExecuting {
		return ErrInvalidStatus
	}
	a.Status = AttemptSucceeded
	now := time.Now().UTC()
	a.EndTime = &now
	a.UpdatedAt = now
	a.Version++
	return nil
}

func (a *ExecutionAttempt) Fail(errMsg string) error {
	if a.Status != AttemptExecuting {
		return ErrInvalidStatus
	}
	a.Status = AttemptFailed
	now := time.Now().UTC()
	a.EndTime = &now
	a.ErrorMessage = errMsg
	a.UpdatedAt = now
	a.Version++
	return nil
}
