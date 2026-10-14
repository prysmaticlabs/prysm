package sync

// EIP-8411 (prototype): the execution_payload gossip topic is retired. Envelopes
// are reconstructed from PayloadChunk messages (see validate_payload_chunk.go)
// or fetched via the by-root/by-range req/resp protocols; the gossip validator
// and subscriber that previously lived here were removed with the topic.

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
)

// hasSeenPayloadEnvelope reports whether an envelope for the given block root and
// builder index was already imported or marked bad.
func (s *Service) hasSeenPayloadEnvelope(root [32]byte, builderIdx primitives.BuilderIndex) bool {
	if s.seenPayloadEnvelopeCache == nil {
		return false
	}

	b := append(bytesutil.Bytes32(uint64(builderIdx)), root[:]...)
	_, seen := s.seenPayloadEnvelopeCache.Get(string(b))
	return seen
}

func (s *Service) setSeenPayloadEnvelope(root [32]byte, builderIdx primitives.BuilderIndex) {
	if s.seenPayloadEnvelopeCache == nil {
		return
	}

	b := append(bytesutil.Bytes32(uint64(builderIdx)), root[:]...)
	s.seenPayloadEnvelopeCache.Add(string(b), true)
}
