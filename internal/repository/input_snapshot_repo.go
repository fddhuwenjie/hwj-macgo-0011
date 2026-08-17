package repository

import (
	"context"

	"experiment-trace/internal/domain"
)

type InputSnapshotFileRepo struct {
	store *FileStore
}

func NewInputSnapshotFileRepo(store *FileStore) *InputSnapshotFileRepo {
	return &InputSnapshotFileRepo{store: store}
}

func (r *InputSnapshotFileRepo) Create(ctx context.Context, is *domain.InputSnapshot) error {
	return r.store.writeEntity(ctx, "input_snapshots", is.ID, is, is.Version)
}

func (r *InputSnapshotFileRepo) Get(ctx context.Context, id string) (*domain.InputSnapshot, error) {
	var is domain.InputSnapshot
	if err := r.store.readEntity(ctx, "input_snapshots", id, &is); err != nil {
		return nil, err
	}
	return &is, nil
}

func (r *InputSnapshotFileRepo) Update(ctx context.Context, is *domain.InputSnapshot) error {
	return r.store.writeEntity(ctx, "input_snapshots", is.ID, is, is.Version)
}

func (r *InputSnapshotFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "input_snapshots", id)
}
