package encoder

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
)

func TestReadVarint(t *testing.T) {
	data := []byte("foobar data")
	prefixedData := append(proto.EncodeVarint(uint64(len(data))), data...)

	vi, err := readVarint(bytes.NewBuffer(prefixedData))
	if err != nil {
		t.Fatal(err)
	}
	if vi != uint64(len(data)) {
		t.Errorf("Received wrong varint. Wanted %d, got %d", len(data), vi)
	}
}
