package types

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	data *pb.ActiveState
	// blockVoteCache is not part of protocol state, it is
	// used as a helper cache for cycle init calculations.
	blockVoteCache map[[32]byte]*VoteCache
}

// NewGenesisActiveState initializes the active state for slot 0.
func NewGenesisActiveState() *ActiveState {
	// Bootstrap recent block hashes to all 0s for first 2 cycles.
	var recentBlockHashes [][]byte
	for i := 0; i < 2*int(params.GetConfig().CycleLength); i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 0, 32))
	}

	return &ActiveState{
		data: &pb.ActiveState{
			PendingAttestations: []*pb.AggregatedAttestation{},
			RecentBlockHashes:   recentBlockHashes,
		},
		blockVoteCache: make(map[[32]byte]*VoteCache),
	}
}

// NewActiveState creates a new active state with a explicitly set data field.
func NewActiveState(data *pb.ActiveState, blockVoteCache map[[32]byte]*VoteCache) *ActiveState {
	return &ActiveState{data: data, blockVoteCache: blockVoteCache}
}

// Proto returns the underlying protobuf data within a state primitive.
func (a *ActiveState) Proto() *pb.ActiveState {
	return a.data
}

// Marshal encodes active state object into the wire format.
func (a *ActiveState) Marshal() ([]byte, error) {
	return proto.Marshal(a.data)
}

// Hash serializes the active state object then uses
// blake2b to hash the serialized object.
func (a *ActiveState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(a.data)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.Hash(data), nil
}

// PendingAttestations returns attestations that have not yet been processed.
func (a *ActiveState) PendingAttestations() []*pb.AggregatedAttestation {
	return a.data.PendingAttestations
}

// RecentBlockHashes returns the most recent 2*EPOCH_LENGTH block hashes.
func (a *ActiveState) RecentBlockHashes() [][32]byte {
	var blockhashes [][32]byte
	for _, hash := range a.data.RecentBlockHashes {
		blockhashes = append(blockhashes, common.BytesToHash(hash))
	}
	return blockhashes
}

// IsVoteCacheEmpty returns false if vote cache of an input block hash doesn't exist.
func (a *ActiveState) isVoteCacheEmpty(blockHash [32]byte) bool {
	_, ok := a.blockVoteCache[blockHash]
	return ok
}

// GetBlockVoteCache returns the entire set of block vote cache.
func (a *ActiveState) GetBlockVoteCache() map[[32]byte]*VoteCache {
	return a.blockVoteCache
}

// appendNewAttestations appends new attestations from block in to active state.
// this is called during cycle transition.
func (a *ActiveState) appendNewAttestations(add []*pb.AggregatedAttestation) []*pb.AggregatedAttestation {
	existing := a.data.PendingAttestations
	return append(existing, add...)
}

// cleanUpAttestations removes attestations older than last state recalc slot.
func (a *ActiveState) cleanUpAttestations(lastStateRecalc uint64) []*pb.AggregatedAttestation {
	existing := a.data.PendingAttestations
	var update []*pb.AggregatedAttestation
	for i := 0; i < len(existing); i++ {
		if existing[i].GetSlot() >= lastStateRecalc {
			update = append(update, existing[i])
		}
	}
	return update
}

// calculateNewBlockHashes builds a new slice of recent block hashes with the
// provided block and the parent slot number.
//
// The algorithm is:
//   1) shift the array by block.SlotNumber - parentSlot (i.e. truncate the
//     first by the number of slots that have occurred between the block and
//     its parent).
//
//   2) fill the array with the parent block hash for all values between the parent
//     slot and the block slot.
//
// Computation of the active state hash depends on this feature that slots with
// missing blocks have the block hash of the next block hash in the chain.
//
// For example, if we have a segment of recent block hashes that look like this
//   [0xF, 0x7, 0x0, 0x0, 0x5]
//
// Where 0x0 is an empty or missing hash where no block was produced in the
// alloted slot. When storing the list (or at least when computing the hash of
// the active state), the list should be backfilled as such:
//
//   [0xF, 0x7, 0x5, 0x5, 0x5]
//
// This method does not mutate the active state.
func (a *ActiveState) calculateNewBlockHashes(block *Block, parentSlot uint64) ([][]byte, error) {
	distance := block.SlotNumber() - parentSlot
	existing := a.data.RecentBlockHashes
	update := existing[distance:]
	for len(update) < 2*int(params.GetConfig().CycleLength) {
		update = append(update, block.data.AncestorHashes[0])
	}

	return update, nil
}

