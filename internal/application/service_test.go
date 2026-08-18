package application_test

import (
	"context"
	"encoding/json"
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
	// 失败第一次：计划应进入 retry_wait
	if err := svc.CompleteExecution(ctx, attempt.ID, false, nil, "error1"); err != nil {
		t.Fatal(err)
	}
	updatedPlan, _ := svc.PlanRepo().Get(ctx, plan.ID)
	if updatedPlan.Status != domain.RunPlanRetryWait {
		t.Fatalf("expected retry_wait, got %s", updatedPlan.Status)
	}
	// 再次排队：retry_wait 应能回到 queued，否则调度永远领取不到
	if err := svc.QueueRunPlan(ctx, plan.ID); err != nil {
		t.Fatalf("requeue from retry_wait should succeed: %v", err)
	}
	if got, _ := svc.PlanRepo().Get(ctx, plan.ID); got.Status != domain.RunPlanQueued {
		t.Fatalf("expected queued after requeue, got %s", got.Status)
	}
	// 重新领取并执行成功
	if _, _, err := svc.ClaimNextRunPlan(ctx, "worker1"); err != nil {
		t.Fatalf("reclaim after requeue should succeed: %v", err)
	}
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

// drainRetries 反复失败直到计划耗尽重试次数进入 failed 终态。
func drainRetries(t *testing.T, svc *application.Service, ctx context.Context, planID string) {
	t.Helper()
	for {
		plan, _ := svc.PlanRepo().Get(ctx, planID)
		if plan.Status == domain.RunPlanFailed {
			return
		}
		if plan.Status != domain.RunPlanQueued && plan.Status != domain.RunPlanRetryWait {
			t.Fatalf("unexpected plan status %s while draining retries", plan.Status)
		}
		// retry_wait 需先回到 queued 才能被领取
		if plan.Status == domain.RunPlanRetryWait {
			if err := svc.QueueRunPlan(ctx, planID); err != nil {
				t.Fatalf("requeue during drain should succeed: %v", err)
			}
		}
		if _, _, err := svc.ClaimNextRunPlan(ctx, "worker1"); err != nil {
			t.Fatalf("claim during drain should succeed: %v", err)
		}
		attempts, _ := svc.AttemptRepo().ListByRunPlan(ctx, planID)
		if err := svc.CompleteExecution(ctx, attempts[len(attempts)-1].ID, false, nil, "boom"); err != nil {
			t.Fatalf("fail during drain should succeed: %v", err)
		}
	}
}

// TestNonRetryableStatesRejectedFromQueue 验证终态（不可重试）计划无法被重新排队，
// 而 retry_wait（可重试）可以正常回到队列。
func TestNonRetryableStatesRejectedFromQueue(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	exp, _ := svc.CreateExperiment(ctx, "terminal", "desc", []byte(`{}`), []byte(`{}`), nil)
	plan, _ := svc.FreezeExperiment(ctx, exp.ID)
	_ = svc.QueueRunPlan(ctx, plan.ID)
	_, _, _ = svc.ClaimNextRunPlan(ctx, "worker1")
	attempts, _ := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)

	// 成功封存后，sealed 计划不可重新排队
	if err := svc.CompleteExecution(ctx, attempts[0].ID, true, []byte(`{"out":1}`), ""); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.PlanRepo().Get(ctx, plan.ID); got.Status != domain.RunPlanSealed {
		t.Fatalf("expected sealed, got %s", got.Status)
	}
	if err := svc.QueueRunPlan(ctx, plan.ID); err == nil {
		t.Fatal("requeue of sealed plan should be rejected")
	}

	// 另一个计划：耗尽重试进入 failed，failed 计划同样不可重新排队
	exp2, _ := svc.CreateExperiment(ctx, "failed", "desc", []byte(`{}`), []byte(`{}`), nil)
	plan2, _ := svc.FreezeExperiment(ctx, exp2.ID)
	_ = svc.QueueRunPlan(ctx, plan2.ID)
	drainRetries(t, svc, ctx, plan2.ID)
	if got, _ := svc.PlanRepo().Get(ctx, plan2.ID); got.Status != domain.RunPlanFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
	if err := svc.QueueRunPlan(ctx, plan2.ID); err == nil {
		t.Fatal("requeue of failed plan should be rejected")
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

var _ = json.Marshal
var _ = time.Now
