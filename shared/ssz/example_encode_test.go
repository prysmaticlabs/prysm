package ssz_test

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// SSZ encoding takes a given data object to the given io.Writer. The most
// common use case is to use bytes.Buffer to collect the results to a buffer
// and consume the result.
func ExampleEncode() {
	// Given a data structure like this.
	type data struct {
		Field1 uint8
		Field2 []byte
	}

	// And some basic data.
	d := data{
		Field1: 10,
		Field2: []byte{1, 2, 3, 4},
	}

	// We use a bytes.Buffer as our io.Writer.
	buffer := new(bytes.Buffer)
	// ssz.Encode writes the encoded data to the buffer.
	if err := ssz.Encode(buffer, d); err != nil {
		// There was some failure with encoding SSZ.
		// You should probably handle this error in a non-fatal way.
		panic(err)
	}
	// And we can return the bytes from the buffer.
	encodedBytes := buffer.Bytes()
	fmt.Printf("ssz.Encode(%v) = %#x", d, encodedBytes)
}
