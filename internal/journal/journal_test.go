package journal

import (
	"context"
	"os"
	"testing"
)

func TestWALAppend(t *testing.T) {
	dir, _ := os.MkdirTemp("", "wal-*")
	defer os.RemoveAll(dir)
	w, err := NewWAL(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Append(context.Background(), "create", "test", "id1", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
}
