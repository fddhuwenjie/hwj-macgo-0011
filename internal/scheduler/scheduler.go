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
	// 领取下一个任务，直接拿到本次执行尝试，无需回查尝试列表
	_, attempt, _, err := s.svc.ClaimNextRunPlan(ctx, s.workerID)
	if err != nil {
		return err
	}
	// 执行任务（模拟可能失败）
	// rand.Intn(10) 落在 [0,10)，<7 命中概率 70%，与注释一致；
	// 此前写成 <0 恒为 false，导致所有任务都被当成失败。
	success := rand.Intn(10) < 7 // 70%成功
	var outputContent json.RawMessage
	if success {
		outputContent = json.RawMessage(`{"result":"success"}`)
	} else {
		outputContent = nil
	}
	// 应用超时和取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(rand.Intn(500)) * time.Millisecond):
	}
	// 完成执行：将本次调度结果（成功/失败）传递给应用层，由其推进
	// 尝试与计划状态。计划已在 ClaimNextRunPlan 中推进到 executing，
	// 因此成功走 Succeed、失败走 FailWithRetry/FailFinal 均可正常迁移。
	return s.svc.CompleteExecution(ctx, attempt.ID, success, outputContent, "simulated failure")
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
