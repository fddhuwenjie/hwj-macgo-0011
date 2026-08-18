package application_test

import (
	"context"
	"errors"
	"testing"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
	"experiment-trace/internal/repository"
)

// 下面几个装饰器在真实文件仓储之上包一层，用于在测试中注入存储失败并观察已创建实体，
// 从而验证创建事务的回滚契约：任一存储步骤失败时，已创建的实体必须被回滚，
// 且函数必须返回错误（而不是返回一个未持久化的“成功”结果，让客户端误以为已创建）。

type recExperimentRepo struct {
	repository.ExperimentRepository
	created    []string
	failCreate error
}

func (r *recExperimentRepo) Create(ctx context.Context, exp *domain.Experiment) error {
	if r.failCreate != nil {
		return r.failCreate
	}
	if err := r.ExperimentRepository.Create(ctx, exp); err != nil {
		return err
	}
	r.created = append(r.created, exp.ID)
	return nil
}

type recParamRepo struct {
	repository.ParamVersionRepository
	created []string
}

func (r *recParamRepo) Create(ctx context.Context, pv *domain.ParamVersion) error {
	if err := r.ParamVersionRepository.Create(ctx, pv); err != nil {
		return err
	}
	r.created = append(r.created, pv.ID)
	return nil
}

type recInputRepo struct {
	repository.InputSnapshotRepository
	created []string
}

func (r *recInputRepo) Create(ctx context.Context, is *domain.InputSnapshot) error {
	if err := r.InputSnapshotRepository.Create(ctx, is); err != nil {
		return err
	}
	r.created = append(r.created, is.ID)
	return nil
}

type recBudgetRepo struct {
	repository.ResourceBudgetRepository
	created    []string
	failCreate error
}

func (r *recBudgetRepo) Create(ctx context.Context, rb *domain.ResourceBudget) error {
	if r.failCreate != nil {
		return r.failCreate
	}
	if err := r.ResourceBudgetRepository.Create(ctx, rb); err != nil {
		return err
	}
	r.created = append(r.created, rb.ID)
	return nil
}

type failIdemRepo struct {
	repository.IdempotencyRepository
	failCreate error
}

func (r *failIdemRepo) Create(ctx context.Context, ik *domain.IdempotencyKey) error {
	if r.failCreate != nil {
		return r.failCreate
	}
	return r.IdempotencyRepository.Create(ctx, ik)
}

// assertRolledBack 验证给定 ID 在底层仓储中已不存在（即已被回滚删除）。
func assertRolledBack(t *testing.T, ctx context.Context, kind, id string, get func(context.Context, string) error) {
	t.Helper()
	if id == "" {
		return
	}
	if err := get(ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("%s %s should have been rolled back, got err=%v", kind, id, err)
	}
}

// TestCreateExperimentFailsWhenExperimentStoreFails 回归 `&& false` 缺陷：
// 实验存储失败时必须返回错误，而不是吞掉错误返回一个未持久化的成功结果。
func TestCreateExperimentFailsWhenExperimentStoreFails(t *testing.T) {
	base, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	expFail := &recExperimentRepo{ExperimentRepository: base.ExpRepo(), failCreate: errors.New("injected experiment store failure")}
	svc := application.NewService(
		expFail, base.ParamRepo(), base.InputRepo(),
		base.PlanRepo(), base.AttemptRepo(), base.LeaseRepo(),
		base.ArtifactRepo(), base.LineageRepo(), base.TagRepo(),
		base.BudgetRepo(), base.IdemRepo(),
	)

	_, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error when experiment store fails; got nil (error was swallowed and a non-persisted success was returned)")
	}

	// 实验存储失败发生在最前，不应有任何实体被持久化。
	exps, err := base.ExpRepo().List(ctx, domain.ExperimentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(exps) != 0 {
		t.Fatalf("expected no experiments persisted, got %d", len(exps))
	}
}

