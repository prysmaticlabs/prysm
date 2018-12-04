package types

import (
	"fmt"

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
func NewGenesisBeaconState(genesisValidatorRegistry []*pb.ValidatorRecord) (*BeaconState, error) {
	// We seed the genesis state with a bunch of validators to
	// bootstrap the system.
	var err error
	if genesisValidatorRegistry == nil {
		genesisValidatorRegistry = v.InitialValidatorRegistry()

	}
	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := v.InitialShardAndCommitteesForSlots(genesisValidatorRegistry)
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

	var latestBlockHashes [][]byte
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		latestBlockHashes = append(latestBlockHashes, make([]byte, 0, 32))
	}

	return &BeaconState{
		data: &pb.BeaconState{
			LastStateRecalculationSlot:      0,
			JustifiedStreak:                 0,
			LastJustifiedSlot:               0,
			LastFinalizedSlot:               0,
			ValidatorRegistryLastChangeSlot: 0,
			LatestCrosslinks:                crosslinks,
			ValidatorRegistry:               genesisValidatorRegistry,
			ShardAndCommitteesForSlots:      shardAndCommitteesForSlots,
			PendingAttestations:             []*pb.AggregatedAttestation{},
			LatestBlockHash32S:              latestBlockHashes,
			RandaoMix:                       make([]byte, 0, 32),
		  ForkData: &pb.ForkData{
				PreForkVersion:  params.BeaconConfig().InitialForkVersion,
				PostForkVersion: params.BeaconConfig().InitialForkVersion,
				ForkSlot:        params.BeaconConfig().InitialForkSlot,
			},
		},
	}, nil
}

