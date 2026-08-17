package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Auditor 审计日志记录器
type Auditor struct {
	mu   sync.Mutex
	file *os.File
}

// NewAuditor 创建审计器，写入指定文件
func NewAuditor(dir string) (*Auditor, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "audit.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Auditor{file: f}, nil
}

// Record 记录一条审计事件
func (a *Auditor) Record(ctx context.Context, eventType, entityType, entityID, message string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	event := struct {
		Timestamp  time.Time `json:"timestamp"`
		EventType  string    `json:"event_type"`
		EntityType string    `json:"entity_type"`
		EntityID   string    `json:"entity_id"`
		Message    string    `json:"message"`
	}{
		Timestamp:  time.Now().UTC(),
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Message:    message,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = a.file.Write(append(data, '\n'))
	return err
}

// Close 关闭审计文件
func (a *Auditor) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}
