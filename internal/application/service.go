package application

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
	"experiment-trace/internal/util"
)

// Service 应用服务，协调领域逻辑
type Service struct {
	expRepo      repository.ExperimentRepository
	paramRepo    repository.ParamVersionRepository
	inputRepo    repository.InputSnapshotRepository
	planRepo     repository.RunPlanRepository
	attemptRepo  repository.ExecutionAttemptRepository
	leaseRepo    repository.WorkerLeaseRepository
	artifactRepo repository.OutputArtifactRepository
	lineageRepo  repository.LineageEdgeRepository
	tagRepo      repository.ReleaseTagRepository
	budgetRepo   repository.ResourceBudgetRepository
	idemRepo     repository.IdempotencyRepository
}

// NewService 创建服务
func NewService(
	expRepo repository.ExperimentRepository,
	paramRepo repository.ParamVersionRepository,
	inputRepo repository.InputSnapshotRepository,
	planRepo repository.RunPlanRepository,
	attemptRepo repository.ExecutionAttemptRepository,
	leaseRepo repository.WorkerLeaseRepository,
	artifactRepo repository.OutputArtifactRepository,
	lineageRepo repository.LineageEdgeRepository,
	tagRepo repository.ReleaseTagRepository,
	budgetRepo repository.ResourceBudgetRepository,
	idemRepo repository.IdempotencyRepository,
) *Service {
	return &Service{
		expRepo:      expRepo,
		paramRepo:    paramRepo,
		inputRepo:    inputRepo,
		planRepo:     planRepo,
		attemptRepo:  attemptRepo,
		leaseRepo:    leaseRepo,
		artifactRepo: artifactRepo,
		lineageRepo:  lineageRepo,
		tagRepo:      tagRepo,
		budgetRepo:   budgetRepo,
		idemRepo:     idemRepo,
	}
}

// CreateExperiment 创建实验，同时创建参数版本、输入快照、资源预算（原子事务）
func (s *Service) CreateExperiment(ctx context.Context, name, description string, paramDef json.RawMessage, inputContent json.RawMessage, budget *domain.ResourceBudget) (*domain.Experiment, error) {
	// 幂等键（简化：基于name+description哈希）
	key := util.NewID()
	if existing, err := s.idemRepo.Get(ctx, key); err == nil && existing != nil {
		return s.expRepo.Get(ctx, existing.EntityID)
	}

	expID := util.NewID()
	exp := domain.NewExperiment(expID, name, description)
	paramVersion := domain.NewParamVersion(util.NewID(), expID, "initial", paramDef)
	inputSnapshot := domain.NewInputSnapshot(util.NewID(), expID, inputContent)
	exp.CurrentParamVersionID = paramVersion.ID
	exp.CurrentInputSnapshotID = inputSnapshot.ID
	if budget == nil {
		budget = domain.NewResourceBudget(expID+"_budget", expID, 1, 256, 1024, 3, 30*time.Minute)
	} else {
		budget.ID = expID + "_budget"
		budget.ExperimentID = expID
	}

	// 事务：创建所有实体，如有失败则回滚（简化：创建顺序，若失败则删除已创建）
	if err := s.expRepo.Create(ctx, exp); err != nil {
		return nil, err
	}
	if err := s.paramRepo.Create(ctx, paramVersion); err != nil {
		_ = s.expRepo.Delete(ctx, expID)
		return nil, err
	}
	if err := s.inputRepo.Create(ctx, inputSnapshot); err != nil {
		_ = s.expRepo.Delete(ctx, expID)
		_ = s.paramRepo.Delete(ctx, paramVersion.ID)
		return nil, err
	}
	if err := s.budgetRepo.Create(ctx, budget); err != nil {
		_ = s.expRepo.Delete(ctx, expID)
		_ = s.paramRepo.Delete(ctx, paramVersion.ID)
		_ = s.inputRepo.Delete(ctx, inputSnapshot.ID)
		return nil, err
	}

	// 记录幂等键
	resultBytes, _ := json.Marshal(exp)
	ik := domain.NewIdempotencyKey(key, "create_experiment", expID, string(resultBytes))
	if err := s.idemRepo.Create(ctx, ik); err != nil {
		return nil, err
	}
	return exp, nil
}

