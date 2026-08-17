package journal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"experiment-trace/internal/persistence"
)

// Entry WAL条目
type Entry struct {
	Seq       int             `json:"seq"`
	Operation string          `json:"op"`
	EntityType string         `json:"entity_type"`
	EntityID  string          `json:"entity_id"`
	Data      json.RawMessage `json:"data"`
}

// WAL 写前日志
type WAL struct {
	mu      sync.Mutex
	file    *os.File
	seq     int
}

// NewWAL 创建WAL，打开wal.log
func NewWAL(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "wal.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	w := &WAL{file: f}
	// 恢复seq：读取现有日志确定最后一个seq
	// 简化：从文件大小估算？我们将在恢复时处理
	return w, nil
}

// Append 追加一条日志
func (w *WAL) Append(ctx context.Context, op, entityType, entityID string, data json.RawMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seq++
	entry := Entry{
		Seq:       w.seq,
		Operation: op,
		EntityType: entityType,
		EntityID:  entityID,
		Data:      data,
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// 使用持久化记录包装
	record := persistence.Record{Version: 1, Data: b}
	if err := persistence.WriteRecord(w.file, record); err != nil {
		return err
	}
	// 确保落盘
	if err := w.file.Sync(); err != nil {
		return err
	}
	return nil
}

// Close 关闭WAL
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
