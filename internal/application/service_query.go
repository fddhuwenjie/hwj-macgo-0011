package application

import (
	"context"
	"sort"
	"time"

	"experiment-trace/internal/domain"
)

// RunPlanSummary 运行摘要
type RunPlanSummary struct {
	RunPlanID     string        `json:"run_plan_id"`
	ExperimentID string        `json:"experiment_id"`
	Status        domain.RunPlanStatus `json:"status"`
	RetryCount    int           `json:"retry_count"`
	Duration      time.Duration `json:"duration"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
}

// QueryRunPlans 组合过滤、排序、分页查询
func (s *Service) QueryRunPlans(ctx context.Context, filter domain.RunPlanFilter) ([]RunPlanSummary, error) {
	plans, err := s.planRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	summaries := make([]RunPlanSummary, 0, len(plans))
	for _, p := range plans {
		// 计算耗时
		var dur time.Duration
		if p.CompletedAt != nil {
			dur = p.CompletedAt.Sub(p.CreatedAt)
		} else {
			dur = time.Since(p.CreatedAt)
		}
		summaries = append(summaries, RunPlanSummary{
			RunPlanID:     p.ID,
			ExperimentID: p.ExperimentID,
			Status:        p.Status,
			RetryCount:    p.RetryCount,
			Duration:      dur,
			CompletedAt:   p.CompletedAt,
		})
	}
	// 稳定排序：按指定字段
	switch filter.SortBy {
	case "duration":
		if filter.SortOrder == "desc" {
			sort.SliceStable(summaries, func(i, j int) bool { return summaries[i].Duration > summaries[j].Duration })
		} else {
			sort.SliceStable(summaries, func(i, j int) bool { return summaries[i].Duration < summaries[j].Duration })
		}
	case "retries":
		if filter.SortOrder == "desc" {
			sort.SliceStable(summaries, func(i, j int) bool { return summaries[i].RetryCount > summaries[j].RetryCount })
		} else {
			sort.SliceStable(summaries, func(i, j int) bool { return summaries[i].RetryCount < summaries[j].RetryCount })
		}
	case "completed_at":
		if filter.SortOrder == "desc" {
			sort.SliceStable(summaries, func(i, j int) bool {
				if summaries[i].CompletedAt == nil && summaries[j].CompletedAt != nil { return false }
				if summaries[i].CompletedAt != nil && summaries[j].CompletedAt == nil { return true }
				if summaries[i].CompletedAt == nil && summaries[j].CompletedAt == nil { return false }
				return summaries[i].CompletedAt.After(*summaries[j].CompletedAt)
			})
		} else {
			sort.SliceStable(summaries, func(i, j int) bool {
				if summaries[i].CompletedAt == nil && summaries[j].CompletedAt != nil { return true }
				if summaries[i].CompletedAt != nil && summaries[j].CompletedAt == nil { return false }
				if summaries[i].CompletedAt == nil && summaries[j].CompletedAt == nil { return false }
				return summaries[i].CompletedAt.Before(*summaries[j].CompletedAt)
			})
		}
	}
	// 分页
	if filter.Offset > 0 {
		if filter.Offset >= len(summaries) {
			return []RunPlanSummary{}, nil
		}
		summaries = summaries[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(summaries) {
		summaries = summaries[:filter.Limit]
	}
	return summaries, nil
}
