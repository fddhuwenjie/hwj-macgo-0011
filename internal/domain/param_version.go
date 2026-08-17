package domain

import (
	"encoding/json"
	"time"
)

// ParamVersion 参数版本
type ParamVersion struct {
	ID           string          `json:"id"`
	ExperimentID string          `json:"experiment_id"`
	Name         string          `json:"name"`
	Definition   json.RawMessage `json:"definition"`
	Version      int             `json:"version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func NewParamVersion(id, experimentID, name string, definition json.RawMessage) *ParamVersion {
	now := time.Now().UTC()
	return &ParamVersion{
		ID:           id,
		ExperimentID: experimentID,
		Name:         name,
		Definition:   definition,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