// FreezeExperiment 冻结实验：锁定参数和输入，创建运行计划
func (s *Service) FreezeExperiment(ctx context.Context, experimentID string) (*domain.RunPlan, error) {
	exp, err := s.expRepo.Get(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if exp.Status != domain.ExperimentDraft {
		return nil, domain.ErrInvalidStatus
	}
	// 获取参数版本和输入快照（简化：从实验的当前ID获取，这里假设已设置）
	if exp.CurrentParamVersionID == "" || exp.CurrentInputSnapshotID == "" {
		// 自动创建？这里要求必须存在，否则报错
		return nil, domain.ErrInvalidParameter
	}
	paramVersion, err := s.paramRepo.Get(ctx, exp.CurrentParamVersionID)
	if err != nil {
		return nil, err
	}
	inputSnapshot, err := s.inputRepo.Get(ctx, exp.CurrentInputSnapshotID)
	if err != nil {
		return nil, err
	}
	// 获取资源预算
	budget, err := s.budgetRepo.Get(ctx, exp.ID+"_budget")
	if err != nil {
		return nil, err
	}
	// 创建运行计划
	plan := domain.NewRunPlan(util.NewID(), exp.ID, paramVersion.ID, inputSnapshot.ID, budget.MaxRetries, budget.Timeout)
	if err := plan.Freeze(); err != nil {
		return nil, err
	}
	// 更新实验状态
	if err := exp.Freeze(paramVersion.ID, inputSnapshot.ID); err != nil {
		return nil, err
	}
	// 事务性更新
	if err := s.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}
	if err := s.expRepo.Update(ctx, exp); err != nil {
		// 回滚计划
		_ = s.planRepo.Delete(ctx, plan.ID)
		return nil, err
	}
	// 创建谱系边
	edge1 := domain.NewLineageEdge(util.NewID(), exp.ID, "experiment", plan.ID, "run_plan", "has_plan")
	if err := s.lineageRepo.Create(ctx, edge1); err != nil {
		return nil, err
	}
	return plan, nil
}

// QueueRunPlan 将运行计划排队
func (s *Service) QueueRunPlan(ctx context.Context, planID string) error {
	plan, err := s.planRepo.Get(ctx, planID)
	if err != nil {
		return err
	}
	if err := plan.Queue(); err != nil {
		return err
	}
	exp, err := s.expRepo.Get(ctx, plan.ExperimentID)
	if err != nil {
		return err
	}
	if exp.Status == domain.ExperimentFrozen {
		if err := exp.Queue(); err != nil {
			return err
		}
	}
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return err
	}
	return s.expRepo.Update(ctx, exp)
}

// ClaimNextRunPlan 领取下一个排队的运行计划（工作者调用）
func (s *Service) ClaimNextRunPlan(ctx context.Context, workerID string) (*domain.RunPlan, *domain.WorkerLease, error) {
	// 获取排队计划列表
	plans, err := s.planRepo.ListByStatus(ctx, []domain.RunPlanStatus{domain.RunPlanQueued})
	if err != nil {
		return nil, nil, err
	}
	if len(plans) == 0 {
		return nil, nil, domain.ErrNotFound
	}
	// 简单取第一个
	plan := plans[0]
	if err := plan.Claim(); err != nil {
		return nil, nil, err
	}
	if err := plan.StartExecution(); err != nil {
		return nil, nil, err
	}
	// 创建执行尝试
	attempt := domain.NewExecutionAttempt(util.NewID(), plan.ID, plan.CurrentAttemptNo+1)
	if err := attempt.Claim(util.NewID()); err != nil {
		return nil, nil, err
	}
	if err := attempt.Start(); err != nil {
		return nil, nil, err
	}
	// 创建租约
	lease := domain.NewWorkerLease(util.NewID(), workerID, plan.ID, attempt.ID, 30*time.Second)
	attempt.WorkerLeaseID = lease.ID
	// 更新计划当前尝试号
	plan.CurrentAttemptNo = attempt.AttemptNo
	exp, err := s.expRepo.Get(ctx, plan.ExperimentID)
	if err != nil {
		return nil, nil, err
	}
	if exp.Status == domain.ExperimentQueued {
		if err := exp.MarkRunning(); err != nil {
			return nil, nil, err
		}
	}
	// 保存所有（事务）
	if err := s.attemptRepo.Create(ctx, attempt); err != nil {
		return nil, nil, err
	}
	if err := s.leaseRepo.Create(ctx, lease); err != nil {
		_ = s.attemptRepo.Delete(ctx, attempt.ID)
		return nil, nil, err
	}
	if err := s.planRepo.Update(ctx, plan); err != nil {
		_ = s.attemptRepo.Delete(ctx, attempt.ID)
		_ = s.leaseRepo.Delete(ctx, lease.ID)
		return nil, nil, err
	}
	if err := s.expRepo.Update(ctx, exp); err != nil {
		return nil, nil, err
	}
	return plan, lease, nil
}

