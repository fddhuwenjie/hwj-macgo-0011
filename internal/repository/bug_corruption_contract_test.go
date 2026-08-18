package repository

import (
    "context"
    "os"
    "testing"
)

func TestCorruptEntityIsRejected(t *testing.T) {
    dir := t.TempDir()
    store, err := NewFileStore(dir)
    if err != nil { t.Fatal(err) }
    type value struct { ID string `json:"id"` }
    if err := store.writeEntity(context.Background(), "test", "x", value{ID: "x"}, 1); err != nil { t.Fatal(err) }
    path := dir + "/test/x.json"
    data, err := os.ReadFile(path)
    if err != nil { t.Fatal(err) }
    data[len(data)-1] ^= 1
    if err := os.WriteFile(path, data, 0644); err != nil { t.Fatal(err) }
    var got value
    if err := store.readEntity(context.Background(), "test", "x", &got); err == nil {
        t.Fatal("corrupt entity was accepted")
    }
}
