package bls

// SignatureBatch refers to the defined set of
// signatures and its respective public keys and
// messages required to verify it.
type SignatureBatch struct {
	Signatures [][]byte
	PublicKeys []PublicKey
	Messages   [][32]byte
}

// NewSet constructs an empty signature batch object.
func NewSet() *SignatureBatch {
	return &SignatureBatch{
		Signatures: [][]byte{},
		PublicKeys: []PublicKey{},
		Messages:   [][32]byte{},
	}
}

// Join merges the provided signature batch to out current one.
func (s *SignatureBatch) Join(set *SignatureBatch) *SignatureBatch {
	s.Signatures = append(s.Signatures, set.Signatures...)
	s.PublicKeys = append(s.PublicKeys, set.PublicKeys...)
	s.Messages = append(s.Messages, set.Messages...)
	return s
}

// Verify the current signature batch using the batch verify algorithm.
func (s *SignatureBatch) Verify() (bool, error) {
	return VerifyMultipleSignatures(s.Signatures, s.Messages, s.PublicKeys)
}

// Copy the attached signature batch and return it
// to the caller.
func (s *SignatureBatch) Copy() *SignatureBatch {
	signatures := make([][]byte, len(s.Signatures))
	pubkeys := make([]PublicKey, len(s.PublicKeys))
	messages := make([][32]byte, len(s.Messages))
	for i := range s.Signatures {
		sig := make([]byte, len(s.Signatures[i]))
		copy(sig, s.Signatures[i])
		signatures[i] = sig
	}
	for i := range s.PublicKeys {
		pubkeys[i] = s.PublicKeys[i].Copy()
	}
	for i := range s.Messages {
		copy(messages[i][:], s.Messages[i][:])
	}
	return &SignatureBatch{
		Signatures: signatures,
		PublicKeys: pubkeys,
		Messages:   messages,
	}
}