// TestCreateExperimentRollsBackOnBudgetStoreFailure 验证预算存储失败时，
// 之前已创建的实验/参数版本/输入快照被回滚，不留下孤儿记录。
func TestCreateExperimentRollsBackOnBudgetStoreFailure(t *testing.T) {
	base, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	expRec := &recExperimentRepo{ExperimentRepository: base.ExpRepo()}
	paramRec := &recParamRepo{ParamVersionRepository: base.ParamRepo()}
	inputRec := &recInputRepo{InputSnapshotRepository: base.InputRepo()}
	budgetFail := &recBudgetRepo{ResourceBudgetRepository: base.BudgetRepo(), failCreate: errors.New("injected budget store failure")}
	svc := application.NewService(
		expRec, paramRec, inputRec,
		base.PlanRepo(), base.AttemptRepo(), base.LeaseRepo(),
		base.ArtifactRepo(), base.LineageRepo(), base.TagRepo(),
		budgetFail, base.IdemRepo(),
	)

	_, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error when budget store fails")
	}

	// 预算创建失败，不应被记录为已创建。
	if len(budgetFail.created) != 0 {
		t.Fatalf("budget should not have been created, recorded: %v", budgetFail.created)
	}
	// 已创建的实验/参数/输入必须被回滚。
	for _, id := range expRec.created {
		assertRolledBack(t, ctx, "experiment", id, func(ctx context.Context, id string) error {
			_, err := base.ExpRepo().Get(ctx, id)
			return err
		})
	}
	for _, id := range paramRec.created {
		assertRolledBack(t, ctx, "param version", id, func(ctx context.Context, id string) error {
			_, err := base.ParamRepo().Get(ctx, id)
			return err
		})
	}
	for _, id := range inputRec.created {
		assertRolledBack(t, ctx, "input snapshot", id, func(ctx context.Context, id string) error {
			_, err := base.InputRepo().Get(ctx, id)
			return err
		})
	}
}

// TestCreateExperimentRollsBackOnIdempotencyStoreFailure 验证幂等键记录失败时，
// 已创建的全部实体（含预算）被回滚，持久化结果与返回的错误保持一致。
func TestCreateExperimentRollsBackOnIdempotencyStoreFailure(t *testing.T) {
	base, cleanup := setupService(t)
	defer cleanup()
	ctx := context.Background()

	expRec := &recExperimentRepo{ExperimentRepository: base.ExpRepo()}
	paramRec := &recParamRepo{ParamVersionRepository: base.ParamRepo()}
	inputRec := &recInputRepo{InputSnapshotRepository: base.InputRepo()}
	budgetRec := &recBudgetRepo{ResourceBudgetRepository: base.BudgetRepo()}
	idemFail := &failIdemRepo{IdempotencyRepository: base.IdemRepo(), failCreate: errors.New("injected idempotency store failure")}
	svc := application.NewService(
		expRec, paramRec, inputRec,
		base.PlanRepo(), base.AttemptRepo(), base.LeaseRepo(),
		base.ArtifactRepo(), base.LineageRepo(), base.TagRepo(),
		budgetRec, idemFail,
	)

	_, err := svc.CreateExperiment(ctx, "test", "desc", []byte(`{}`), []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error when idempotency store fails")
	}

	// 全部四个实体在幂等键记录前都已创建，必须全部被回滚。
	for _, id := range expRec.created {
		assertRolledBack(t, ctx, "experiment", id, func(ctx context.Context, id string) error {
			_, err := base.ExpRepo().Get(ctx, id)
			return err
		})
	}
	for _, id := range paramRec.created {
		assertRolledBack(t, ctx, "param version", id, func(ctx context.Context, id string) error {
			_, err := base.ParamRepo().Get(ctx, id)
			return err
		})
	}
	for _, id := range inputRec.created {
		assertRolledBack(t, ctx, "input snapshot", id, func(ctx context.Context, id string) error {
			_, err := base.InputRepo().Get(ctx, id)
			return err
		})
	}
	for _, id := range budgetRec.created {
		assertRolledBack(t, ctx, "resource budget", id, func(ctx context.Context, id string) error {
			_, err := base.BudgetRepo().Get(ctx, id)
			return err
		})
	}
}
