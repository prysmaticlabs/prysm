package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconState defines the core beacon chain's single
// state containing items pertaining to the validator
// set, recent block hashes, finalized slots, and more.
type BeaconState struct {
	data *pb.BeaconState
}

// NewBeaconState creates a new beacon state with a explicitly set data field.
func NewBeaconState(data *pb.BeaconState) *BeaconState {
	return &BeaconState{data: data}
}

// NewGenesisBeaconState initializes the beacon chain state for slot 0.
func NewGenesisBeaconState(genesisValidators []*pb.ValidatorRecord) (*BeaconState, error) {
	// We seed the genesis state with a bunch of validators to
	// bootstrap the system.
	var err error
	if genesisValidators == nil {
		genesisValidators = v.InitialValidators()

	}
	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := v.InitialShardAndCommitteesForSlots(genesisValidators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinks []*pb.CrosslinkRecord
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			ShardBlockHash: make([]byte, 0, 32),
			Slot:           0,
		})
	}

	return &BeaconState{
		data: &pb.BeaconState{
			LastStateRecalculationSlot: 0,
			JustifiedStreak:            0,
			LastJustifiedSlot:          0,
			LastFinalizedSlot:          0,
			ValidatorSetChangeSlot:     0,
			ForkSlotNumber:             0,
			Crosslinks:                 crosslinks,
			Validators:                 genesisValidators,
			ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
			PreForkVersion:             uint64(params.BeaconConfig().InitialForkVersion),
			PostForkVersion:            uint64(params.BeaconConfig().InitialForkVersion),
		},
	}, nil
}

// CopyState returns a deep copy of the current state.
func (b *BeaconState) CopyState() *BeaconState {
	crosslinks := make([]*pb.CrosslinkRecord, len(b.Crosslinks()))
	for index, crossLink := range b.Crosslinks() {
		crosslinks[index] = &pb.CrosslinkRecord{
			ShardBlockHash: crossLink.GetShardBlockHash(),
			Slot:           crossLink.GetSlot(),
		}
	}

	validators := make([]*pb.ValidatorRecord, len(b.Validators()))
	for index, validator := range b.Validators() {
		validators[index] = &pb.ValidatorRecord{
			Pubkey:            validator.GetPubkey(),
			WithdrawalShard:   validator.GetWithdrawalShard(),
			WithdrawalAddress: validator.GetWithdrawalAddress(),
			RandaoCommitment:  validator.GetRandaoCommitment(),
			Balance:           validator.GetBalance(),
			Status:            validator.GetStatus(),
			ExitSlot:          validator.GetExitSlot(),
		}
	}

	shardAndCommitteesForSlots := make([]*pb.ShardAndCommitteeArray, len(b.ShardAndCommitteesForSlots()))
	for index, shardAndCommitteesForSlot := range b.ShardAndCommitteesForSlots() {
		shardAndCommittees := make([]*pb.ShardAndCommittee, len(shardAndCommitteesForSlot.GetArrayShardAndCommittee()))
		for index, shardAndCommittee := range shardAndCommitteesForSlot.GetArrayShardAndCommittee() {
			shardAndCommittees[index] = &pb.ShardAndCommittee{
				Shard:     shardAndCommittee.GetShard(),
				Committee: shardAndCommittee.GetCommittee(),
			}
		}
		shardAndCommitteesForSlots[index] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: shardAndCommittees,
		}
	}

	newState := BeaconState{&pb.BeaconState{
		LastStateRecalculationSlot: b.LastStateRecalculationSlot(),
		JustifiedStreak:            b.JustifiedStreak(),
		LastJustifiedSlot:          b.LastJustifiedSlot(),
		LastFinalizedSlot:          b.LastFinalizedSlot(),
		ValidatorSetChangeSlot:     b.ValidatorSetChangeSlot(),
		Crosslinks:                 crosslinks,
		Validators:                 validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
		DepositsPenalizedInPeriod:  b.DepositsPenalizedInPeriod(),
		PreForkVersion:             b.data.PreForkVersion,
		PostForkVersion:            b.data.PostForkVersion,
		ForkSlotNumber:             b.data.ForkSlotNumber,
	}}

	return &newState
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

// IsValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func (b *BeaconState) IsValidatorSetChange(slotNumber uint64) bool {
	if b.LastFinalizedSlot() <= b.ValidatorSetChangeSlot() {
		return false
	}
	if slotNumber-b.ValidatorSetChangeSlot() < params.BeaconConfig().MinValidatorSetChangeInterval {
		return false
	}

	shardProcessed := map[uint64]bool{}
	for _, shardAndCommittee := range b.ShardAndCommitteesForSlots() {
		for _, committee := range shardAndCommittee.ArrayShardAndCommittee {
			shardProcessed[committee.Shard] = true
		}
	}

	crosslinks := b.Crosslinks()
	for shard := range shardProcessed {
		if b.ValidatorSetChangeSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
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

// PenalizedETH calculates penalized total ETH during the last 3 withdrawal periods.
func (b *BeaconState) PenalizedETH(period uint64) uint64 {
	var totalPenalty uint64
	penalties := b.DepositsPenalizedInPeriod()
	totalPenalty += getPenaltyForPeriod(penalties, period)

	if period >= 1 {
		totalPenalty += getPenaltyForPeriod(penalties, period-1)
	}

	if period >= 2 {
		totalPenalty += getPenaltyForPeriod(penalties, period-2)
	}

	return totalPenalty
}

// SetCrossLinks updates the inner proto's cross link records.
func (b *BeaconState) SetCrossLinks(crossLinks []*pb.CrosslinkRecord) {
	b.data.Crosslinks = crossLinks
}

// SetLastJustifiedSlot updates the inner proto's last justified slot.
func (b *BeaconState) SetLastJustifiedSlot(justifiedSlot uint64) {
	b.data.LastJustifiedSlot = justifiedSlot
}

// SetLastFinalizedSlot updates the inner proto's last finalized slot.
func (b *BeaconState) SetLastFinalizedSlot(finalizedSlot uint64) {
	b.data.LastFinalizedSlot = finalizedSlot
}

// SetJustifiedStreak updates the inner proto's justified streak.
func (b *BeaconState) SetJustifiedStreak(justifiedSlot uint64) {
	b.data.JustifiedStreak = justifiedSlot
}

// SetLastStateRecalculationSlot updates the inner proto's last state recalc slot.
func (b *BeaconState) SetLastStateRecalculationSlot(slot uint64) {
	b.data.LastStateRecalculationSlot = slot
}

// SetShardAndCommitteesForSlots updates the inner proto's shard and committees for slots.
func (b *BeaconState) SetShardAndCommitteesForSlots(shardAndCommitteesForSlot []*pb.ShardAndCommitteeArray) {
	b.data.ShardAndCommitteesForSlots = shardAndCommitteesForSlot
}

// SetValidators updates the state's internal validator set.
func (b *BeaconState) SetValidators(validators []*pb.ValidatorRecord) {
	b.data.Validators = validators
}

// SetValidatorSetChangeSlot updates the inner proto's validator set change slot.
func (b *BeaconState) SetValidatorSetChangeSlot(slot uint64) {
	b.data.ValidatorSetChangeSlot = slot
}

func getPenaltyForPeriod(penalties []uint64, period uint64) uint64 {
	numPeriods := uint64(len(penalties))
	if numPeriods < period+1 {
		return 0
	}

	return penalties[period]
}
