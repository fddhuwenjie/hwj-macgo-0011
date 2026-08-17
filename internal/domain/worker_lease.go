package domain

import "time"

// WorkerLease 工作者租约
type WorkerLease struct {
	ID              string     `json:"id"`
	WorkerID        string     `json:"worker_id"`
	RunPlanID       string     `json:"run_plan_id"`
	AttemptID       string     `json:"attempt_id"`
	LeaseGeneration int        `json:"lease_generation"`
	ExpiresAt       time.Time  `json:"expires_at"`
	ReleasedAt      *time.Time `json:"released_at,omitempty"`
	Status          string     `json:"status"` // active/released
	Version         int        `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func NewWorkerLease(id, workerID, runPlanID, attemptID string, leaseDuration time.Duration) *WorkerLease {
	now := time.Now().UTC()
	return &WorkerLease{
		ID:              id,
		WorkerID:        workerID,
		RunPlanID:       runPlanID,
		AttemptID:       attemptID,
		LeaseGeneration: 1,
		ExpiresAt:       now.Add(leaseDuration),
		Status:          "active",
		Version:         1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (l *WorkerLease) IsExpired(now time.Time) bool {
	return l.Status == "active" && now.After(l.ExpiresAt)
}

func (l *WorkerLease) Release() error {
	if l.Status != "active" {
		return ErrInvalidStatus
	}
	l.Status = "released"
	now := time.Now().UTC()
	l.ReleasedAt = &now
	l.UpdatedAt = now
	l.Version++
	return nil
}

func (l *WorkerLease) Renew(leaseDuration time.Duration) error {
	if l.Status != "active" {
		return ErrInvalidStatus
	}
	l.ExpiresAt = time.Now().UTC().Add(leaseDuration)
	l.LeaseGeneration++
	l.UpdatedAt = time.Now().UTC()
	l.Version++
	return nil
}
