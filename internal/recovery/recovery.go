package recovery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"experiment-trace/internal/journal"
	"experiment-trace/internal/persistence"
)

// Recover 从WAL恢复数据到目标目录（应用所有条目）
// 实际实现需与存储配合，这里提供框架
func Recover(ctx context.Context, walDir, dataDir string) error {
	// 1. 如果存在快照，加载快照
	// 2. 重放WAL
	// 简化：这里只读取WAL并打印（实际应调用存储写入）
	walPath := filepath.Join(walDir, "wal.log")
	f, err := os.Open(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	for {
		record, err := persistence.ReadRecord(f)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		var entry journal.Entry
		if err := json.Unmarshal(record.Data, &entry); err != nil {
			return err
		}
		// 这里可以应用entry到存储
		_ = entry
	}
	return nil
}
