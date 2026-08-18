package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"experiment-trace/internal/domain"
	"experiment-trace/internal/persistence"
)

// FileStore 基于文件的存储，每个实体类型一个子目录
type FileStore struct {
	mu      sync.RWMutex
	baseDir string
}

// NewFileStore 创建文件存储
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	// 创建各实体子目录
	dirs := []string{
		"experiments", "param_versions", "input_snapshots", "run_plans",
		"execution_attempts", "worker_leases", "output_artifacts", "lineage_edges",
		"release_tags", "resource_budgets", "idempotency_keys",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, d), 0755); err != nil {
			return nil, err
		}
	}
	return &FileStore{baseDir: baseDir}, nil
}

func (s *FileStore) entityDir(entityType string) string {
	return filepath.Join(s.baseDir, entityType)
}

func (s *FileStore) writeEntity(ctx context.Context, entityType, id string, obj interface{}, expectedVersion int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// writeEntity 不执行乐观锁校验，仅做原子持久化写入；
	// 实体的乐观并发控制由 writeExperimentCAS 负责。
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	// 添加元数据：版本、校验和、长度
	record := persistence.Record{
		Version: 0,
		Data:    data,
	}
	fullData, err := persistence.EncodeRecord(record)
	if err != nil {
		return err
	}
	finalFile := filepath.Join(s.entityDir(entityType), id+".json")
	return persistence.AtomicWriteFile(finalFile, fullData, 0644)
}

// writeExperimentCAS 以乐观锁（compare-and-swap）原子提交实验更新。
// candidate.Version 必须等于当前盘上版本（调用方读取到的基础版本/代次）；
// 命中则写入 current.Version+1，否则返回 ErrVersionConflict——同一代并发写入
// 中仅一个成功，其余得到明确冲突。写入经 AtomicWriteFile 落盘并 fsync，重启后
// 仍保留唯一提交结果。
func (s *FileStore) writeExperimentCAS(ctx context.Context, candidate *domain.Experiment) (*domain.Experiment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	finalFile := filepath.Join(s.entityDir("experiments"), candidate.ID+".json")
	currentData, err := os.ReadFile(finalFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	currentRecord, err := persistence.DecodeRecord(currentData)
	if err != nil {
		return nil, err
	}
	var current domain.Experiment
	if err := json.Unmarshal(currentRecord.Data, &current); err != nil {
		return nil, err
	}
	// 乐观锁：基础版本不匹配即并发冲突，拒绝写入。
	if candidate.Version != current.Version {
		return nil, domain.ErrVersionConflict
	}
	next := *candidate
	next.Version = current.Version + 1
	data, err := json.Marshal(&next)
	if err != nil {
		return nil, err
	}
	encoded, err := persistence.EncodeRecord(persistence.Record{Version: next.Version, Data: data})
	if err != nil {
		return nil, err
	}
	if err := persistence.AtomicWriteFile(finalFile, encoded, 0644); err != nil {
		return nil, err
	}
	return &next, nil
}

func (s *FileStore) readEntity(ctx context.Context, entityType, id string, obj interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.entityDir(entityType), id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.ErrNotFound
		}
		return err
	}
	record, err := persistence.DecodeRecord(data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(record.Data, obj); err != nil {
		return err
	}
	return nil
}

func (s *FileStore) listEntities(ctx context.Context, entityType string, filter func([]byte) bool) ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := s.entityDir(entityType)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var results [][]byte
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		record, err := persistence.DecodeRecord(data)
		if err != nil {
			return nil, err
		}
		if filter != nil && !filter(record.Data) {
			continue
		}
		results = append(results, record.Data)
	}
	return results, nil
}

func (s *FileStore) deleteEntity(ctx context.Context, entityType, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filePath := filepath.Join(s.entityDir(entityType), id+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *FileStore) Close() error {
	return nil
}

var _ = fmt.Sprintf // 保留import
