package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// BeaconState defines the core beacon chain's single
// state containing items pertaining to the validator
// set, recent block hashes, finalized slots, and more.
type BeaconState struct {
	data *pb.BeaconState
}

// Proto returns the underlying protobuf data within a state primitive.
func (b *BeaconState) Proto() *pb.BeaconState {
	return b.data
}

// Marshal encodes state object into the wire format.
func (b *BeaconState) Marshal() ([]byte, error) {
	return proto.Marshal(b.data)
}

// Hash serializes the state object then uses
// blake2b to hash the serialized object.
func (b *BeaconState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(b.data)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.Hash(data), nil
}

// ValidatorSetChangeSlot returns the slot of last time validator set changes.
func (b *BeaconState) ValidatorSetChangeSlot() uint64 {
	return b.data.ValidatorSetChangeSlot
}

// ShardAndCommitteesForSlots returns the shard committee object.
func (b *BeaconState) ShardAndCommitteesForSlots() []*pb.ShardAndCommitteeArray {
	return b.data.ShardAndCommitteesForSlots
}

// Crosslinks returns the cross link records of the all the shards.
func (b *BeaconState) Crosslinks() []*pb.CrosslinkRecord {
	return b.data.Crosslinks
}

// Validators returns list of validators.
func (b *BeaconState) Validators() []*pb.ValidatorRecord {
	return b.data.Validators
}

// LastStateRecalculationSlot returns when the last time crystallized state recalculated.
func (b *BeaconState) LastStateRecalculationSlot() uint64 {
	return b.data.LastStateRecalculationSlot
}

// LastFinalizedSlot returns the last finalized Slot of the beacon chain.
func (b *BeaconState) LastFinalizedSlot() uint64 {
	return b.data.LastFinalizedSlot
}

// LastJustifiedSlot return the last justified slot of the beacon chain.
func (b *BeaconState) LastJustifiedSlot() uint64 {
	return b.data.LastJustifiedSlot
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (b *BeaconState) JustifiedStreak() uint64 {
	return b.data.JustifiedStreak
}

// DepositsPenalizedInPeriod returns total deposits penalized in the given withdrawal period.
func (b *BeaconState) DepositsPenalizedInPeriod() []uint64 {
	return b.data.DepositsPenalizedInPeriod
}

// ForkSlotNumber returns the slot of last fork.
func (b *BeaconState) ForkSlotNumber() uint64 {
	return b.data.ForkSlotNumber
}

// PreForkVersion returns the last pre fork version.
func (b *BeaconState) PreForkVersion() uint64 {
	return b.data.PreForkVersion
}

// PostForkVersion returns the last post fork version.
func (b *BeaconState) PostForkVersion() uint64 {
	return b.data.PostForkVersion
}

// RecentBlockHashes returns the most recent 2*EPOCH_LENGTH block hashes.
func (b *BeaconState) RecentBlockHashes() [][32]byte {
	var blockhashes [][32]byte
	for _, hash := range b.data.RecentBlockHashes {
		blockhashes = append(blockhashes, common.BytesToHash(hash))
	}
	return blockhashes
}

// PendingAttestations returns attestations that have not yet been processed.
func (b *BeaconState) PendingAttestations() []*pb.AggregatedAttestation {
	return b.data.PendingAttestations
}

// RandaoMix tracks the current RANDAO state.
func (b *BeaconState) RandaoMix() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoMix)
	return h
}
