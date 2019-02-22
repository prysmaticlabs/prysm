package ssz_test

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func ExampleEncode() {
	type data struct {
		Field1 uint8
		Field2 []byte
	}

	d := data{
		Field1: 10,
		Field2: []byte{1, 2, 3, 4},
	}

	buffer := new(bytes.Buffer)
	if err := ssz.Encode(buffer, d); err != nil {
		// There was some failure with encoding SSZ.
		panic(err)
	}
	encodedBytes := buffer.Bytes()

	fmt.Printf("ssz.Encode(%v) = %#x", d, encodedBytes)
}
