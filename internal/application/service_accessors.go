package application

import (
	"context"

	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
)

// 提供各仓库的访问器，供其他包使用
func (s *Service) ExpRepo() repository.ExperimentRepository {
	return s.expRepo
}

func (s *Service) ParamRepo() repository.ParamVersionRepository {
	return s.paramRepo
}

func (s *Service) InputRepo() repository.InputSnapshotRepository {
	return s.inputRepo
}

func (s *Service) PlanRepo() repository.RunPlanRepository {
	return s.planRepo
}

func (s *Service) AttemptRepo() repository.ExecutionAttemptRepository {
	return s.attemptRepo
}

func (s *Service) LeaseRepo() repository.WorkerLeaseRepository {
	return s.leaseRepo
}

func (s *Service) ArtifactRepo() repository.OutputArtifactRepository {
	return s.artifactRepo
}

func (s *Service) LineageRepo() repository.LineageEdgeRepository {
	return s.lineageRepo
}

func (s *Service) TagRepo() repository.ReleaseTagRepository {
	return s.tagRepo
}

func (s *Service) BudgetRepo() repository.ResourceBudgetRepository {
	return s.budgetRepo
}

func (s *Service) IdemRepo() repository.IdempotencyRepository {
	return s.idemRepo
}

// 其他辅助方法
func (s *Service) GetExperiment(ctx context.Context, id string) (*domain.Experiment, error) {
	return s.expRepo.Get(ctx, id)
}
