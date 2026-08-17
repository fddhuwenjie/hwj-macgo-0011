package audit

import (
	"context"
	"os"
	"testing"
)

func TestAuditRecord(t *testing.T) {
	dir, _ := os.MkdirTemp("", "audit-*")
	defer os.RemoveAll(dir)
	a, err := NewAuditor(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Record(context.Background(), "test", "entity", "id1", "hello"); err != nil {
		t.Fatal(err)
	}
}
