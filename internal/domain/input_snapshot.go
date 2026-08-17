package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// InputSnapshot 输入快照，不可变
type InputSnapshot struct {
	ID           string          `json:"id"`
	ExperimentID string          `json:"experiment_id"`
	Content      json.RawMessage `json:"content"`
	Hash         string          `json:"hash"`
	Immutable    bool            `json:"immutable"`
	Version      int             `json:"version"`
	CreatedAt    time.Time       `json:"created_at"`
}

func NewInputSnapshot(id, experimentID string, content json.RawMessage) *InputSnapshot {
	now := time.Now().UTC()
	h := sha256.Sum256(content)
	return &InputSnapshot{
		ID:           id,
		ExperimentID: experimentID,
		Content:      content,
		Hash:         hex.EncodeToString(h[:]),
		Immutable:    true,
		Version:      1,
		CreatedAt:    now,
	}
}
