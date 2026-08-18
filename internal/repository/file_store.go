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

	// 检查版本（乐观锁）
	// 注意：这里简化，需要对象有Version字段，但反射获取不方便，我们依赖上层在更新前检查版本
	// 这里假设调用者已检查版本，我们只做原子写入
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	// 添加元数据：版本、校验和、长度
	record := persistence.Record{
		Version: expectedVersion,
		Data:    data,
	}
	fullData, err := persistence.EncodeRecord(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.entityDir(entityType), 0755); err != nil {
		return err
	}
	// 原子写入：先写临时文件再重命名
	tmpFile := filepath.Join(s.entityDir(entityType), id+".tmp")
	finalFile := filepath.Join(s.entityDir(entityType), id+".json")
	if err := os.WriteFile(tmpFile, fullData, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, finalFile); err != nil {
		os.Remove(tmpFile)
		return err
	}
	return nil
}

func (s *FileStore) writeExperimentCAS(ctx context.Context, candidate *domain.Experiment) (*domain.Experiment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	current, err := persistence.DecodeRecord(currentData)
	if err != nil {
		return nil, err
	}
	var currentEntity domain.Experiment
	if err := json.Unmarshal(current.Data, &currentEntity); err != nil {
		return nil, err
	}
	if candidate.Version < currentEntity.Version || candidate.Version > currentEntity.Version+1 {
		return nil, domain.ErrVersionConflict
	}
	next := *candidate
	if next.Version == currentEntity.Version {
		next.Version++
	}
	data, err := json.Marshal(&next)
	if err != nil {
		return nil, err
	}
	fullData, err := persistence.EncodeRecord(persistence.Record{
		Version: next.Version,
		Data:    data,
	})
	if err != nil {
		return nil, err
	}
	tmpFile := filepath.Join(s.entityDir("experiments"), candidate.ID+".tmp")
	if err := os.WriteFile(tmpFile, fullData, 0644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpFile, finalFile); err != nil {
		_ = os.Remove(tmpFile)
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
