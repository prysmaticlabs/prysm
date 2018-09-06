package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "types")

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	data           *pb.ActiveState
	blockVoteCache map[[32]byte]*VoteCache //blockVoteCache is not part of protocol state, it is used as a helper cache for cycle init calculations.
}

// VoteCache is a helper cache to track which validators voted for this block hash and total deposit supported for this block hash.
type VoteCache struct {
	VoterIndices     []uint32
	VoteTotalDeposit uint64
}

// NewActiveState creates a new active state with a explicitly set data field.
func NewActiveState(data *pb.ActiveState, blockVoteCache map[[32]byte]*VoteCache) *ActiveState {
	return &ActiveState{data: data, blockVoteCache: blockVoteCache}
}

// NewGenesisStates initializes a beacon chain with starting parameters.
func NewGenesisStates() (*ActiveState, *CrystallizedState, error) {
	// Bootstrap recent block hashes to all 0s for first 2 cycles (128 slots).
	var recentBlockHashes [][]byte
	for i := 0; i < 2*params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, make([]byte, 0, 32))
	}

	active := &ActiveState{
		data: &pb.ActiveState{
			PendingAttestations: []*pb.AttestationRecord{},
			RecentBlockHashes:   recentBlockHashes,
		},
		blockVoteCache: make(map[[32]byte]*VoteCache),
	}

	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	// TODO: Perform this task from some sort of genesis state json config instead.
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.BootstrappedValidatorsCount; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: params.DefaultEndDynasty, Balance: params.DefaultBalance, WithdrawalAddress: []byte{}, PublicKey: 0}
		validators = append(validators, validator)
	}

	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	seed := make([]byte, 0, 32)
	committees, err := casper.ShuffleValidatorsToCommittees(common.BytesToHash(seed), validators, 1, 0)
	if err != nil {
		return nil, nil, err
	}

	// Starting with 2 cycles (128 slots) with the same committees.
	committees = append(committees, committees...)
	indicesForSlots := append(committees, committees...)

	// Bootstrap cross link records.
	var crosslinkRecords []*pb.CrosslinkRecord
	for i := 0; i < params.ShardCount; i++ {
		crosslinkRecords = append(crosslinkRecords, &pb.CrosslinkRecord{
			Dynasty:   0,
			Blockhash: make([]byte, 0, 32),
		})
	}

	// Calculate total deposit from boot strapped validators.
	var totalDeposit uint64
	for _, v := range validators {
		totalDeposit += v.Balance
	}

	crystallized := &CrystallizedState{
		data: &pb.CrystallizedState{
			LastStateRecalc:        0,
			JustifiedStreak:        0,
			LastJustifiedSlot:      0,
			LastFinalizedSlot:      0,
			CurrentDynasty:         1,
			CrosslinkingStartShard: 0,
			TotalDeposits:          totalDeposit,
			DynastySeed:            []byte{},
			DynastySeedLastReset:   0,
			CrosslinkRecords:       crosslinkRecords,
			Validators:             validators,
			IndicesForSlots:        indicesForSlots,
		},
	}
	return active, crystallized, nil
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

// DeriveActiveState returns the active state for `block` based on its own state.
// This method should not modify its own state.
func (a *ActiveState) DeriveActiveState(block *Block, cState *CrystallizedState, parentSlot uint64) (*ActiveState, error) {
	// Derive the new set of pending attestations
	pendingAttestations := a.data.PendingAttestations
	newPendingAttestations := make([]*pb.AttestationRecord, 0)
	for i := 0; i < len(pendingAttestations); i++ {
		if pendingAttestations[i].GetSlot() >= cState.LastStateRecalc() {
			newPendingAttestations = append(newPendingAttestations, pendingAttestations[i])
		}
	}
	for i := 0; i < len(block.Attestations()); i++ {
		newPendingAttestations = append(newPendingAttestations, block.Attestations()[i])
	}

	// Derive the new value for RecentBlockHashes
	blockHash, err := block.Hash()
	if err != nil {
		return nil, err
	}
	newRecentBlockHashes := make([][]byte, 0)
	slotDist := block.SlotNumber() - parentSlot
	recentBlockHashes := a.data.RecentBlockHashes
	newRecentBlockHashes = recentBlockHashes[slotDist:]
	for ; len(newRecentBlockHashes) < 2 * params.CycleLength; {
		newRecentBlockHashes = append(newRecentBlockHashes, blockHash[:])
	}

	// With a valid beacon block, we can compute its attestations and store its votes/deposits in cache.
	var blockVoteCache map[[32]byte]*VoteCache
	for index := range block.Attestations() {
		blockVoteCache, err = a.calculateBlockVoteCache(index, block, cState)
		if err != nil {
			return nil, err
		}
	}

	return NewActiveState(&pb.ActiveState{
		PendingAttestations: newPendingAttestations,
		RecentBlockHashes: newRecentBlockHashes,
	}, blockVoteCache), nil
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

// calculateBlockVoteCache calculates and updates active state's block vote cache.
func (a *ActiveState) calculateBlockVoteCache(attestationIndex int, block *Block, cState *CrystallizedState) (map[[32]byte]*VoteCache, error) {
	attestation := block.Attestations()[attestationIndex]

	newVoteCache := a.GetBlockVoteCache()
	parentHashes := a.getSignedParentHashes(block, attestation)
	attesterIndices, err := cState.GetAttesterIndices(attestation)
	if err != nil {
		return nil, err
	}

	for _, h := range parentHashes {
		// Skip calculating for this hash if the hash is part of oblique parent hashes.
		var skip bool
		for _, obliqueParentHash := range attestation.ObliqueParentHashes {
			if bytes.Equal(h[:], obliqueParentHash) {
				skip = true
			}
		}
		if skip {
			continue
		}

		// Initialize vote cache of a given block hash if it doesn't exist already.
		if !a.isVoteCacheEmpty(h) {
			newVoteCache[h] = &VoteCache{VoterIndices: []uint32{}, VoteTotalDeposit: 0}
		}

		// Loop through attester indices, if the attester has voted but was not accounted for
		// in the cache, then we add attester's index and balance to the block cache.
		for i, attesterIndex := range attesterIndices {
			var existingAttester bool
			if !utils.CheckBit(attestation.AttesterBitfield, i) {
				continue
			}
			for _, indexInCache := range newVoteCache[h].VoterIndices {
				if attesterIndex == indexInCache {
					existingAttester = true
				}
			}
			if !existingAttester {
				newVoteCache[h].VoterIndices = append(newVoteCache[h].VoterIndices, attesterIndex)
				newVoteCache[h].VoteTotalDeposit += cState.Validators()[attesterIndex].Balance
			}
		}
	}
	return newVoteCache, nil
}
