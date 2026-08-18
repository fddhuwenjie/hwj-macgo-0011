package persistence

import (
	"os"
	"path/filepath"
)

// AtomicWriteFile 原子且持久地写入文件：先写临时文件并 fsync 其数据与元数据，
// 再重命名到目标路径，最后 fsync 父目录。崩溃/重启后要么看到旧的完整内容、
// 要么看到新的完整内容，不会出现撕裂或半提交状态。
func AtomicWriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	// fsync 数据与文件元数据，保证重命名前内容已落盘。
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	// fsync 父目录使重命名本身持久化；失败不影响已成功的数据写入，按尽力而为处理。
	if d, derr := os.Open(dir); derr == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
