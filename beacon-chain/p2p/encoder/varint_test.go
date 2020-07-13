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

func TestReadVarint_ExceedsMaxLength(t *testing.T) {
	fByte := byte(1 << 7)
	// Terminating byte.
	tByte := byte(1 << 6)
	header := []byte{}
	for i := 0; i < 9; i++ {
		header = append(header, fByte)
	}
	header = append(header, tByte)
	_, err := readVarint(bytes.NewBuffer(header))
	if err != nil {
		t.Fatal("Expected no error from reading valid header")
	}
	length := len(header)
	// Add an additional byte to make header invalid.
	header = append(header[:length-1], []byte{fByte, tByte}...)

	_, err = readVarint(bytes.NewBuffer(header))
	if err == nil {
		t.Fatal("Expected error from reading invalid header")
	}
	if err != errExcessMaxLength {
		t.Errorf("Got incorrect error, wanted %v but got %v", errExcessMaxLength, err)
	}
}
