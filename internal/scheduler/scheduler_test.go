package scheduler_test

import (
	"context"
	"os"
	"testing"
	"time"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
	"experiment-trace/internal/scheduler"
)

// setupService 构造一个基于临时文件存储的 Service，与 application 测试保持一致。
func setupService(t *testing.T) (*application.Service, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "sched-test-*")
	if err != nil {
		t.Fatal(err)
	}
	store, err := repository.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	svc := application.NewService(
		repository.NewExperimentFileRepo(store),
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
	return svc, func() { os.RemoveAll(dir) }
}

// queueNPlans 创建并排队 N 个计划，每个预算 MaxRetries=1，使每个计划只跑一次尝试：
// 这样单次尝试成功率(≈70%)即等于计划成功率，便于稳定地观察到成功与失败并存。
func queueNPlans(t *testing.T, svc *application.Service, ctx context.Context, n int) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		// MaxRetries=1：失败即终态，不进入 retry_wait
		budget := domain.NewResourceBudget("", "", 1, 256, 1024, 1, 30*time.Minute)
		exp, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{"p":1}`), []byte(`{"i":2}`), budget)
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
		ids = append(ids, plan.ID)
	}
	return ids
}

// TestSchedulerDrivesPlansToTerminal 验证调度器把计划推进到终态，且：
//   - 成功率既非 0%(修复前的 <0 恒失败 bug) 也非 100%(sanity)；
//   - 计划状态与尝试状态一致(sealed<->succeeded, failed<->failed)；
//   - 没有计划卡在 claimed/executing/queued/retry_wait。
func TestSchedulerDrivesPlansToTerminal(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	const n = 20
	planIDs := queueNPlans(t, svc, ctx, n)

	// 单 worker：避免 ClaimNextRunPlan 非原子领取在多 worker 下的竞态(独立于本次修复)。
	sch := scheduler.NewScheduler(svc, 1)
	sch.Start()
	defer sch.Stop()

	deadline := time.Now().Add(60 * time.Second)
	allTerminal := false
	for time.Now().Before(deadline) {
		allTerminal = true
		for _, pid := range planIDs {
			p, err := svc.PlanRepo().Get(ctx, pid)
			if err != nil {
				t.Fatal(err)
			}
			if p.Status != domain.RunPlanSealed && p.Status != domain.RunPlanFailed {
				allTerminal = false
				break
			}
		}
		if allTerminal {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !allTerminal {
		t.Fatalf("not all plans reached terminal state within timeout")
	}

	sealed, failed := 0, 0
	for _, pid := range planIDs {
		p, err := svc.PlanRepo().Get(ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		atts, err := svc.AttemptRepo().ListByRunPlan(ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		if len(atts) == 0 {
			t.Fatalf("plan %s has no attempts", pid)
		}
		last := atts[len(atts)-1]
		switch p.Status {
		case domain.RunPlanSealed:
			sealed++
			if last.Status != domain.AttemptSucceeded {
				t.Errorf("plan %s is sealed but last attempt is %s", pid, last.Status)
			}
		case domain.RunPlanFailed:
			failed++
			if last.Status != domain.AttemptFailed {
				t.Errorf("plan %s is failed but last attempt is %s", pid, last.Status)
			}
		default:
			t.Errorf("plan %s stuck in non-terminal state %s", pid, p.Status)
		}
	}

	// 修复前 rand.Intn(10) < 0 导致 success 恒为 false，所有计划都会 failed。
	// 这里要求既有成功也有失败，锁死该回归。
	if sealed == 0 {
		t.Fatalf("expected some successes, got 0/%d (success rate regression?)", n)
	}
	if sealed == n {
		t.Fatalf("expected some failures, got %d/%d", sealed, n)
	}
	t.Logf("scheduler results: %d sealed, %d failed out of %d (%.0f%% success)",
		sealed, failed, n, float64(sealed)/float64(n)*100)
}
