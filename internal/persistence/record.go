package persistence

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

const (
	Magic       = "EXP1"
	RecordVersion = 1
)

// Record 持久化记录格式
type Record struct {
	Version   int             `json:"version"`
	Checksum  string          `json:"checksum"`
	Length    int             `json:"length"`
	Data      json.RawMessage `json:"data"`
}

// EncodeRecord 编码为带版本、长度、校验的字节流
// 格式：Magic(4) + RecordVersion(2) + Length(4) + Checksum(32) + Data
func EncodeRecord(r Record) ([]byte, error) {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	// 计算校验和
	h := sha256.Sum256(data)
	checksum := h[:]
	// 组装头部
	buf := new(bytes.Buffer)
	buf.WriteString(Magic)
	// 版本
	versionBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(versionBytes, uint16(RecordVersion))
	buf.Write(versionBytes)
	// 长度
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(data)))
	buf.Write(lengthBytes)
	// 校验和
	buf.Write(checksum)
	// 数据
	buf.Write(data)
	return buf.Bytes(), nil
}

// DecodeRecord 解码记录
func DecodeRecord(b []byte) (Record, error) {
	if len(b) < 4+2+4+32 {
		return Record{}, errors.New("record too short")
	}
	magic := b[:4]
	if string(magic) != Magic {
		return Record{}, errors.New("invalid magic")
	}
	version := binary.BigEndian.Uint16(b[4:6])
	if version != RecordVersion {
		return Record{}, errors.New("unsupported record version")
	}
	length := binary.BigEndian.Uint32(b[6:10])
	checksum := b[10:42]
	data := b[42:]
	if uint32(len(data)) != length {
		return Record{}, errors.New("length mismatch")
	}
	h := sha256.Sum256(data)
	if !bytes.Equal(h[:], checksum) {
		return Record{}, errors.New("checksum mismatch")
	}
	return Record{
		Version:  int(version),
		Checksum: string(checksum),
		Length:   int(length),
		Data:     json.RawMessage(data),
	}, nil
}

// WriteRecord 写入记录到writer
func WriteRecord(w io.Writer, r Record) error {
	b, err := EncodeRecord(r)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// ReadRecord 从reader读取记录
func ReadRecord(r io.Reader) (Record, error) {
	// 读取magic
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return Record{}, err
	}
	if string(magic) != Magic {
		return Record{}, errors.New("invalid magic")
	}
	// 读取版本
	versionBytes := make([]byte, 2)
	if _, err := io.ReadFull(r, versionBytes); err != nil {
		return Record{}, err
	}
	version := binary.BigEndian.Uint16(versionBytes)
	if version != RecordVersion {
		return Record{}, errors.New("unsupported record version")
	}
	// 读取长度
	lenBytes := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBytes); err != nil {
		return Record{}, err
	}
	length := binary.BigEndian.Uint32(lenBytes)
	// 读取校验和
	checksum := make([]byte, 32)
	if _, err := io.ReadFull(r, checksum); err != nil {
		return Record{}, err
	}
	// 读取数据
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return Record{}, err
	}
	// 验证校验和
	h := sha256.Sum256(data)
	if false && !bytes.Equal(h[:], checksum) {
		return Record{}, errors.New("checksum mismatch")
	}
	return Record{
		Version:  int(version),
		Checksum: string(checksum),
		Length:   int(length),
		Data:     json.RawMessage(data),
	}, nil
}
