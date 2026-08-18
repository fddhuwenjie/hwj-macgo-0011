package repository

import (
	"context"
	"encoding/json"

	"experiment-trace/internal/domain"
)

// ExperimentFileRepo 实验文件仓库
type ExperimentFileRepo struct {
	store *FileStore
}

func NewExperimentFileRepo(store *FileStore) *ExperimentFileRepo {
	return &ExperimentFileRepo{store: store}
}

func (r *ExperimentFileRepo) Create(ctx context.Context, exp *domain.Experiment) error {
	return r.store.writeEntity(ctx, "experiments", exp.ID, exp, exp.Version)
}

func (r *ExperimentFileRepo) Get(ctx context.Context, id string) (*domain.Experiment, error) {
	var exp domain.Experiment
	if err := r.store.readEntity(ctx, "experiments", id, &exp); err != nil {
		return nil, err
	}
	return &exp, nil
}

// Update 以乐观锁提交实验更新：候选 Version 必须等于调用方读取到的基础版本
// （代次），仓储在 CAS 命中后将其递增为 current.Version+1 落盘；否则返回
// ErrVersionConflict，使同一代并发写入中仅一个成功。
func (r *ExperimentFileRepo) Update(ctx context.Context, exp *domain.Experiment) error {
	candidate := *exp
	next, err := r.store.writeExperimentCAS(ctx, &candidate)
	if err != nil {
		return err
	}
	*exp = *next
	return nil
}

func (r *ExperimentFileRepo) List(ctx context.Context, filter domain.ExperimentFilter) ([]*domain.Experiment, error) {
	filterFn := func(data []byte) bool {
		var exp domain.Experiment
		if err := json.Unmarshal(data, &exp); err != nil {
			return false
		}
		if filter.Status != "" && exp.Status != filter.Status {
			return false
		}
		if filter.NameLike != "" && !contains(exp.Name, filter.NameLike) {
			return false
		}
		return true
	}
	dataList, err := r.store.listEntities(ctx, "experiments", filterFn)
	if err != nil {
		return nil, err
	}
	var results []*domain.Experiment
	for _, data := range dataList {
		var exp domain.Experiment
		if err := json.Unmarshal(data, &exp); err != nil {
			return nil, err
		}
		results = append(results, &exp)
	}
	// 分页
	if filter.Offset > 0 {
		if filter.Offset >= len(results) {
			return []*domain.Experiment{}, nil
		}
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}
	return results, nil
}

func (r *ExperimentFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "experiments", id)
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
