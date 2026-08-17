package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
)

// Scheduler 后台执行器
type Scheduler struct {
	svc      *application.Service
	workers  int
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	workerID string
}

// NewScheduler 创建调度器
func NewScheduler(svc *application.Service, workers int) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		svc:      svc,
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
		workerID: "worker-" + randString(6),
	}
}

// Start 启动后台工作者
func (s *Scheduler) Start() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.workerLoop(i)
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

func (s *Scheduler) workerLoop(id int) {
	defer s.wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.processNext(s.ctx); err != nil {
				if !errors.Is(err, domain.ErrNotFound) {
					log.Printf("worker %d error: %v", id, err)
				}
			}
		}
	}
}

func (s *Scheduler) processNext(ctx context.Context) error {
	// 领取下一个任务
	plan, _, err := s.svc.ClaimNextRunPlan(ctx, s.workerID)
	if err != nil {
		return err
	}
	// 执行任务（模拟可能失败）
	attemptID := ""
	// 从plan中获取当前尝试ID无法直接获取，需要查找最新尝试
	// 简化：直接执行并立即完成
	// 根据一定概率模拟成功/失败
	success := rand.Intn(10) < 7 // 70%成功
	var outputContent json.RawMessage
	if success {
		outputContent = json.RawMessage(`{"result":"success"}`)
	} else {
		outputContent = nil
	}
	// 找到attempt ID
	attempts, err := s.svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	if err != nil {
		return err
	}
	if len(attempts) == 0 {
		return domain.ErrNotFound
	}
	attemptID = attempts[len(attempts)-1].ID
	// 应用超时和取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(rand.Intn(500)) * time.Millisecond):
	}
	// 完成执行
	return s.svc.CompleteExecution(ctx, attemptID, success, outputContent, "simulated failure")
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
