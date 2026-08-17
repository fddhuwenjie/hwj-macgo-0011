package repository

import (
	"context"
	"experiment-trace/internal/domain"
)

// ExperimentRepository 实验仓库接口
type ExperimentRepository interface {
	Create(ctx context.Context, exp *domain.Experiment) error
	Get(ctx context.Context, id string) (*domain.Experiment, error)
	Update(ctx context.Context, exp *domain.Experiment) error
	List(ctx context.Context, filter domain.ExperimentFilter) ([]*domain.Experiment, error)
	Delete(ctx context.Context, id string) error
}

// ParamVersionRepository 参数版本仓库
type ParamVersionRepository interface {
	Create(ctx context.Context, pv *domain.ParamVersion) error
	Get(ctx context.Context, id string) (*domain.ParamVersion, error)
	Update(ctx context.Context, pv *domain.ParamVersion) error
	Delete(ctx context.Context, id string) error
}

// InputSnapshotRepository 输入快照仓库
type InputSnapshotRepository interface {
	Create(ctx context.Context, is *domain.InputSnapshot) error
	Get(ctx context.Context, id string) (*domain.InputSnapshot, error)
	Update(ctx context.Context, is *domain.InputSnapshot) error
	Delete(ctx context.Context, id string) error
}

// RunPlanRepository 运行计划仓库
type RunPlanRepository interface {
	Create(ctx context.Context, rp *domain.RunPlan) error
	Get(ctx context.Context, id string) (*domain.RunPlan, error)
	Update(ctx context.Context, rp *domain.RunPlan) error
	ListByStatus(ctx context.Context, statuses []domain.RunPlanStatus) ([]*domain.RunPlan, error)
	List(ctx context.Context, filter domain.RunPlanFilter) ([]*domain.RunPlan, error)
	Delete(ctx context.Context, id string) error
}

// ExecutionAttemptRepository 执行尝试仓库
type ExecutionAttemptRepository interface {
	Create(ctx context.Context, ea *domain.ExecutionAttempt) error
	Get(ctx context.Context, id string) (*domain.ExecutionAttempt, error)
	Update(ctx context.Context, ea *domain.ExecutionAttempt) error
	ListByRunPlan(ctx context.Context, runPlanID string) ([]*domain.ExecutionAttempt, error)
	Delete(ctx context.Context, id string) error
}

// WorkerLeaseRepository 工作者租约仓库
type WorkerLeaseRepository interface {
	Create(ctx context.Context, wl *domain.WorkerLease) error
	Get(ctx context.Context, id string) (*domain.WorkerLease, error)
	Update(ctx context.Context, wl *domain.WorkerLease) error
	ListActive(ctx context.Context) ([]*domain.WorkerLease, error)
	Delete(ctx context.Context, id string) error
}

// OutputArtifactRepository 输出制品仓库
type OutputArtifactRepository interface {
	Create(ctx context.Context, oa *domain.OutputArtifact) error
	Get(ctx context.Context, id string) (*domain.OutputArtifact, error)
	Update(ctx context.Context, oa *domain.OutputArtifact) error
	GetByAttempt(ctx context.Context, attemptID string) (*domain.OutputArtifact, error)
}

// LineageEdgeRepository 谱系边仓库
type LineageEdgeRepository interface {
	Create(ctx context.Context, le *domain.LineageEdge) error
	Get(ctx context.Context, id string) (*domain.LineageEdge, error)
	ListByFrom(ctx context.Context, fromID string) ([]*domain.LineageEdge, error)
	ListByTo(ctx context.Context, toID string) ([]*domain.LineageEdge, error)
}

// ReleaseTagRepository 发布标签仓库
type ReleaseTagRepository interface {
	Create(ctx context.Context, rt *domain.ReleaseTag) error
	Get(ctx context.Context, id string) (*domain.ReleaseTag, error)
	Update(ctx context.Context, rt *domain.ReleaseTag) error
	ListByExperiment(ctx context.Context, experimentID string) ([]*domain.ReleaseTag, error)
	Delete(ctx context.Context, id string) error
}

// ResourceBudgetRepository 资源预算仓库
type ResourceBudgetRepository interface {
	Create(ctx context.Context, rb *domain.ResourceBudget) error
	Get(ctx context.Context, id string) (*domain.ResourceBudget, error)
	Update(ctx context.Context, rb *domain.ResourceBudget) error
	Delete(ctx context.Context, id string) error
}

// IdempotencyRepository 幂等键仓库
type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*domain.IdempotencyKey, error)
	Create(ctx context.Context, ik *domain.IdempotencyKey) error
}
