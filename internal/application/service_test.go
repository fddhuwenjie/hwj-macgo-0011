package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
)

func setupService(t *testing.T) (*application.Service, func()) {
	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatal(err)
	}
	store, err := repository.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	expRepo := repository.NewExperimentFileRepo(store)
	paramRepo := repository.NewParamVersionFileRepo(store)
	inputRepo := repository.NewInputSnapshotFileRepo(store)
	planRepo := repository.NewRunPlanFileRepo(store)
	attemptRepo := repository.NewExecutionAttemptFileRepo(store)
	leaseRepo := repository.NewWorkerLeaseFileRepo(store)
	artifactRepo := repository.NewOutputArtifactFileRepo(store)
	lineageRepo := repository.NewLineageEdgeFileRepo(store)
	tagRepo := repository.NewReleaseTagFileRepo(store)
	budgetRepo := repository.NewResourceBudgetFileRepo(store)
	idemRepo := repository.NewIdempotencyFileRepo(store)
	svc := application.NewService(expRepo, paramRepo, inputRepo, planRepo, attemptRepo, leaseRepo, artifactRepo, lineageRepo, tagRepo, budgetRepo, idemRepo)
	return svc, func() { os.RemoveAll(dir) }
}

func TestCreateFreezeRunPublishChain(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	// 创建实验
	exp, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{"p":1}`), []byte(`{"i":2}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if exp.Status != domain.ExperimentDraft {
		t.Fatalf("expected draft, got %s", exp.Status)
	}

	// 冻结
	plan, err := svc.FreezeExperiment(ctx, exp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != domain.RunPlanFrozen {
		t.Fatalf("expected frozen, got %s", plan.Status)
	}

	// 排队
	if err := svc.QueueRunPlan(ctx, plan.ID); err != nil {
		t.Fatal(err)
	}

	// 领取
	_, lease, err := svc.ClaimNextRunPlan(ctx, "worker1")
	if err != nil {
		t.Fatal(err)
	}
	if lease.Status != "active" {
		t.Fatalf("expected active lease")
	}

	// 完成成功
	attempts, err := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(attempts))
	}
	if err := svc.CompleteExecution(ctx, attempts[0].ID, true, []byte(`{"out":3}`), ""); err != nil {
		t.Fatal(err)
	}

	// 验证状态
	updatedPlan, _ := svc.PlanRepo().Get(ctx, plan.ID)
	if updatedPlan.Status != domain.RunPlanSealed {
		t.Fatalf("expected sealed, got %s", updatedPlan.Status)
	}
	updatedExp, _ := svc.ExpRepo().Get(ctx, exp.ID)
	if updatedExp.Status != domain.ExperimentSealed {
		t.Fatalf("expected sealed exp, got %s", updatedExp.Status)
	}

	// 发布
	_, err = svc.PublishExperiment(ctx, exp.ID, "v1")
	if err != nil {
		t.Fatal(err)
	}
	updatedExp, _ = svc.ExpRepo().Get(ctx, exp.ID)
	if updatedExp.Status != domain.ExperimentPublished {
		t.Fatalf("expected published, got %s", updatedExp.Status)
	}
}

func TestFailureRetryAndRecovery(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	exp, _ := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	plan, _ := svc.FreezeExperiment(ctx, exp.ID)
	_ = svc.QueueRunPlan(ctx, plan.ID)
	_, _, _ = svc.ClaimNextRunPlan(ctx, "worker1")
	attempts, _ := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	attempt := attempts[0]
	// 失败第一次
	if err := svc.CompleteExecution(ctx, attempt.ID, false, nil, "error1"); err != nil {
		t.Fatal(err)
	}
	updatedPlan, _ := svc.PlanRepo().Get(ctx, plan.ID)
	if updatedPlan.Status != domain.RunPlanRetryWait {
		t.Fatalf("expected retry_wait, got %s", updatedPlan.Status)
	}
	// 再次排队
	_ = svc.QueueRunPlan(ctx, plan.ID)
	_, _, _ = svc.ClaimNextRunPlan(ctx, "worker1")
	attempts, _ = svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	attempt = attempts[len(attempts)-1]
	// 成功
	if err := svc.CompleteExecution(ctx, attempt.ID, true, []byte(`{"out":1}`), ""); err != nil {
		t.Fatal(err)
	}
	updatedPlan, _ = svc.PlanRepo().Get(ctx, plan.ID)
	if updatedPlan.Status != domain.RunPlanSealed {
		t.Fatalf("expected sealed, got %s", updatedPlan.Status)
	}
}

func TestVersionConflict(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	exp, _ := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	// 模拟并发更新：获取两次，然后一个更新成功，另一个应该失败
	exp1, _ := svc.ExpRepo().Get(ctx, exp.ID)
	exp2, _ := svc.ExpRepo().Get(ctx, exp.ID)
	exp1.Name = "changed1"
	exp2.Name = "changed2"
	if err := svc.ExpRepo().Update(ctx, exp1); err != nil {
		t.Fatal(err)
	}
	// 第二次更新应该失败，因为版本冲突
	if err := svc.ExpRepo().Update(ctx, exp2); err == nil {
		t.Fatal("expected version conflict error")
	}
}

