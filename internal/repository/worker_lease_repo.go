package repository

import (
	"context"
	"encoding/json"
	"time"

	"experiment-trace/internal/domain"
)

type WorkerLeaseFileRepo struct {
	store *FileStore
}

func NewWorkerLeaseFileRepo(store *FileStore) *WorkerLeaseFileRepo {
	return &WorkerLeaseFileRepo{store: store}
}

func (r *WorkerLeaseFileRepo) Create(ctx context.Context, wl *domain.WorkerLease) error {
	return r.store.writeEntity(ctx, "worker_leases", wl.ID, wl, wl.Version)
}

func (r *WorkerLeaseFileRepo) Get(ctx context.Context, id string) (*domain.WorkerLease, error) {
	var wl domain.WorkerLease
	if err := r.store.readEntity(ctx, "worker_leases", id, &wl); err != nil {
		return nil, err
	}
	return &wl, nil
}

func (r *WorkerLeaseFileRepo) Update(ctx context.Context, wl *domain.WorkerLease) error {
	return r.store.writeEntity(ctx, "worker_leases", wl.ID, wl, wl.Version)
}

func (r *WorkerLeaseFileRepo) ListActive(ctx context.Context) ([]*domain.WorkerLease, error) {
	now := time.Now().UTC()
	filterFn := func(data []byte) bool {
		var wl domain.WorkerLease
		if err := json.Unmarshal(data, &wl); err != nil {
			return false
		}
		return wl.Status == "active" && now.Before(wl.ExpiresAt)
	}
	dataList, err := r.store.listEntities(ctx, "worker_leases", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.WorkerLease
	for _, data := range dataList {
		var wl domain.WorkerLease
		if err := json.Unmarshal(data, &wl); err != nil {
			return nil, err
		}
		results = append(results, &wl)
	}
	return results, nil
}

func (r *WorkerLeaseFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "worker_leases", id)
}
