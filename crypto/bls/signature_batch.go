package bls

import "github.com/pkg/errors"

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

// RemoveDuplicates removes duplicate signature sets from the signature batch.
func (s *SignatureBatch) RemoveDuplicates() (int, *SignatureBatch, error) {
	if len(s.Signatures) == 0 || len(s.PublicKeys) == 0 || len(s.Messages) == 0 {
		return 0, s, nil
	}
	if len(s.Signatures) != len(s.PublicKeys) || len(s.Signatures) != len(s.Messages) {
		return 0, s, errors.Errorf("mismatch number of signatures, publickeys and messages in signature batch. "+
			"Signatures %d, Public Keys %d , Messages %d", s.Signatures, s.PublicKeys, s.Messages)
	}
	sigMap := make(map[string]int)
	duplicateSet := make(map[int]bool)
	for i := 0; i < len(s.Signatures); i++ {
		if sigIdx, ok := sigMap[string(s.Signatures[i])]; ok {
			if s.PublicKeys[sigIdx].Equals(s.PublicKeys[i]) &&
				s.Messages[sigIdx] == s.Messages[i] {
				duplicateSet[i] = true
				continue
			}
		}
		sigMap[string(s.Signatures[i])] = i
	}

	sigs := s.Signatures[:0]
	pubs := s.PublicKeys[:0]
	msgs := s.Messages[:0]

	for i := 0; i < len(s.Signatures); i++ {
		if duplicateSet[i] {
			continue
		}
		sigs = append(sigs, s.Signatures[i])
		pubs = append(pubs, s.PublicKeys[i])
		msgs = append(msgs, s.Messages[i])
	}

	s.Signatures = sigs
	s.PublicKeys = pubs
	s.Messages = msgs

	return len(duplicateSet), s, nil
}

// AggregateBatch aggregates common messages in the provided batch to
// reduce the number of pairings required when we finally verify the
// whole batch.
func (s *SignatureBatch) AggregateBatch() (*SignatureBatch, error) {
	if len(s.Signatures) == 0 || len(s.PublicKeys) == 0 || len(s.Messages) == 0 {
		return s, nil
	}
	if len(s.Signatures) != len(s.PublicKeys) || len(s.Signatures) != len(s.Messages) {
		return s, errors.Errorf("mismatch number of signatures, publickeys and messages in signature batch. "+
			"Signatures %d, Public Keys %d , Messages %d", s.Signatures, s.PublicKeys, s.Messages)
	}
	msgMap := make(map[[32]byte]*SignatureBatch)

	for i := 0; i < len(s.Messages); i++ {
		currMsg := s.Messages[i]
		currBatch, ok := msgMap[currMsg]
		if ok {
			currBatch.Signatures = append(currBatch.Signatures, s.Signatures[i])
			currBatch.Messages = append(currBatch.Messages, s.Messages[i])
			currBatch.PublicKeys = append(currBatch.PublicKeys, s.PublicKeys[i])
			continue
		}
		currBatch = &SignatureBatch{
			Signatures: [][]byte{s.Signatures[i]},
			Messages:   [][32]byte{s.Messages[i]},
			PublicKeys: []PublicKey{s.PublicKeys[i]},
		}
		msgMap[currMsg] = currBatch
	}
	newSt := NewSet()
	for rt, b := range msgMap {
		if len(b.PublicKeys) > 1 {
			aggPub := AggregateMultiplePubkeys(b.PublicKeys)
			aggSig, err := AggregateCompressedSignatures(b.Signatures)
			if err != nil {
				return nil, err
			}
			copiedRt := rt
			b.PublicKeys = []PublicKey{aggPub}
			b.Signatures = [][]byte{aggSig.Marshal()}
			b.Messages = [][32]byte{copiedRt}
		}
		newObj := *b
		newSt = newSt.Join(&newObj)
	}
	return newSt, nil
}
