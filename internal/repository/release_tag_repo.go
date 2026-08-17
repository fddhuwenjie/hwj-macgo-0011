package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

type ReleaseTagFileRepo struct {
	store *FileStore
}

func NewReleaseTagFileRepo(store *FileStore) *ReleaseTagFileRepo {
	return &ReleaseTagFileRepo{store: store}
}

func (r *ReleaseTagFileRepo) Create(ctx context.Context, rt *domain.ReleaseTag) error {
	return r.store.writeEntity(ctx, "release_tags", rt.ID, rt, rt.Version)
}

func (r *ReleaseTagFileRepo) Get(ctx context.Context, id string) (*domain.ReleaseTag, error) {
	var rt domain.ReleaseTag
	if err := r.store.readEntity(ctx, "release_tags", id, &rt); err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *ReleaseTagFileRepo) Update(ctx context.Context, rt *domain.ReleaseTag) error {
	return r.store.writeEntity(ctx, "release_tags", rt.ID, rt, rt.Version)
}

func (r *ReleaseTagFileRepo) ListByExperiment(ctx context.Context, experimentID string) ([]*domain.ReleaseTag, error) {
	filterFn := func(data []byte) bool {
		var rt domain.ReleaseTag
		if err := json.Unmarshal(data, &rt); err != nil {
			return false
		}
		return rt.ExperimentID == experimentID
	}
	dataList, err := r.store.listEntities(ctx, "release_tags", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.ReleaseTag
	for _, data := range dataList {
		var rt domain.ReleaseTag
		if err := json.Unmarshal(data, &rt); err != nil {
			return nil, err
		}
		results = append(results, &rt)
	}
	return results, nil
}

func (r *ReleaseTagFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "release_tags", id)
}
