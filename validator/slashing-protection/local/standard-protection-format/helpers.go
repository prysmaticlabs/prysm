package interchangeformat

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func uint64FromString(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}

func pubKeyFromHex(str string) ([48]byte, error) {
	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return [48]byte{}, err
	}
	if len(pubKeyBytes) != 48 {
		return [48]byte{}, fmt.Errorf("public key does not correct, 48-byte length: %s", str)
	}
	var pk [48]byte
	copy(pk[:], pubKeyBytes[:48])
	return pk, nil
}

func rootFromHex(str string) ([32]byte, error) {
	rootHexBytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return [32]byte{}, err
	}
	if len(rootHexBytes) != 32 {
		return [32]byte{}, fmt.Errorf("public key does not correct, 32-byte length: %s", str)
	}
	var root [32]byte
	copy(root[:], rootHexBytes[:32])
	return root, nil
}
