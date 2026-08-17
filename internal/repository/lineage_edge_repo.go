package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

type LineageEdgeFileRepo struct {
	store *FileStore
}

func NewLineageEdgeFileRepo(store *FileStore) *LineageEdgeFileRepo {
	return &LineageEdgeFileRepo{store: store}
}

func (r *LineageEdgeFileRepo) Create(ctx context.Context, le *domain.LineageEdge) error {
	return r.store.writeEntity(ctx, "lineage_edges", le.ID, le, le.Version)
}

func (r *LineageEdgeFileRepo) Get(ctx context.Context, id string) (*domain.LineageEdge, error) {
	var le domain.LineageEdge
	if err := r.store.readEntity(ctx, "lineage_edges", id, &le); err != nil {
		return nil, err
	}
	return &le, nil
}

func (r *LineageEdgeFileRepo) ListByFrom(ctx context.Context, fromID string) ([]*domain.LineageEdge, error) {
	filterFn := func(data []byte) bool {
		var le domain.LineageEdge
		if err := json.Unmarshal(data, &le); err != nil {
			return false
		}
		return le.FromEntityID == fromID
	}
	dataList, err := r.store.listEntities(ctx, "lineage_edges", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.LineageEdge
	for _, data := range dataList {
		var le domain.LineageEdge
		if err := json.Unmarshal(data, &le); err != nil {
			return nil, err
		}
		results = append(results, &le)
	}
	return results, nil
}

func (r *LineageEdgeFileRepo) ListByTo(ctx context.Context, toID string) ([]*domain.LineageEdge, error) {
	filterFn := func(data []byte) bool {
		var le domain.LineageEdge
		if err := json.Unmarshal(data, &le); err != nil {
			return false
		}
		return le.ToEntityID == toID
	}
	dataList, err := r.store.listEntities(ctx, "lineage_edges", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.LineageEdge
	for _, data := range dataList {
		var le domain.LineageEdge
		if err := json.Unmarshal(data, &le); err != nil {
			return nil, err
		}
		results = append(results, &le)
	}
	return results, nil
}

func (r *LineageEdgeFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "lineage_edges", id)
}
