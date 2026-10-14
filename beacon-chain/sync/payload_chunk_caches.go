package sync

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// EIP-8411 seen caches.
//
// payloadChunkCommitments tracks (slot, payload_chunks_root) pairs from every
// signature-valid bid seen on gossip and from bids inside valid beacon blocks.
// The stored value is the beacon block root once a block containing the bid is
// known, or a zero root until then. Chunks are only accepted for known
// commitments, which is what bounds pre-block buffering.
//
// seenPayloadChunkCache enforces first-seen per (payload_chunks_root, index),
// analogous to the data column sidecar (block_root, index) tracking.

func payloadChunkCommitmentKey(slot primitives.Slot, chunksRoot [32]byte) string {
	return string(append(bytesutil.Bytes8(uint64(slot)), chunksRoot[:]...))
}

func payloadChunkSeenKey(chunksRoot [32]byte, index uint64) string {
	return string(append(bytesutil.Bytes8(index), chunksRoot[:]...))
}

// registerPayloadChunkCommitment records a bid's chunk commitment. A non-zero
// block root binds the commitment to its beacon block; the first non-zero
// binding wins and is never overwritten by a zero one.
func (s *Service) registerPayloadChunkCommitment(slot primitives.Slot, chunksRoot, blockRoot [32]byte) {
	if s.payloadChunkCommitments == nil {
		return
	}
	key := payloadChunkCommitmentKey(slot, chunksRoot)
	if existing, ok := s.payloadChunkCommitments.Get(key); ok {
		existingRoot, rootOk := existing.([32]byte)
		if rootOk && existingRoot != ([32]byte{}) {
			return
		}
		if blockRoot == ([32]byte{}) {
			return
		}
	}
	s.payloadChunkCommitments.Add(slot, key, blockRoot)
	// A commitment becoming known may release chunks buffered before the bid
	// or block arrived.
	go s.drainPendingPayloadChunks(slot, chunksRoot)
}

// payloadChunkCommitmentBlockRoot reports whether the (slot, chunks root)
// commitment is known, and the beacon block root bound to it (zero when the
// containing block has not been seen yet).
func (s *Service) payloadChunkCommitmentBlockRoot(slot primitives.Slot, chunksRoot [32]byte) ([32]byte, bool) {
	if s.payloadChunkCommitments == nil {
		return [32]byte{}, false
	}
	v, ok := s.payloadChunkCommitments.Get(payloadChunkCommitmentKey(slot, chunksRoot))
	if !ok {
		return [32]byte{}, false
	}
	root, rootOk := v.([32]byte)
	if !rootOk {
		return [32]byte{}, true
	}
	return root, true
}

// registerPayloadChunkCommitmentFromBlock extracts the bid from a Gloas block
// and binds its chunk commitment to the block root.
func (s *Service) registerPayloadChunkCommitmentFromBlock(blk interfaces.ReadOnlyBeaconBlock, blockRoot [32]byte) {
	if blk.Version() < version.Gloas {
		return
	}
	sbid, err := blk.Body().SignedExecutionPayloadBid()
	if err != nil || sbid == nil || sbid.Message == nil {
		return
	}
	s.registerPayloadChunkCommitment(sbid.Message.Slot, bytesutil.ToBytes32(sbid.Message.PayloadChunksRoot), blockRoot)
}

func (s *Service) hasSeenPayloadChunk(chunksRoot [32]byte, index uint64) bool {
	if s.seenPayloadChunkCache == nil {
		return false
	}
	_, seen := s.seenPayloadChunkCache.Get(payloadChunkSeenKey(chunksRoot, index))
	return seen
}

func (s *Service) setSeenPayloadChunk(slot primitives.Slot, chunksRoot [32]byte, index uint64) {
	if s.seenPayloadChunkCache == nil {
		return
	}
	s.seenPayloadChunkCache.Add(slot, payloadChunkSeenKey(chunksRoot, index), true)
}

// unsetSeenPayloadChunk rolls back a first-seen marker for a chunk the pool
// could not hold, so a later delivery is not silently discarded.
func (s *Service) unsetSeenPayloadChunk(chunksRoot [32]byte, index uint64) {
	if s.seenPayloadChunkCache == nil {
		return
	}
	s.seenPayloadChunkCache.Remove(payloadChunkSeenKey(chunksRoot, index))
}
