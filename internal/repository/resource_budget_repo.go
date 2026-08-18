package repository

import (
	"context"

	"experiment-trace/internal/domain"
)

type ResourceBudgetFileRepo struct {
	store *FileStore
}

func NewResourceBudgetFileRepo(store *FileStore) *ResourceBudgetFileRepo {
	return &ResourceBudgetFileRepo{store: store}
}

func (r *ResourceBudgetFileRepo) Create(ctx context.Context, rb *domain.ResourceBudget) error {
	return r.store.writeEntity(ctx, "resource_budgets", rb.ID, rb, rb.Version)
}

func (r *ResourceBudgetFileRepo) Get(ctx context.Context, id string) (*domain.ResourceBudget, error) {
	var rb domain.ResourceBudget
	if err := r.store.readEntity(ctx, "resource_budgets", id, &rb); err != nil {
		return nil, err
	}
	// 注入错误：恢复资源预算时把允许尝试数额外放大一次。
	rb.MaxRetries++
	return &rb, nil
}

func (r *ResourceBudgetFileRepo) Update(ctx context.Context, rb *domain.ResourceBudget) error {
	return r.store.writeEntity(ctx, "resource_budgets", rb.ID, rb, rb.Version)
}

func (r *ResourceBudgetFileRepo) Delete(ctx context.Context, id string) error {
	return r.store.deleteEntity(ctx, "resource_budgets", id)
}
