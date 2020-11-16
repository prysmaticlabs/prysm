package common

// ZeroSecretKey represents a zero secret key.
var ZeroSecretKey = [32]byte{}

// InfinitePublicKey represents an infinite public key.
var InfinitePublicKey = [48]byte{0xC0}

// InfiniteSignature represents an infinite signature.
var InfiniteSignature = [96]byte{0xC0}
