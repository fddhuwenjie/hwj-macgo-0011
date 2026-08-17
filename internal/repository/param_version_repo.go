package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

type ParamVersionFileRepo struct {
	store *FileStore
}

func NewParamVersionFileRepo(store *FileStore) *ParamVersionFileRepo {
	return &ParamVersionFileRepo{store: store}
}

func (r *ParamVersionFileRepo) Create(ctx context.Context, pv *domain.ParamVersion) error {
	return r.store.writeEntity(ctx, "param_versions", pv.ID, pv, pv.Version)
}

func (r *ParamVersionFileRepo) Get(ctx context.Context, id string) (*domain.ParamVersion, error) {
	var pv domain.ParamVersion
	if err := r.store.readEntity(ctx, "param_versions", id, &pv); err != nil {
		return nil, err
	}
	return &pv, nil
}

func (r *ParamVersionFileRepo) Update(ctx context.Context, pv *domain.ParamVersion) error {
	return r.store.writeEntity(ctx, "param_versions", pv.ID, pv, pv.Version)
}

func (r *ParamVersionFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "param_versions", id)
}

var _ = json.Marshal
