package ssz

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncodeUint8(t *testing.T) {
	b := new(bytes.Buffer)
	Encode(b, uint8(12))
	fmt.Println(b)
}

func TestEncodeUint16(t *testing.T) {
	b := new(bytes.Buffer)
	Encode(b, uint16(256))
	fmt.Println(b)
}
