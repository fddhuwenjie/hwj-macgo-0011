package query

import (
	"context"
	"testing"

	"experiment-trace/internal/domain"
)

type mockPlanRepo struct{}

func (m *mockPlanRepo) List(ctx context.Context, filter domain.RunPlanFilter) ([]*domain.RunPlan, error) {
	return []*domain.RunPlan{}, nil
}

func TestQueryService(t *testing.T) {
	q := NewQueryService(&mockPlanRepo{})
	plans, err := q.GetRunPlanSummaries(context.Background(), domain.RunPlanFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if plans == nil {
		t.Fatal("expected non-nil plans")
	}
}
