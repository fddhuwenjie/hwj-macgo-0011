package domain

import "time"

// LineageEdge 谱系边
type LineageEdge struct {
	ID             string    `json:"id"`
	FromEntityID   string    `json:"from_entity_id"`
	FromEntityType string    `json:"from_entity_type"`
	ToEntityID     string    `json:"to_entity_id"`
	ToEntityType   string    `json:"to_entity_type"`
	Relation       string    `json:"relation"`
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewLineageEdge(id, fromID, fromType, toID, toType, relation string) *LineageEdge {
	return &LineageEdge{
		ID:             id,
		FromEntityID:   fromID,
		FromEntityType: fromType,
		ToEntityID:     toID,
		ToEntityType:   toType,
		Relation:       relation,
		Version:        1,
		CreatedAt:      time.Now().UTC(),
	}
}
