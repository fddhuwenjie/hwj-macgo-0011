package persistence

import (
	"bytes"
	"testing"
)

// Red-only contract: the streaming reader must reject a record whose payload
// was changed after encoding, even when the framing and length are intact.
func TestReadRecordRejectsPayloadCorruption(t *testing.T) {
	b, err := EncodeRecord(Record{Version: 1, Data: []byte(`{"run":"r-1"}`)})
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 0xff
	if _, err := ReadRecord(bytes.NewReader(b)); err == nil {
		t.Fatal("expected streaming reader to reject checksum mismatch")
	}
}
