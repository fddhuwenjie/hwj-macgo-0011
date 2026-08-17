package persistence

import (
	"bytes"
	"testing"
)

func TestRecordRoundTrip(t *testing.T) {
	r := Record{
		Version: 1,
		Data:    []byte(`{"hello":"world"}`),
	}
	b, err := EncodeRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadRecord(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Version != r.Version {
		t.Fatalf("version mismatch: %d vs %d", decoded.Version, r.Version)
	}
	if string(decoded.Data) != string(r.Data) {
		t.Fatalf("data mismatch: %s vs %s", decoded.Data, r.Data)
	}
}

func TestCorruptRecord(t *testing.T) {
	b, _ := EncodeRecord(Record{Version:1, Data: []byte(`{"x":1}`)})
	b[len(b)-1] ^= 0xff // 破坏最后一个字节
	_, err := DecodeRecord(b)
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func FuzzDecodeRecord(f *testing.F) {
	b, _ := EncodeRecord(Record{Version:1, Data: []byte(`{"x":1}`)})
	f.Add(b)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeRecord(data)
	})
}
