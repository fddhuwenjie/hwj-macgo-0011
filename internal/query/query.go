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

// GetRunPlanSummaries 返回全部符合条件的运行计划，不丢条。
// 顺序与底层仓库返回顺序一致（文件名字典序，同一份数据重启后仍一致）；
// 若调用方需要排序或分页，应使用应用层 QueryRunPlans。
func (q *QueryService) GetRunPlanSummaries(ctx context.Context, filter domain.RunPlanFilter) ([]domain.RunPlan, error) {
	plans, err := q.planRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.RunPlan, 0, len(plans))
	for _, p := range plans {
		if p != nil {
			summaries = append(summaries, *p)
		}
	}
	return summaries, nil
}