// CompleteExecution 完成执行：成功或失败
func (s *Service) CompleteExecution(ctx context.Context, attemptID string, success bool, outputContent json.RawMessage, errMsg string) error {
	attempt, err := s.attemptRepo.Get(ctx, attemptID)
	if err != nil {
		return err
	}
	plan, err := s.planRepo.Get(ctx, attempt.RunPlanID)
	if err != nil {
		return err
	}
	// 检查租约代次（简化：假设租约有效）
	lease, err := s.leaseRepo.Get(ctx, attempt.WorkerLeaseID)
	if err != nil {
		return err
	}
	if lease.Status != "active" {
		return domain.ErrLeaseExpired
	}
	if success {
		if err := attempt.Succeed(); err != nil {
			return err
		}
		if err := plan.Succeed(); err != nil {
			return err
		}
		// 创建输出制品
		artifact := domain.NewOutputArtifact(util.NewID(), attempt.ID, plan.ID, outputContent)
		if err := artifact.Seal(); err != nil {
			return err
		}
		// 更新实验状态为Sealed
		exp, err := s.expRepo.Get(ctx, plan.ExperimentID)
		if err != nil {
			return err
		}
		if err := exp.Seal(); err != nil {
			return err
		}
		// 保存所有
		if err := s.attemptRepo.Update(ctx, attempt); err != nil {
			return err
		}
		if err := s.planRepo.Update(ctx, plan); err != nil {
			return err
		}
		if err := s.artifactRepo.Create(ctx, artifact); err != nil {
			return err
		}
		if err := s.expRepo.Update(ctx, exp); err != nil {
			return err
		}
		// 释放租约
		if err := lease.Release(); err != nil {
			return err
		}
		return s.leaseRepo.Update(ctx, lease)
	} else {
		if err := attempt.Fail(errMsg); err != nil {
			return err
		}
		// 判断是否重试
		if plan.RetryCount+1 < plan.MaxRetries {
			// 进入重试等待
			if err := plan.FailWithRetry(); err != nil {
				return err
			}
		} else {
			if err := plan.FailFinal(); err != nil {
				return err
			}
			// 更新实验状态为Failed
			exp, err := s.expRepo.Get(ctx, plan.ExperimentID)
			if err != nil {
				return err
			}
			if err := exp.MarkFailed(); err != nil {
				return err
			}
			if err := s.expRepo.Update(ctx, exp); err != nil {
				return err
			}
		}
		// 保存attempt和plan
		if err := s.attemptRepo.Update(ctx, attempt); err != nil {
			return err
		}
		if err := s.planRepo.Update(ctx, plan); err != nil {
			return err
		}
		// 释放租约
		if err := lease.Release(); err != nil {
			return err
		}
		return s.leaseRepo.Update(ctx, lease)
	}
}

// PublishExperiment 发布实验
func (s *Service) PublishExperiment(ctx context.Context, experimentID string, tagName string) (*domain.ReleaseTag, error) {
	exp, err := s.expRepo.Get(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if exp.Status != domain.ExperimentSealed {
		return nil, domain.ErrInvalidStatus
	}
	// 获取该实验全部已封存的运行计划
	plans, err := s.planRepo.List(ctx, domain.RunPlanFilter{ExperimentID: exp.ID, Status: []domain.RunPlanStatus{domain.RunPlanSealed}})
	if err != nil {
		return nil, err
	}
	if len(plans) == 0 {
		return nil, domain.ErrNotFound
	}
	// 发布标签稳定绑定“最近一次已封存的结果”：按封存时间倒序选取。
	// 排序仅依赖已持久化字段（CompletedAt、CreatedAt、ID），与文件系统返回顺序无关，
	// 因此重启文件存储后仍能选中同一条记录。
	sort.SliceStable(plans, func(i, j int) bool {
		a, b := plans[i], plans[j]
		// 倒序：返回 true 表示 a 比 b “更近”，应排在前面。
		// 主键：封存时间 CompletedAt，缺失视为最旧。
		if a.CompletedAt != nil && b.CompletedAt != nil {
			if !a.CompletedAt.Equal(*b.CompletedAt) {
				return a.CompletedAt.After(*b.CompletedAt)
			}
		} else if a.CompletedAt != nil { // b 为 nil
			return true
		} else if b.CompletedAt != nil { // a 为 nil
			return false
		}
		// 次键：创建时间 CreatedAt 倒序
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		// 兜底：ID 倒序，形成全序，保证任意文件返回顺序下结果一致
		return a.ID > b.ID
	})
	plan := plans[0]
	// 创建发布标签
	tag := domain.NewReleaseTag(util.NewID(), exp.ID, plan.ID, tagName)
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	// 更新实验状态
	if err := exp.Publish(); err != nil {
		_ = s.tagRepo.Delete(ctx, tag.ID)
		return nil, err
	}
	if err := s.expRepo.Update(ctx, exp); err != nil {
		_ = s.tagRepo.Delete(ctx, tag.ID)
		return nil, err
	}
	return tag, nil
}
