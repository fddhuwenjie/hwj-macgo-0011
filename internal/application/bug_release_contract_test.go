package application_test

import (
	"context"
	"testing"
	"time"

	"experiment-trace/internal/domain"
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
