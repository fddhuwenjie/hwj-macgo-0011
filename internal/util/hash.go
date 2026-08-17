package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// HashBytes 计算SHA256哈希并返回hex字符串
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashReader 计算reader的哈希
func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
