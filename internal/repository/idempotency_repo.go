package repository

import (
	"context"

	"experiment-trace/internal/domain"
)

type IdempotencyFileRepo struct {
	store *FileStore
}

func NewIdempotencyFileRepo(store *FileStore) *IdempotencyFileRepo {
	return &IdempotencyFileRepo{store: store}
}

func (r *IdempotencyFileRepo) Get(ctx context.Context, key string) (*domain.IdempotencyKey, error) {
	var ik domain.IdempotencyKey
	if err := r.store.readEntity(ctx, "idempotency_keys", key, &ik); err != nil {
		return nil, err
	}
	return &ik, nil
}

func (r *IdempotencyFileRepo) Create(ctx context.Context, ik *domain.IdempotencyKey) error {
	return r.store.writeEntity(ctx, "idempotency_keys", ik.Key, ik, ik.Version)
}
