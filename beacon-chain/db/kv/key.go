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

func hasCapellaKey(enc []byte) bool {
	if len(capellaKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(capellaKey)], capellaKey)
}

func hasCapellaBlindKey(enc []byte) bool {
	if len(capellaBlindKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(capellaBlindKey)], capellaBlindKey)
}

func hasEip4844Key(enc []byte) bool {
	if len(eip4844Key) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(eip4844Key)], eip4844Key)
}

func hasEip4844BlindKey(enc []byte) bool {
	if len(eip4844BlindKey) >= len(enc) {
		return false
	}
	return bytes.Equal(enc[:len(eip4844BlindKey)], eip4844BlindKey)
}
