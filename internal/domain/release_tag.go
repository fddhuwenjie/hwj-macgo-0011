package domain

import "time"

// ReleaseTag 发布标签
type ReleaseTag struct {
	ID           string    `json:"id"`
	ExperimentID string    `json:"experiment_id"`
	RunPlanID    string    `json:"run_plan_id"`
	TagName      string    `json:"tag_name"`
	Status       string    `json:"status"` // active/superseded
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewReleaseTag(id, experimentID, runPlanID, tagName string) *ReleaseTag {
	now := time.Now().UTC()
	return &ReleaseTag{
		ID:           id,
		ExperimentID: experimentID,
		RunPlanID:    runPlanID,
		TagName:      tagName,
		Status:       "active",
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (t *ReleaseTag) Supersede() error {
	if t.Status != "active" {
		return ErrInvalidStatus
	}
	t.Status = "superseded"
	t.UpdatedAt = time.Now().UTC()
	t.Version++
	return nil
}
