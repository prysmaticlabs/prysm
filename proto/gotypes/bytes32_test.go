package gotypes_test

import (
	"bytes"
	"testing"

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
