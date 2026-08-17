package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"experiment-trace/internal/application"
	"experiment-trace/internal/audit"
	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
	"experiment-trace/internal/scheduler"
	"experiment-trace/internal/web"
)

func main() {
	selfCheck := flag.Bool("self-check", false, "run self check and exit")
	flag.Parse()

	if *selfCheck {
		if err := runSelfCheck(); err != nil {
			log.Fatalf("self-check failed: %v", err)
		}
		log.Println("self-check passed")
		return
	}

	// 初始化存储
	store, err := repository.NewFileStore("./data")
	if err != nil {
		log.Fatal(err)
	}
	// 创建各仓库实现
	expRepo := repository.NewExperimentFileRepo(store)
	paramRepo := repository.NewParamVersionFileRepo(store)
	inputRepo := repository.NewInputSnapshotFileRepo(store)
	planRepo := repository.NewRunPlanFileRepo(store)
	attemptRepo := repository.NewExecutionAttemptFileRepo(store)
	leaseRepo := repository.NewWorkerLeaseFileRepo(store)
	artifactRepo := repository.NewOutputArtifactFileRepo(store)
	lineageRepo := repository.NewLineageEdgeFileRepo(store)
	tagRepo := repository.NewReleaseTagFileRepo(store)
	budgetRepo := repository.NewResourceBudgetFileRepo(store)
	idemRepo := repository.NewIdempotencyFileRepo(store)

	// 创建应用服务
	svc := application.NewService(expRepo, paramRepo, inputRepo, planRepo, attemptRepo, leaseRepo, artifactRepo, lineageRepo, tagRepo, budgetRepo, idemRepo)

	// 审计器
	auditor, err := audit.NewAuditor("./data/audit")
	if err != nil {
		log.Fatal(err)
	}
	defer auditor.Close()

	// 启动调度器
	sch := scheduler.NewScheduler(svc, 3)
	sch.Start()
	defer sch.Stop()

	// HTTP服务器
	srv := web.NewServer(svc)
	handler := srv.Handler()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func runSelfCheck() error {
	// 简单自检：创建临时存储，创建实验，冻结，运行，查询
	dir, err := os.MkdirTemp("", "selfcheck-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	store, err := repository.NewFileStore(dir)
	if err != nil {
		return err
	}
	expRepo := repository.NewExperimentFileRepo(store)
	paramRepo := repository.NewParamVersionFileRepo(store)
	inputRepo := repository.NewInputSnapshotFileRepo(store)
	planRepo := repository.NewRunPlanFileRepo(store)
	attemptRepo := repository.NewExecutionAttemptFileRepo(store)
	leaseRepo := repository.NewWorkerLeaseFileRepo(store)
	artifactRepo := repository.NewOutputArtifactFileRepo(store)
	lineageRepo := repository.NewLineageEdgeFileRepo(store)
	tagRepo := repository.NewReleaseTagFileRepo(store)
	budgetRepo := repository.NewResourceBudgetFileRepo(store)
	idemRepo := repository.NewIdempotencyFileRepo(store)
	svc := application.NewService(expRepo, paramRepo, inputRepo, planRepo, attemptRepo, leaseRepo, artifactRepo, lineageRepo, tagRepo, budgetRepo, idemRepo)

	ctx := context.Background()
	exp, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{"a":1}`), []byte(`{"x":"y"}`), nil)
	if err != nil {
		return err
	}
	plan, err := svc.FreezeExperiment(ctx, exp.ID)
	if err != nil {
		return err
	}
	if err := svc.QueueRunPlan(ctx, plan.ID); err != nil {
		return err
	}
	// 模拟领取并完成
	_, _, err = svc.ClaimNextRunPlan(ctx, "worker1")
	if err != nil {
		return err
	}
	// 获取attempt ID
	attempts, err := svc.AttemptRepo().ListByRunPlan(ctx, plan.ID)
	if err != nil {
		return err
	}
	if len(attempts) == 0 {
		return domain.ErrNotFound
	}
	if err := svc.CompleteExecution(ctx, attempts[0].ID, true, []byte(`{"out":1}`), ""); err != nil {
		return err
	}
	if _, err := svc.PublishExperiment(ctx, exp.ID, "v1"); err != nil {
		return err
	}
	return nil
}
