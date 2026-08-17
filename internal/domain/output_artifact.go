package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// ArtifactStatus 制品状态
type ArtifactStatus string

const (
	ArtifactDraft  ArtifactStatus = "draft"
	ArtifactSealed ArtifactStatus = "sealed"
)

// OutputArtifact 输出制品
type OutputArtifact struct {
	ID        string          `json:"id"`
	AttemptID string          `json:"attempt_id"`
	RunPlanID string          `json:"run_plan_id"`
	Content   json.RawMessage `json:"content"`
	Hash      string          `json:"hash"`
	Status    ArtifactStatus  `json:"status"`
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func NewOutputArtifact(id, attemptID, runPlanID string, content json.RawMessage) *OutputArtifact {
	now := time.Now().UTC()
	h := sha256.Sum256(content)
	return &OutputArtifact{
		ID:        id,
		AttemptID: attemptID,
		RunPlanID: runPlanID,
		Content:   content,
		Hash:      hex.EncodeToString(h[:]),
		Status:    ArtifactDraft,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (a *OutputArtifact) Seal() error {
	if a.Status != ArtifactDraft {
		return ErrInvalidStatus
	}
	a.Status = ArtifactSealed
	a.UpdatedAt = time.Now().UTC()
	a.Version++
	return nil
}
