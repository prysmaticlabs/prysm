package kv

import "bytes"

// In order for an encoding to be Altair compatible, it must be prefixed with altair key.
func hasAltairKey(enc []byte) bool {
	if len(altairKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(altairKey)], altairKey)
}

func hasBellatrixKey(enc []byte) bool {
	if len(bellatrixKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(bellatrixKey)], bellatrixKey)
}

func hasBellatrixBlindKey(enc []byte) bool {
	if len(bellatrixBlindKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(bellatrixBlindKey)], bellatrixBlindKey)
}

func hasEip4844Key(enc []byte) bool {
	if len(eip4844Key) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(eip4844Key)], eip4844Key)
}
