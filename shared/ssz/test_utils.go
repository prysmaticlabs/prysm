package ssz

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// unhex converts a hex string to byte array
func unhex(str string) []byte {
	b, err := hex.DecodeString(stripSpace(str))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

func stripSpace(str string) string {
	return strings.Replace(str, " ", "", -1)
}

type simpleStruct struct {
	B uint16
	A uint8
}

type innerStruct struct {
	V uint16
}

type outerStruct struct {
	V    uint8
	SubV innerStruct
}
