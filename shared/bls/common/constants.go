package common

import "bytes"

// ZeroSecretKey represents a zero secret key.
var ZeroSecretKey = [32]byte{}

// InfinitePublicKey represents an infinite public key.
var InfinitePublicKey = [48]byte{0xC0}

// InfiniteSignature represents an infinite signature.
var InfiniteSignature = [96]byte{0xC0}

// SecretKeyIsZero checks the validity of a secret key.
func SecretKeyIsZero(key []byte) bool {
	return bytes.Equal(key, ZeroSecretKey[:])
}

// PublicKeyIsInfinite checks the validity of a public key.
func PublicKeyIsInfinite(key []byte) bool {
	return bytes.Equal(key, InfinitePublicKey[:])
}

// SignatureIsInfinite checks the validity of a signature.
func SignatureIsInfinite(key []byte) bool {
	return bytes.Equal(key, InfiniteSignature[:])
}
