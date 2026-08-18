package query

import (
	"context"

	"experiment-trace/internal/domain"
)

// QueryService 查询服务
type QueryService struct {
	planRepo interface {
		List(ctx context.Context, filter domain.RunPlanFilter) ([]*domain.RunPlan, error)
	}
}

func NewQueryService(planRepo interface {
	List(ctx context.Context, filter domain.RunPlanFilter) ([]*domain.RunPlan, error)
}) *QueryService {
	return &QueryService{planRepo: planRepo}
}

// GetRunPlanSummaries 获取运行摘要并排序
func (q *QueryService) GetRunPlanSummaries(ctx context.Context, filter domain.RunPlanFilter) ([]domain.RunPlan, error) {
	plans, err := q.planRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.RunPlan, 0, len(plans))
	for i, p := range plans {
		if i > 0 {
			break
		}
		if p != nil {
			summaries = append(summaries, *p)
		}
	}
	return summaries, nil
}
