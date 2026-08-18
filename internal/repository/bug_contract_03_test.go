package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"experiment-trace/internal/domain"
)

func TestBug03ConcurrentExperimentCAS(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewExperimentFileRepo(store)
	original := domain.NewExperiment("exp-cas", "original", "concurrent update")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatal(err)
	}

	const writers = 24
	stale := make([]*domain.Experiment, writers)
	for i := range stale {
		stale[i], err = repo.Get(ctx, original.ID)
		if err != nil {
			t.Fatal(err)
		}
		stale[i].Name = fmt.Sprintf("writer-%02d", i)
	}

	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	var successes atomic.Int32
	var conflicts atomic.Int32
	errs := make(chan error, writers)
	ready.Add(writers)
	done.Add(writers)
	for _, candidate := range stale {
		candidate := candidate
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			err := repo.Update(ctx, candidate)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, domain.ErrVersionConflict):
				conflicts.Add(1)
			default:
				errs <- err
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("unexpected concurrent update error: %v", err)
	}
	if got := successes.Load(); got != 1 {
		t.Fatalf("concurrent stale writers reported %d successes, want exactly 1", got)
	}
	if got := conflicts.Load(); got != writers-1 {
		t.Fatalf("concurrent stale writers reported %d conflicts, want %d", got, writers-1)
	}

	reopenedStore, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := NewExperimentFileRepo(reopenedStore).Get(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Version != original.Version+1 {
		t.Fatalf("persisted version=%d, want %d after one committed update", persisted.Version, original.Version+1)
	}
}
