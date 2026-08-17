package repository

import (
	"context"
	"encoding/json"
	"sort"

	"experiment-trace/internal/domain"
)

type ExecutionAttemptFileRepo struct {
	store *FileStore
}

func NewExecutionAttemptFileRepo(store *FileStore) *ExecutionAttemptFileRepo {
	return &ExecutionAttemptFileRepo{store: store}
}

func (r *ExecutionAttemptFileRepo) Create(ctx context.Context, ea *domain.ExecutionAttempt) error {
	return r.store.writeEntity(ctx, "execution_attempts", ea.ID, ea, ea.Version)
}

func (r *ExecutionAttemptFileRepo) Get(ctx context.Context, id string) (*domain.ExecutionAttempt, error) {
	var ea domain.ExecutionAttempt
	if err := r.store.readEntity(ctx, "execution_attempts", id, &ea); err != nil {
		return nil, err
	}
	return &ea, nil
}

func (r *ExecutionAttemptFileRepo) Update(ctx context.Context, ea *domain.ExecutionAttempt) error {
	return r.store.writeEntity(ctx, "execution_attempts", ea.ID, ea, ea.Version)
}

func (r *ExecutionAttemptFileRepo) ListByRunPlan(ctx context.Context, runPlanID string) ([]*domain.ExecutionAttempt, error) {
	filterFn := func(data []byte) bool {
		var ea domain.ExecutionAttempt
		if err := json.Unmarshal(data, &ea); err != nil {
			return false
		}
		return ea.RunPlanID == runPlanID
	}
	dataList, err := r.store.listEntities(ctx, "execution_attempts", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.ExecutionAttempt
	for _, data := range dataList {
		var ea domain.ExecutionAttempt
		if err := json.Unmarshal(data, &ea); err != nil {
			return nil, err
		}
		results = append(results, &ea)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].AttemptNo < results[j].AttemptNo
	})
	return results, nil
}

func (r *ExecutionAttemptFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "execution_attempts", id)
}
