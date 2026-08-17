package domain

import "time"

// IdempotencyKey 幂等键记录
type IdempotencyKey struct {
	Key          string    `json:"key"`
	RequestType  string    `json:"request_type"`
	EntityID     string    `json:"entity_id"`
	Result       string    `json:"result"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewIdempotencyKey(key, requestType, entityID, result string) *IdempotencyKey {
	return &IdempotencyKey{
		Key:         key,
		RequestType: requestType,
		EntityID:    entityID,
		Result:      result,
		Version:     1,
		CreatedAt:   time.Now().UTC(),
	}
}