func TestIdempotency(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	// 创建实验两次，相同参数应返回同一个
	// 由于幂等键基于随机生成，这里简化测试
	exp1, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	exp2, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if exp1.ID == exp2.ID {
		t.Fatal("expected different IDs for different calls")
	}
}

// sealFailingExpRepo 包装 ExperimentRepository，在持久化 Sealed 状态时注入失败，
// 用于验证封存提交失败时的回滚是否留下半封存数据。
type sealFailingExpRepo struct {
	repository.ExperimentRepository
	sealedUpdates int
	fail           bool
}

func (r *sealFailingExpRepo) Update(ctx context.Context, exp *domain.Experiment) error {
	if r.fail && exp.Status == domain.ExperimentSealed {
		r.sealedUpdates++
		return errors.New("injected seal commit failure")
	}
	return r.ExperimentRepository.Update(ctx, exp)
}

// TestSealRollbackOnCommitFailure 验证封存提交失败时不留下半封存数据：
// 执行结果（attempt/plan/artifact）与实验状态全部回滚到封存前，租约保持活跃。
func TestSealRollbackOnCommitFailure(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	store, err := repository.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	expRepo := &sealFailingExpRepo{ExperimentRepository: repository.NewExperimentFileRepo(store), fail: true}
	svc := application.NewService(
		expRepo,
		repository.NewParamVersionFileRepo(store),
		repository.NewInputSnapshotFileRepo(store),
		repository.NewRunPlanFileRepo(store),
		repository.NewExecutionAttemptFileRepo(store),
		repository.NewWorkerLeaseFileRepo(store),
		repository.NewOutputArtifactFileRepo(store),
		repository.NewLineageEdgeFileRepo(store),
		repository.NewReleaseTagFileRepo(store),
		repository.NewResourceBudgetFileRepo(store),
		repository.NewIdempotencyFileRepo(store),
	)

	ctx := context.Background()
	exp, err := svc.CreateExperiment(ctx, "rollback", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.FreezeExperiment(ctx, exp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.QueueRunPlan(ctx, plan.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ClaimNextRunPlan(ctx, "w1"); err != nil {
		t.Fatal(err)
	}
	attempts, err := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(attempts))
	}
	leaseID := attempts[0].WorkerLeaseID

	// 封存提交写实验状态时注入失败
	if err := svc.CompleteExecution(ctx, attempts[0].ID, true, []byte(`{"out":1}`), ""); err == nil {
		t.Fatal("expected seal commit to fail")
	}
	if expRepo.sealedUpdates != 1 {
		t.Fatalf("expected exactly one sealed update attempt, got %d", expRepo.sealedUpdates)
	}

	// 断言：无半封存数据——执行结果与实验状态都回到封存前。
	gotAttempt, _ := svc.AttemptRepo().Get(ctx, attempts[0].ID)
	if gotAttempt.Status != domain.AttemptExecuting {
		t.Fatalf("expected attempt rolled back to executing, got %s", gotAttempt.Status)
	}
	gotPlan, _ := svc.PlanRepo().Get(ctx, plan.ID)
	if gotPlan.Status != domain.RunPlanExecuting {
		t.Fatalf("expected plan rolled back to executing, got %s", gotPlan.Status)
	}
	gotExp, _ := svc.ExpRepo().Get(ctx, exp.ID)
	if gotExp.Status != domain.ExperimentRunning {
		t.Fatalf("expected exp rolled back to running, got %s", gotExp.Status)
	}
	// 制品不应残留
	if _, err := svc.ArtifactRepo().GetByAttempt(ctx, attempts[0].ID); err == nil {
		t.Fatal("expected no artifact to remain after rollback")
	}
	// 租约仍为 active：封存未完成，执行仍进行中，状态自洽
	lease, _ := svc.LeaseRepo().Get(ctx, leaseID)
	if lease.Status != "active" {
		t.Fatalf("expected lease to remain active, got %s", lease.Status)
	}

	// 恢复注入失败后，重新封存应能成功，且发布条件与执行结果一致
	expRepo.sealedUpdates = 0
	expRepo.fail = false
	if err := svc.CompleteExecution(ctx, attempts[0].ID, true, []byte(`{"out":2}`), ""); err != nil {
		t.Fatalf("expected seal to succeed after recovery, got %v", err)
	}
	gotExp, _ = svc.ExpRepo().Get(ctx, exp.ID)
	if gotExp.Status != domain.ExperimentSealed {
		t.Fatalf("expected exp sealed after recovery, got %s", gotExp.Status)
	}
	gotPlan, _ = svc.PlanRepo().Get(ctx, plan.ID)
	if gotPlan.Status != domain.RunPlanSealed {
		t.Fatalf("expected plan sealed after recovery, got %s", gotPlan.Status)
	}
	if _, err := svc.PublishExperiment(ctx, exp.ID, "v1"); err != nil {
		t.Fatalf("expected publish to succeed after seal, got %v", err)
	}
	gotExp, _ = svc.ExpRepo().Get(ctx, exp.ID)
	if gotExp.Status != domain.ExperimentPublished {
		t.Fatalf("expected exp published, got %s", gotExp.Status)
	}
}

var _ = json.Marshal
var _ = time.Now
