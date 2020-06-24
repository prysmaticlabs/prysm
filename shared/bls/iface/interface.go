package iface

// SecretKey represents a BLS secret or private key.
type SecretKey interface {
	PublicKey() PublicKey
	Sign(msg []byte) Signature
	Marshal() []byte
}

// PublicKey represents a BLS public key.
type PublicKey interface {
	Marshal() []byte
	Copy() PublicKey
	Aggregate(p2 PublicKey) PublicKey
}

// Signature represents a BLS signature.
type Signature interface {
	Verify(pubKey PublicKey, msg []byte) bool
	AggregateVerify(pubKeys []PublicKey, msgs [][32]byte) bool
	FastAggregateVerify(pubKeys []PublicKey, msg [32]byte) bool
	Marshal() []byte
	Copy() Signature
}
