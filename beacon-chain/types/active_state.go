package types

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	data           *pb.ActiveState
	blockVoteCache map[[32]byte]*VoteCache //blockVoteCache is not part of protocol state, it is used as a helper cache for cycle init calculations.
}

// NewGenesisActiveState initializes the active state for slot 0.
func NewGenesisActiveState() *ActiveState {
	// Bootstrap recent block hashes to all 0s for first 2 cycles (128 slots).
	var recentBlockHashes [][]byte
	for i := 0; i < 2*params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 0, 32))
	}

	return &ActiveState{
		data: &pb.ActiveState{
			PendingAttestations: []*pb.AttestationRecord{},
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
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
}

// PendingAttestations returns attestations that have not yet been processed.
func (a *ActiveState) PendingAttestations() []*pb.AttestationRecord {
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

func (a *ActiveState) calculateNewAttestations(add []*pb.AttestationRecord, lastStateRecalc uint64) []*pb.AttestationRecord {
	existing := a.data.PendingAttestations
	update := []*pb.AttestationRecord{}
	for i := 0; i < len(existing); i++ {
		if existing[i].GetSlot() >= lastStateRecalc {
			update = append(update, existing[i])
		}
	}
	update = append(update, add...)

	return update
}

func (a *ActiveState) calculateNewBlockHashes(block *Block, parentSlot uint64) ([][]byte, error) {
	hash, err := block.Hash()
	if err != nil {
		return nil, err
	}

	dist := block.SlotNumber() - parentSlot
	existing := a.data.RecentBlockHashes
	update := existing[dist:]
	for len(update) < 2*params.CycleLength {
		update = append(update, hash[:])
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
				if !utils.CheckBit(attestation.AttesterBitfield, i) {
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
func (a *ActiveState) CalculateNewActiveState(block *Block, cState *CrystallizedState, parentSlot uint64) (*ActiveState, error) {
	// Derive the new set of pending attestations.
	newPendingAttestations := a.calculateNewAttestations(block.data.Attestations, cState.LastStateRecalc())

	// Derive the new value for RecentBlockHashes.
	newRecentBlockHashes, err := a.calculateNewBlockHashes(block, parentSlot)
	if err != nil {
		return nil, fmt.Errorf("failed to update recent block hashes: %v", err)
	}

	// With a valid beacon block, we can compute its attestations and store its votes/deposits in cache.
	newBlockVoteCache, err := a.calculateNewVoteCache(block, cState)
	if err != nil {
		return nil, fmt.Errorf("failed to update vote cache: %v", err)
	}

	return NewActiveState(&pb.ActiveState{
		PendingAttestations: newPendingAttestations,
		RecentBlockHashes:   newRecentBlockHashes,
	}, newBlockVoteCache), nil
}

// getSignedParentHashes returns all the parent hashes stored in active state up to last cycle length.
func (a *ActiveState) getSignedParentHashes(block *Block, attestation *pb.AttestationRecord) [][32]byte {
	var signedParentHashes [][32]byte
	start := block.SlotNumber() - attestation.Slot
	end := block.SlotNumber() - attestation.Slot - uint64(len(attestation.ObliqueParentHashes)) + params.CycleLength

	recentBlockHashes := a.RecentBlockHashes()
	signedParentHashes = append(signedParentHashes, recentBlockHashes[start:end]...)

	for _, obliqueParentHashes := range attestation.ObliqueParentHashes {
		hashes := common.BytesToHash(obliqueParentHashes)
		signedParentHashes = append(signedParentHashes, hashes)
	}
	return signedParentHashes
}
