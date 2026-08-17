package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

type OutputArtifactFileRepo struct {
	store *FileStore
}

func NewOutputArtifactFileRepo(store *FileStore) *OutputArtifactFileRepo {
	return &OutputArtifactFileRepo{store: store}
}

func (r *OutputArtifactFileRepo) Create(ctx context.Context, oa *domain.OutputArtifact) error {
	return r.store.writeEntity(ctx, "output_artifacts", oa.ID, oa, oa.Version)
}

func (r *OutputArtifactFileRepo) Get(ctx context.Context, id string) (*domain.OutputArtifact, error) {
	var oa domain.OutputArtifact
	if err := r.store.readEntity(ctx, "output_artifacts", id, &oa); err != nil {
		return nil, err
	}
	return &oa, nil
}

func (r *OutputArtifactFileRepo) Update(ctx context.Context, oa *domain.OutputArtifact) error {
	return r.store.writeEntity(ctx, "output_artifacts", oa.ID, oa, oa.Version)
}

func (r *OutputArtifactFileRepo) GetByAttempt(ctx context.Context, attemptID string) (*domain.OutputArtifact, error) {
	filterFn := func(data []byte) bool {
		var oa domain.OutputArtifact
		if err := json.Unmarshal(data, &oa); err != nil {
			return false
		}
		return oa.AttemptID == attemptID
	}
	dataList, err := r.store.listEntities(ctx, "output_artifacts", filterFn)
	if err != nil {
		return nil, err
	}
	if len(dataList) == 0 {
		return nil, domain.ErrNotFound
	}
	var oa domain.OutputArtifact
	if err := json.Unmarshal(dataList[0], &oa); err != nil {
		return nil, err
	}
	return &oa, nil
}

func (r *OutputArtifactFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "output_artifacts", id)
}