// CopyState returns a deep copy of the current state.
func (b *BeaconState) CopyState() *BeaconState {
	crosslinks := make([]*pb.CrosslinkRecord, len(b.LatestCrosslinks()))
	for index, crossLink := range b.LatestCrosslinks() {
		crosslinks[index] = &pb.CrosslinkRecord{
			ShardBlockHash: crossLink.GetShardBlockHash(),
			Slot:           crossLink.GetSlot(),
		}
	}

	validators := make([]*pb.ValidatorRecord, len(b.ValidatorRegistry()))
	for index, validator := range b.ValidatorRegistry() {
		validators[index] = &pb.ValidatorRecord{
			Pubkey:                 validator.GetPubkey(),
			RandaoCommitmentHash32: validator.GetRandaoCommitmentHash32(),
			Balance:                validator.GetBalance(),
			Status:                 validator.GetStatus(),
			LatestStatusChangeSlot: validator.GetLatestStatusChangeSlot(),
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
		LastStateRecalculationSlot:      b.LastStateRecalculationSlot(),
		JustifiedStreak:                 b.JustifiedStreak(),
		LastJustifiedSlot:               b.LastJustifiedSlot(),
		LastFinalizedSlot:               b.LastFinalizedSlot(),
		ValidatorRegistryLastChangeSlot: b.ValidatorRegistryLastChangeSlot(),
		LatestCrosslinks:                crosslinks,
		ValidatorRegistry:               validators,
		ShardAndCommitteesForSlots:      shardAndCommitteesForSlots,
		DepositsPenalizedInPeriod:       b.DepositsPenalizedInPeriod(),
		ForkData:                        b.ForkData(),
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

// ValidatorRegistryLastChangeSlot returns the slot of last time validator set changes.
func (b *BeaconState) ValidatorRegistryLastChangeSlot() uint64 {
	return b.data.ValidatorRegistryLastChangeSlot
}

// IsValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func (b *BeaconState) IsValidatorSetChange(slotNumber uint64) bool {
	if b.LastFinalizedSlot() <= b.ValidatorRegistryLastChangeSlot() {
		return false
	}
	if slotNumber-b.ValidatorRegistryLastChangeSlot() < params.BeaconConfig().MinValidatorSetChangeInterval {
		return false
	}

	shardProcessed := map[uint64]bool{}
	for _, shardAndCommittee := range b.ShardAndCommitteesForSlots() {
		for _, committee := range shardAndCommittee.ArrayShardAndCommittee {
			shardProcessed[committee.Shard] = true
		}
	}

	crosslinks := b.LatestCrosslinks()
	for shard := range shardProcessed {
		if b.ValidatorRegistryLastChangeSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new state transition will occur in the beacon chain.
func (b *BeaconState) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= b.LastStateRecalculationSlot()+params.BeaconConfig().CycleLength
}

// ShardAndCommitteesForSlots returns the shard committee object.
func (b *BeaconState) ShardAndCommitteesForSlots() []*pb.ShardAndCommitteeArray {
	return b.data.ShardAndCommitteesForSlots
}

// LatestCrosslinks returns the cross link records of the all the shards.
func (b *BeaconState) LatestCrosslinks() []*pb.CrosslinkRecord {
	return b.data.LatestCrosslinks
}

// ValidatorRegistry returns list of validators.
func (b *BeaconState) ValidatorRegistry() []*pb.ValidatorRecord {
	return b.data.ValidatorRegistry
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

// ForkData returns the relevant fork data for this beacon state.
func (b *BeaconState) ForkData() *pb.ForkData {
	return b.data.ForkData
}

// LatestBlockHashes returns the most recent 2*EPOCH_LENGTH block hashes.
func (b *BeaconState) LatestBlockHashes() [][32]byte {
	var blockhashes [][32]byte
	for _, hash := range b.data.LatestBlockHash32S {
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

// SignedParentHashes returns all the parent hashes stored in active state up to last cycle length.
func (b *BeaconState) SignedParentHashes(block *Block, attestation *pb.AggregatedAttestation) ([][32]byte, error) {
	latestBlockHashes := b.LatestBlockHashes()
	obliqueParentHashes := attestation.ObliqueParentHashes
	earliestSlot := int(block.SlotNumber()) - len(latestBlockHashes)

	startIdx := int(attestation.Slot) - earliestSlot - int(params.BeaconConfig().CycleLength) + 1
	endIdx := startIdx - len(attestation.ObliqueParentHashes) + int(params.BeaconConfig().CycleLength)
	if startIdx < 0 || endIdx > len(latestBlockHashes) || endIdx <= startIdx {
		return nil, fmt.Errorf("attempt to fetch recent blockhashes from %d to %d invalid", startIdx, endIdx)
	}

	hashes := make([][32]byte, 0, params.BeaconConfig().CycleLength)
	for i := startIdx; i < endIdx; i++ {
		hashes = append(hashes, latestBlockHashes[i])
	}

	for i := 0; i < len(obliqueParentHashes); i++ {
		hash := common.BytesToHash(obliqueParentHashes[i])
		hashes = append(hashes, hash)
	}
	return hashes, nil
}

// ClearAttestations removes attestations older than last state recalc slot.
func (b *BeaconState) ClearAttestations(lastStateRecalc uint64) {
	existing := b.data.PendingAttestations
	update := make([]*pb.AggregatedAttestation, 0, len(existing))
	for _, a := range existing {
		if a.GetSlot() >= lastStateRecalc {
			update = append(update, a)
		}
	}
	b.data.PendingAttestations = update
}

// CalculateNewBlockHashes builds a new slice of recent block hashes with the
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
// Computation of the state hash depends on this feature that slots with
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
// This method does not mutate the state.
func (b *BeaconState) CalculateNewBlockHashes(block *Block, parentSlot uint64) ([][]byte, error) {
	distance := block.SlotNumber() - parentSlot
	existing := b.data.LatestBlockHash32S
	update := existing[distance:]
	for len(update) < 2*int(params.BeaconConfig().CycleLength) {
		update = append(update, block.AncestorHash32S()[0])
	}
	return update, nil
}

// SetCrossLinks updates the inner proto's cross link records.
func (b *BeaconState) SetCrossLinks(crossLinks []*pb.CrosslinkRecord) {
	b.data.LatestCrosslinks = crossLinks
}

// SetDepositsPenalizedInPeriod updates the inner proto's penalized deposits.
func (b *BeaconState) SetDepositsPenalizedInPeriod(penalizedDeposits []uint64) {
	b.data.DepositsPenalizedInPeriod = penalizedDeposits
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

// SetPendingAttestations updates the inner proto's pending attestations.
func (b *BeaconState) SetPendingAttestations(pendingAttestations []*pb.AggregatedAttestation) {
	b.data.PendingAttestations = pendingAttestations
}

// SetRandaoMix updates the inner proto's randao mix.
func (b *BeaconState) SetRandaoMix(randaoMix []byte) {
	b.data.RandaoMix = randaoMix
}

// SetLatestBlockHashes updates the inner proto's recent block hashes.
func (b *BeaconState) SetLatestBlockHashes(blockHashes [][]byte) {
	b.data.LatestBlockHash32S = blockHashes
}

// SetShardAndCommitteesForSlots updates the inner proto's shard and committees for slots.
func (b *BeaconState) SetShardAndCommitteesForSlots(shardAndCommitteesForSlot []*pb.ShardAndCommitteeArray) {
	b.data.ShardAndCommitteesForSlots = shardAndCommitteesForSlot
}

// SetValidatorRegistry updates the state's internal validator set.
func (b *BeaconState) SetValidatorRegistry(validators []*pb.ValidatorRecord) {
	b.data.ValidatorRegistry = validators
}

// SetValidatorRegistryLastChangeSlot updates the inner proto's validator set change slot.
func (b *BeaconState) SetValidatorRegistryLastChangeSlot(slot uint64) {
	b.data.ValidatorRegistryLastChangeSlot = slot
}

func getPenaltyForPeriod(penalties []uint64, period uint64) uint64 {
	numPeriods := uint64(len(penalties))
	if numPeriods < period+1 {
		return 0
	}

	return penalties[period]
}
