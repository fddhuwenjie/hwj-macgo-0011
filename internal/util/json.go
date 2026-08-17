package util

import (
	"bytes"
	"encoding/json"
)

// MarshalJSON 序列化为JSON，带缩进
func MarshalJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// 去除末尾换行
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// UnmarshalJSON 反序列化
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