// calculateBlockVoteCache calculates and updates active state's block vote cache.
func (a *ActiveState) calculateNewVoteCache(block *Block, cState *CrystallizedState) (map[[32]byte]*VoteCache, error) {
	update := voteCacheDeepCopy(a.GetBlockVoteCache())

	for i := 0; i < len(block.Attestations()); i++ {
		attestation := block.Attestations()[i]

		parentHashes := a.getSignedParentHashes(block, attestation)
		attesterIndices, err := cState.getAttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		for _, h := range parentHashes {
			// Skip calculating for this hash if the hash is part of oblique parent hashes.
			var skip bool
			for _, oblique := range attestation.ObliqueParentHashes {
				if bytes.Equal(h[:], oblique) {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			// Initialize vote cache of a given block hash if it doesn't exist already.
			if !a.isVoteCacheEmpty(h) {
				update[h] = newVoteCache()
			}

			// Loop through attester indices, if the attester has voted but was not accounted for
			// in the cache, then we add attester's index and balance to the block cache.
			for i, attesterIndex := range attesterIndices {
				var attesterExists bool
				if !bitutil.CheckBit(attestation.AttesterBitfield, i) {
					continue
				}
				for _, indexInCache := range update[h].VoterIndices {
					if attesterIndex == indexInCache {
						attesterExists = true
						break
					}
				}
				if !attesterExists {
					update[h].VoterIndices = append(update[h].VoterIndices, attesterIndex)
					update[h].VoteTotalDeposit += cState.Validators()[attesterIndex].Balance
				}
			}
		}
	}

	return update, nil
}

// CalculateNewActiveState returns the active state for `block` based on its own state.
// This method should not modify its own state.
func (a *ActiveState) CalculateNewActiveState(
	block *Block,
	cState *CrystallizedState,
	parentSlot uint64,
	enableAttestationValidity bool) (*ActiveState, error) {
	// Derive the new set of pending attestations.
	newPendingAttestations := a.appendNewAttestations(block.data.Attestations)

	// Derive the new set of recent block hashes.
	newRecentBlockHashes, err := a.calculateNewBlockHashes(block, parentSlot)
	if err != nil {
		return nil, fmt.Errorf("failed to update recent block hashes: %v", err)
	}

	log.Debugf("Calculating new active state. Crystallized state lastStateRecalc is %d", cState.LastStateRecalculationSlot())

	// With a valid beacon block, we can compute its attestations and store its votes/deposits in cache.
	blockVoteCache := a.blockVoteCache

	if enableAttestationValidity {
		blockVoteCache, err = a.calculateNewVoteCache(block, cState)
		if err != nil {
			return nil, fmt.Errorf("failed to update vote cache: %v", err)
		}
	}

	return NewActiveState(&pb.ActiveState{
		PendingAttestations: newPendingAttestations,
		RecentBlockHashes:   newRecentBlockHashes,
	}, blockVoteCache), nil
}

// getSignedParentHashes returns all the parent hashes stored in active state up to last cycle length.
func (a *ActiveState) getSignedParentHashes(block *Block, attestation *pb.AggregatedAttestation) [][32]byte {
	var signedParentHashes [][32]byte
	start := block.SlotNumber() - attestation.Slot
	end := block.SlotNumber() - attestation.Slot - uint64(len(attestation.ObliqueParentHashes)) + params.GetConfig().CycleLength

	recentBlockHashes := a.RecentBlockHashes()
	signedParentHashes = append(signedParentHashes, recentBlockHashes[start:end]...)

	for _, obliqueParentHashes := range attestation.ObliqueParentHashes {
		hashes := common.BytesToHash(obliqueParentHashes)
		signedParentHashes = append(signedParentHashes, hashes)
	}
	return signedParentHashes
}
