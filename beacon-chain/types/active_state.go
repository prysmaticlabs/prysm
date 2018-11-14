package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	b "github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// ActiveState contains fields of current state of beacon chain,
// it changes every block.
type ActiveState struct {
	data *pb.ActiveState
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
			PendingSpecials:     []*pb.SpecialRecord{},
			RecentBlockHashes:   recentBlockHashes,
			RandaoMix:           make([]byte, 0, 32),
		},
	}
}

// NewActiveState creates a new active state with a explicitly set data field.
func NewActiveState(data *pb.ActiveState) *ActiveState {
	return &ActiveState{data: data}
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

// CopyState returns a deep copy of the current active state.
func (a *ActiveState) CopyState() *ActiveState {
	pendingAttestations := make([]*pb.AggregatedAttestation, len(a.PendingAttestations()))
	for index, pendingAttestation := range a.PendingAttestations() {
		pendingAttestations[index] = &pb.AggregatedAttestation{
			Slot:                pendingAttestation.GetSlot(),
			Shard:               pendingAttestation.GetShard(),
			JustifiedSlot:       pendingAttestation.GetJustifiedSlot(),
			JustifiedBlockHash:  pendingAttestation.GetJustifiedBlockHash(),
			ShardBlockHash:      pendingAttestation.GetShardBlockHash(),
			AttesterBitfield:    pendingAttestation.GetAttesterBitfield(),
			ObliqueParentHashes: pendingAttestation.GetObliqueParentHashes(),
			AggregateSig:        pendingAttestation.GetAggregateSig(),
		}
	}

	recentBlockHashes := make([][]byte, len(a.RecentBlockHashes()))
	for r, hash := range a.data.RecentBlockHashes {
		recentBlockHashes[r] = hash
	}

	pendingSpecials := make([]*pb.SpecialRecord, len(a.PendingSpecials()))
	for index, pendingSpecial := range a.PendingSpecials() {
		pendingSpecials[index] = &pb.SpecialRecord{
			Kind: pendingSpecial.GetKind(),
			Data: pendingSpecial.GetData(),
		}
	}
	randaoMix := a.RandaoMix()

	newA := ActiveState{
		data: &pb.ActiveState{
			PendingAttestations: pendingAttestations,
			RecentBlockHashes:   recentBlockHashes,
			PendingSpecials:     pendingSpecials,
			RandaoMix:           randaoMix[:],
		},
	}

	return &newA
}

// PendingAttestations returns attestations that have not yet been processed.
func (a *ActiveState) PendingAttestations() []*pb.AggregatedAttestation {
	return a.data.PendingAttestations
}

// PendingSpecials returns special records that have not yet been processed.
func (a *ActiveState) PendingSpecials() []*pb.SpecialRecord {
	return a.data.PendingSpecials
}

// RecentBlockHashes returns the most recent 2*EPOCH_LENGTH block hashes.
func (a *ActiveState) RecentBlockHashes() [][32]byte {
	var blockhashes [][32]byte
	for _, hash := range a.data.RecentBlockHashes {
		blockhashes = append(blockhashes, common.BytesToHash(hash))
	}
	return blockhashes
}

// RandaoMix tracks the current RANDAO state.
func (a *ActiveState) RandaoMix() [32]byte {
	var h [32]byte
	copy(h[:], a.data.RandaoMix)
	return h
}

// UpdateAttestations returns a new state with the provided attestations.
func (a *ActiveState) UpdateAttestations(attestations []*pb.AggregatedAttestation) *ActiveState {
	newState := a.CopyState()

	newState.data.PendingAttestations = append(newState.data.PendingAttestations, attestations...)
	return newState
}

// appendNewSpecialObject appends new special record object from block in to active state.
// this is called during block processing.
func (a *ActiveState) appendNewSpecialObject(record *pb.SpecialRecord) []*pb.SpecialRecord {
	existing := a.data.PendingSpecials
	return append(existing, record)
}

// clearAttestations removes attestations older than last state recalc slot.
func (a *ActiveState) clearAttestations(lastStateRecalc uint64) {
	existing := a.data.PendingAttestations
	update := make([]*pb.AggregatedAttestation, 0, len(existing))
	for _, a := range existing {
		if a.GetSlot() >= lastStateRecalc {
			update = append(update, a)
		}
	}

	a.data.PendingAttestations = update
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

// CalculateNewActiveState returns the active state for `block` based on its own state.
// This method should not modify its own state.
func (a *ActiveState) CalculateNewActiveState(
	block *Block,
	cState *CrystallizedState,
	parentSlot uint64) (*ActiveState, error) {
	var err error

	newState := a.CopyState()

	newState.clearAttestations(cState.LastStateRecalculationSlot())

	// Derive the new set of recent block hashes.
	newState.data.RecentBlockHashes, err = newState.calculateNewBlockHashes(block, parentSlot)
	if err != nil {
		return nil, fmt.Errorf("failed to update recent block hashes: %v", err)
	}

	log.Debugf("Calculating new active state. Crystallized state lastStateRecalc is %d", cState.LastStateRecalculationSlot())

	_, proposerIndex, err := casper.ProposerShardAndIndex(
		cState.ShardAndCommitteesForSlots(),
		cState.LastStateRecalculationSlot(),
		parentSlot)
	if err != nil {
		return nil, fmt.Errorf("could not get proposer index %v", err)
	}

	newRandao := setRandaoMix(block.RandaoReveal(), a.RandaoMix())
	newState.data.RandaoMix = newRandao[:]

	specialRecordData := make([][]byte, 2)
	for i := range specialRecordData {
		specialRecordData[i] = make([]byte, 32)
	}
	blockRandao := block.RandaoReveal()
	proposerIndexBytes := b.Bytes8(proposerIndex)
	specialRecordData[0] = proposerIndexBytes
	specialRecordData[1] = blockRandao[:]

	newState.data.PendingSpecials = a.appendNewSpecialObject(&pb.SpecialRecord{
		Kind: uint32(params.RandaoChange),
		Data: specialRecordData,
	})

	return newState, nil
}

// GetSignedParentHashes returns all the parent hashes stored in active state up to last cycle length.
func (a *ActiveState) GetSignedParentHashes(block *Block, attestation *pb.AggregatedAttestation) ([][32]byte, error) {
	recentBlockHashes := a.RecentBlockHashes()
	obliqueParentHashes := attestation.ObliqueParentHashes
	earliestSlot := int(block.SlotNumber()) - len(recentBlockHashes)

	startIdx := int(attestation.Slot) - earliestSlot - int(params.GetConfig().CycleLength) + 1
	endIdx := startIdx - len(attestation.ObliqueParentHashes) + int(params.GetConfig().CycleLength)
	if startIdx < 0 || endIdx > len(recentBlockHashes) || endIdx <= startIdx {
		return nil, fmt.Errorf("attempt to fetch recent blockhashes from %d to %d invalid", startIdx, endIdx)
	}

	hashes := make([][32]byte, 0, params.GetConfig().CycleLength)
	for i := startIdx; i < endIdx; i++ {
		hashes = append(hashes, recentBlockHashes[i])
	}

	for i := 0; i < len(obliqueParentHashes); i++ {
		hash := common.BytesToHash(obliqueParentHashes[i])
		hashes = append(hashes, hash)
	}

	return hashes, nil
}

// setRandaoMix sets the current randao seed into active state.
func setRandaoMix(blockRandao [32]byte, aStateRandao [32]byte) [32]byte {
	for i, b := range blockRandao {
		aStateRandao[i] ^= b
	}
	return aStateRandao
}
