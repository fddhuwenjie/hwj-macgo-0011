package recovery

import (
	"context"
	"os"
	"path/filepath"
)

// CreateSnapshot 创建当前数据目录的快照
func CreateSnapshot(ctx context.Context, dataDir, snapshotDir string) error {
	// 简化：直接复制目录（生产环境可优化）
	if err := os.RemoveAll(snapshotDir); err != nil {
		return err
	}
	return filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(snapshotDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
