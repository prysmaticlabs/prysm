package gotypes_test

import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/prysmaticlabs/prysm/proto/gotypes"
)

func TestBytes32(t *testing.T) {
	input := make([]byte, 32)
	copy(input, []byte("Foobar!"))
	b := gotypes.NewBytes32(input)

	if err := b.Unmarshal(input); err != nil {
		t.Fatalf("Failed to unmarshal input. err = %v", err)
	}

	output := make([]byte, 32)
	n, err := b.MarshalTo(output)
	if n != 32 {
		t.Errorf("Unexpected n. Wanted 32, got %d", n)
	}
	if err != nil {
		t.Errorf("Unexpected error = %v", err)
	}
	if !bytes.Equal(input, output) {
		t.Errorf("Input != output bytes. Input=%v. Output=%v", input, output)
	}

	output, err = b.Marshal()
	if err != nil {
		t.Errorf("Unexpected error = %v", err)
	}
	if !bytes.Equal(input, output) {
		t.Errorf("Input != output bytes. Input=%v. Output=%v", input, output)
	}
}

func BenchmarkBytes32(b *testing.B) {
	bar := []gotypes.Bytes32{
		*gotypes.NewBytes32([]byte("A")),
	}

	for i := 0; i < b.N; i++ {

		var baz [][32]byte

		baz = *(*[][32]byte)(unsafe.Pointer(&state.LatestBlockRootHash32S))

		_ = baz
	}
}
