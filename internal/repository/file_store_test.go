package repository

import (
	"context"
	"os"
	"testing"
)

func TestFileStoreBasic(t *testing.T) {
	dir, _ := os.MkdirTemp("", "store-*")
	defer os.RemoveAll(dir)
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	type dummy struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	d := dummy{ID: "d1", Version: 1}
	if err := store.writeEntity(context.Background(), "test", "d1", d, 1); err != nil {
		t.Fatal(err)
	}
	var out dummy
	if err := store.readEntity(context.Background(), "test", "d1", &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "d1" {
		t.Fatalf("expected d1, got %s", out.ID)
	}
}
