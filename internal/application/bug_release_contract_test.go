package application_test

import (
	"context"
	"os"
	"testing"
	"time"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
	planquery "experiment-trace/internal/query"
)

func TestPlanHistoryAndPublishUseCompleteLatestPlan(t *testing.T) {
	svc, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	experiment := domain.NewExperiment("exp-release", "release", "multiple completed runs")
	experiment.Status = domain.ExperimentSealed
	if err := svc.ExpRepo().Create(ctx, experiment); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)
	plans := []*domain.RunPlan{
		sealedPlan("plan-old", experiment.ID, base, base.Add(time.Minute)),
		sealedPlan("plan-middle", experiment.ID, base.Add(time.Minute), base.Add(2*time.Minute)),
		sealedPlan("plan-new", experiment.ID, base.Add(2*time.Minute), base.Add(3*time.Minute)),
	}
	for _, plan := range plans {
		if err := svc.PlanRepo().Create(ctx, plan); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("history keeps every matching plan", func(t *testing.T) {
		summaries, err := planquery.NewQueryService(svc.PlanRepo()).GetRunPlanSummaries(
			ctx,
			domain.RunPlanFilter{ExperimentID: experiment.ID, Status: []domain.RunPlanStatus{domain.RunPlanSealed}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != len(plans) {
			t.Fatalf("expected %d completed plans, got %d", len(plans), len(summaries))
		}
	})

	t.Run("release binds the most recently completed plan", func(t *testing.T) {
		tag, err := svc.PublishExperiment(ctx, experiment.ID, "v2")
		if err != nil {
			t.Fatal(err)
		}
		if tag.RunPlanID != "plan-new" {
			t.Fatalf("release tag points to %q, want newest plan %q", tag.RunPlanID, "plan-new")
		}
	})
}

func sealedPlan(id, experimentID string, createdAt, completedAt time.Time) *domain.RunPlan {
	return &domain.RunPlan{
		ID:               id,
		ExperimentID:     experimentID,
		ParamVersionID:   "param-" + id,
		InputSnapshotID:  "input-" + id,
		Status:           domain.RunPlanSealed,
		Version:          1,
		CreatedAt:        createdAt,
		UpdatedAt:        completedAt,
		CompletedAt:      &completedAt,
		CurrentAttemptNo: 1,
	}
}

// setupServiceAtDir 在指定目录上构建一套完整的文件存储服务，用于模拟“重启文件存储”。
func setupServiceAtDir(t *testing.T, dir string) *application.Service {
	t.Helper()
	store, err := repository.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return application.NewService(
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
}

// TestReleaseSurvivesFileStoreRestart 验证“重启文件存储后发布与历史关系仍然稳定”。
// 重建文件存储后，历史不丢条，且发布仍绑定最近一次已封存的结果。
func TestReleaseSurvivesFileStoreRestart(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "test-release-restart-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// 先用一个文件存储写入实验与多次已封存的运行计划。
	svc := setupServiceAtDir(t, dir)
	experiment := domain.NewExperiment("exp-restart", "restart", "stable across restart")
	experiment.Status = domain.ExperimentSealed
	if err := svc.ExpRepo().Create(ctx, experiment); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, time.August, 18, 8, 0, 0, 0, time.UTC)
	// 使用与时间线不一致的 ID，确保选中结果不依赖文件名字典序。
	plans := []*domain.RunPlan{
		sealedPlan("z-old-plan", experiment.ID, base, base.Add(time.Minute)),
		sealedPlan("a-middle-plan", experiment.ID, base.Add(time.Minute), base.Add(2*time.Minute)),
		sealedPlan("m-new-plan", experiment.ID, base.Add(2*time.Minute), base.Add(3*time.Minute)),
	}
	for _, p := range plans {
		if err := svc.PlanRepo().Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	// 重启文件存储：在同一目录重建 store 与服务，仅依据已持久化数据。
	restarted := setupServiceAtDir(t, dir)

	// 1) 重启后历史查询仍保留全部符合条件的记录，不丢条。
	summaries, err := planquery.NewQueryService(restarted.PlanRepo()).GetRunPlanSummaries(
		ctx,
		domain.RunPlanFilter{ExperimentID: experiment.ID, Status: []domain.RunPlanStatus{domain.RunPlanSealed}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != len(plans) {
		t.Fatalf("after restart expected %d plans, got %d", len(plans), len(summaries))
	}

	// 2) 重启后发布仍稳定绑定最近一次已封存的结果。
	tag, err := restarted.PublishExperiment(ctx, experiment.ID, "v-restart")
	if err != nil {
		t.Fatal(err)
	}
	if tag.RunPlanID != "m-new-plan" {
		t.Fatalf("after restart release points to %q, want %q", tag.RunPlanID, "m-new-plan")
	}
}
