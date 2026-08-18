package application_test

import (
	"context"
	"testing"
	"time"

	"experiment-trace/internal/domain"
)

func TestRetryBudgetSurvivesRecoveryAndStopsAtBoundary(t *testing.T) {
	t.Run("persisted budget is copied into the plan unchanged", func(t *testing.T) {
		svc, cleanup := setupService(t)
		defer cleanup()
		ctx := context.Background()
		budget := domain.NewResourceBudget("", "", 1, 256, 1024, 1, time.Minute)
		experiment, err := svc.CreateExperiment(ctx, "budget", "restore boundary", []byte(`{}`), []byte(`{}`), budget)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := svc.FreezeExperiment(ctx, experiment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if plan.MaxRetries != 1 {
			t.Fatalf("restored retry budget=%d, want configured value 1", plan.MaxRetries)
		}
	})

	t.Run("the final allowed attempt enters terminal failure", func(t *testing.T) {
		svc, cleanup := setupService(t)
		defer cleanup()
		ctx := context.Background()
		budget := domain.NewResourceBudget("", "", 1, 256, 1024, 1, time.Minute)
		experiment, err := svc.CreateExperiment(ctx, "terminal", "retry boundary", []byte(`{}`), []byte(`{}`), budget)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := svc.FreezeExperiment(ctx, experiment.ID)
		if err != nil {
			t.Fatal(err)
		}
		plan.MaxRetries = 1
		if err := svc.PlanRepo().Update(ctx, plan); err != nil {
			t.Fatal(err)
		}
		if err := svc.QueueRunPlan(ctx, plan.ID); err != nil {
			t.Fatal(err)
		}
		if _, _, err := svc.ClaimNextRunPlan(ctx, "worker-boundary"); err != nil {
			t.Fatal(err)
		}
		attempts, err := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
		if err != nil || len(attempts) != 1 {
			t.Fatalf("expected one claimed attempt, got len=%d err=%v", len(attempts), err)
		}
		if err := svc.CompleteExecution(ctx, attempts[0].ID, false, nil, "terminal failure"); err != nil {
			t.Fatal(err)
		}
		gotPlan, err := svc.PlanRepo().Get(ctx, plan.ID)
		if err != nil {
			t.Fatal(err)
		}
		if gotPlan.Status != domain.RunPlanFailed || gotPlan.CompletedAt == nil {
			t.Fatalf("exhausted plan status=%s completed_at=%v, want terminal failed", gotPlan.Status, gotPlan.CompletedAt)
		}
		gotExperiment, err := svc.ExpRepo().Get(ctx, experiment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if gotExperiment.Status != domain.ExperimentFailed {
			t.Fatalf("experiment status=%s, want failed after retry budget is exhausted", gotExperiment.Status)
		}
	})
}
