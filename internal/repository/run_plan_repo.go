package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

type RunPlanFileRepo struct {
	store *FileStore
}

func NewRunPlanFileRepo(store *FileStore) *RunPlanFileRepo {
	return &RunPlanFileRepo{store: store}
}

func (r *RunPlanFileRepo) Create(ctx context.Context, rp *domain.RunPlan) error {
	return r.store.writeEntity(ctx, "run_plans", rp.ID, rp, rp.Version)
}

func (r *RunPlanFileRepo) Get(ctx context.Context, id string) (*domain.RunPlan, error) {
	var rp domain.RunPlan
	if err := r.store.readEntity(ctx, "run_plans", id, &rp); err != nil {
		return nil, err
	}
	return &rp, nil
}

func (r *RunPlanFileRepo) Update(ctx context.Context, rp *domain.RunPlan) error {
	return r.store.writeEntity(ctx, "run_plans", rp.ID, rp, rp.Version)
}

func (r *RunPlanFileRepo) ListByStatus(ctx context.Context, statuses []domain.RunPlanStatus) ([]*domain.RunPlan, error) {
	statusSet := make(map[domain.RunPlanStatus]bool)
	for _, s := range statuses {
		statusSet[s] = true
	}
	filterFn := func(data []byte) bool {
		var rp domain.RunPlan
		if err := json.Unmarshal(data, &rp); err != nil {
			return false
		}
		return statusSet[rp.Status]
	}
	dataList, err := r.store.listEntities(ctx, "run_plans", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.RunPlan
	for _, data := range dataList {
		var rp domain.RunPlan
		if err := json.Unmarshal(data, &rp); err != nil {
			return nil, err
		}
		results = append(results, &rp)
	}
	return results, nil
}

func (r *RunPlanFileRepo) List(ctx context.Context, filter domain.RunPlanFilter) ([]*domain.RunPlan, error) {
	filterFn := func(data []byte) bool {
		var rp domain.RunPlan
		if err := json.Unmarshal(data, &rp); err != nil {
			return false
		}
		if filter.ExperimentID != "" && rp.ExperimentID != filter.ExperimentID {
			return false
		}
		if len(filter.Status) > 0 {
			found := false
			for _, s := range filter.Status {
				if rp.Status == s {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		if filter.MinRetries > 0 && rp.RetryCount < filter.MinRetries {
			return false
		}
		if filter.MaxRetries > 0 && rp.RetryCount > filter.MaxRetries {
			return false
		}
		if !filter.CompletedBefore.IsZero() && rp.CompletedAt != nil && rp.CompletedAt.After(filter.CompletedBefore) {
			return false
		}
		if !filter.CompletedAfter.IsZero() && rp.CompletedAt != nil && rp.CompletedAt.Before(filter.CompletedAfter) {
			return false
		}
		return true
	}
	dataList, err := r.store.listEntities(ctx, "run_plans", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.RunPlan
	for _, data := range dataList {
		var rp domain.RunPlan
		if err := json.Unmarshal(data, &rp); err != nil {
			return nil, err
		}
		results = append(results, &rp)
	}
	// 分页由application层处理，这里返回全部
	return results, nil
}

func (r *RunPlanFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "run_plans", id)
}
