package domain_test

import (
    "testing"
    "time"

    "experiment-trace/internal/domain"
)

func TestExpiredLeaseIsRecognized(t *testing.T) {
    lease := domain.NewWorkerLease("l", "w", "p", "a", time.Second)
    if !lease.IsExpired(time.Now().Add(2 * time.Second)) {
        t.Fatal("expired lease was treated as active")
    }
}
