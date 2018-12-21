package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbcomm "github.com/prysmaticlabs/prysm/proto/common"
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
			ShardBlockRootHash32: make([]byte, 0, 32),
			Slot:                 0,
		})
	}

	var latestBlockHashes [][]byte
	for i := 0; i < 2*int(params.BeaconConfig().CycleLength); i++ {
		latestBlockHashes = append(latestBlockHashes, make([]byte, 0, 32))
	}

	return &BeaconState{
		data: &pb.BeaconState{
			ValidatorRegistry:                    genesisValidatorRegistry,
			ValidatorRegistryLastChangeSlot:      0,
			ValidatorRegistryExitCount:           0,
			ValidatorRegistryDeltaChainTipHash32: make([]byte, 0, 32),
			RandaoMixHash32:                      make([]byte, 0, 32),
			NextSeedHash32:                       make([]byte, 0, 32),
			ShardAndCommitteesAtSlots:            shardAndCommitteesForSlots,
			PersistentCommittees:                 []*pbcomm.Uint32List{},
			PersistentCommitteeReassignments:     []*pb.ShardReassignmentRecord{},
			PreviousJustifiedSlot:                0,
			JustifiedSlot:                        0,
			JustifiedSlotBitfield:                0,
			FinalizedSlot:                        0,
			LatestCrosslinks:                     crosslinks,
			LastStateRecalculationSlot:           0,
			LatestBlockRootHash32S:               latestBlockHashes,
			LatestPenalizedExitBalances:          []uint64{},
			LatestAttestations:                   []*pb.PendingAttestationRecord{},
			ProcessedPowReceiptRootHash32:        []byte{},
			CandidatePowReceiptRoots:             []*pb.CandidatePoWReceiptRootRecord{},
			GenesisTime:                          0,
			ForkData: &pb.ForkData{
				PreForkVersion:  params.BeaconConfig().InitialForkVersion,
				PostForkVersion: params.BeaconConfig().InitialForkVersion,
				ForkSlot:        params.BeaconConfig().InitialForkSlot,
			},
			Slot:                       0,
			JustifiedStreak:            0,
			PendingAttestations:        []*pb.Attestation{},
			ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
		},
	}, nil
}

// CopyState returns a deep copy of the current state.
func (b *BeaconState) CopyState() *BeaconState {
	crosslinks := make([]*pb.CrosslinkRecord, len(b.LatestCrosslinks()))
	for index, crossLink := range b.LatestCrosslinks() {
		crosslinks[index] = &pb.CrosslinkRecord{
			ShardBlockRootHash32: crossLink.GetShardBlockRootHash32(),
			Slot:                 crossLink.GetSlot(),
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

	shardAndCommitteesAtSlots := make([]*pb.ShardAndCommitteeArray, len(b.ShardAndCommitteesAtSlots()))
	for index, shardAndCommitteesAtSlot := range b.ShardAndCommitteesAtSlots() {
		shardAndCommittees := make([]*pb.ShardAndCommittee, len(shardAndCommitteesAtSlot.GetArrayShardAndCommittee()))
		for index, shardAndCommittee := range shardAndCommitteesAtSlot.GetArrayShardAndCommittee() {
			shardAndCommittees[index] = &pb.ShardAndCommittee{
				Shard:     shardAndCommittee.GetShard(),
				Committee: shardAndCommittee.GetCommittee(),
			}
		}
		shardAndCommitteesAtSlots[index] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: shardAndCommittees,
		}
	}

	newState := BeaconState{&pb.BeaconState{
		LastStateRecalculationSlot:      b.LastStateRecalculationSlot(),
		JustifiedStreak:                 b.JustifiedStreak(),
		JustifiedSlot:                   b.LastJustifiedSlot(),
		FinalizedSlot:                   b.LastFinalizedSlot(),
		ValidatorRegistryLastChangeSlot: b.ValidatorRegistryLastChangeSlot(),
		LatestCrosslinks:                crosslinks,
		ValidatorRegistry:               validators,
		ShardAndCommitteesForSlots:      shardAndCommitteesForSlots,
		ShardAndCommitteesAtSlots:       shardAndCommitteesAtSlots,
		LatestPenalizedExitBalances:     b.LatestPenalizedExitBalances(),
		ForkData:                        b.ForkData(),
		LatestBlockRootHash32S:          b.data.LatestBlockRootHash32S,
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

// ValidatorRegistry returns list of validators.
func (b *BeaconState) ValidatorRegistry() []*pb.ValidatorRecord {
	return b.data.ValidatorRegistry
}

// ValidatorRegistryLastChangeSlot returns the slot of last time validator set changes.
func (b *BeaconState) ValidatorRegistryLastChangeSlot() uint64 {
	return b.data.ValidatorRegistryLastChangeSlot
}

// ValidatorRegistryExitCount returns the count of the
// exited validators.
func (b *BeaconState) ValidatorRegistryExitCount() uint64 {
	return b.data.ValidatorRegistryExitCount
}

// ValidatorRegistryDeltaChainTipHash32 returns the delta hash of the
// validator registry.
func (b *BeaconState) ValidatorRegistryDeltaChainTipHash32() []byte {
	return b.data.ValidatorRegistryDeltaChainTipHash32
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

// ShardAndCommitteesAtSlots returns the shard committee object.
func (b *BeaconState) ShardAndCommitteesAtSlots() []*pb.ShardAndCommitteeArray {
	return b.data.ShardAndCommitteesAtSlots
}

// LatestCrosslinks returns the cross link records of the all the shards.
func (b *BeaconState) LatestCrosslinks() []*pb.CrosslinkRecord {
	return b.data.LatestCrosslinks
}

// NextSeedHash returns the next seed to be used in RANDAO.
func (b *BeaconState) NextSeedHash() [32]byte {
	var h [32]byte
	copy(h[:], b.data.NextSeedHash32)
	return h
}

// PersistentCommittees returns the committees stored in the beacon state.
func (b *BeaconState) PersistentCommittees() []*pbcomm.Uint32List {
	return b.data.PersistentCommittees
}

// PersistentCommitteeReassignments returns the assignments of the committees.
func (b *BeaconState) PersistentCommitteeReassignments() []*pb.ShardReassignmentRecord {
	return b.data.PersistentCommitteeReassignments
}

// PreviousJustifiedSlot retrieves the previous justified slot in the state.
func (b *BeaconState) PreviousJustifiedSlot() uint64 {
	return b.data.PreviousJustifiedSlot
}

// JustifiedSlotBitfield returns the bitfield of the justified slot.
func (b *BeaconState) JustifiedSlotBitfield() uint64 {
	return b.data.JustifiedSlotBitfield
}

// LatestAttestations returns the latest pending attestations that have not been
// processed.
func (b *BeaconState) LatestAttestations() []*pb.PendingAttestationRecord {
	return b.data.LatestAttestations
}

// ProcessedPowReceiptRootHash32 returns the root hashes of the
// processed transaction receipts from the POW chain.
func (b *BeaconState) ProcessedPowReceiptRootHash32() [32]byte {
	var h [32]byte
	copy(h[:], b.data.ProcessedPowReceiptRootHash32)
	return h
}

// CandidatePowReceiptRoots returns the root records of receipts that have
// yet to be processed.
func (b *BeaconState) CandidatePowReceiptRoots() []*pb.CandidatePoWReceiptRootRecord {
	return b.data.CandidatePowReceiptRoots
}

// GenesisTime returns the creation time of the
// genesis block.
func (b *BeaconState) GenesisTime() uint64 {
	return b.data.GenesisTime
}

// Slot refers to the slot of the last processed beacon block.
func (b *BeaconState) Slot() uint64 {
	return b.data.Slot
}

// LastStateRecalculationSlot returns when the last time crystallized state recalculated.
func (b *BeaconState) LastStateRecalculationSlot() uint64 {
	return b.data.LastStateRecalculationSlot
}

// LastFinalizedSlot returns the last finalized Slot of the beacon chain.
func (b *BeaconState) LastFinalizedSlot() uint64 {
	return b.data.FinalizedSlot
}

// LastJustifiedSlot return the last justified slot of the beacon chain.
func (b *BeaconState) LastJustifiedSlot() uint64 {
	return b.data.JustifiedSlot
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (b *BeaconState) JustifiedStreak() uint64 {
	return b.data.JustifiedStreak
}

// LatestPenalizedExitBalances returns total deposits penalized of the latest period.
func (b *BeaconState) LatestPenalizedExitBalances() []uint64 {
	return b.data.LatestPenalizedExitBalances
}

// ForkData returns the relevant fork data for this beacon state.
func (b *BeaconState) ForkData() *pb.ForkData {
	return b.data.ForkData
}

// LatestBlockRootHashes32 returns the most recent 2*EPOCH_LENGTH block hashes.
func (b *BeaconState) LatestBlockRootHashes32() [][32]byte {
	var blockhashes [][32]byte
	for _, hash := range b.data.LatestBlockRootHash32S {
		blockhashes = append(blockhashes, common.BytesToHash(hash))
	}
	return blockhashes
}

// PendingAttestations returns attestations that have not yet been processed.
func (b *BeaconState) PendingAttestations() []*pb.Attestation {
	return b.data.PendingAttestations
}

// RandaoMix tracks the current RANDAO state.
func (b *BeaconState) RandaoMix() [32]byte {
	var h [32]byte
	copy(h[:], b.data.RandaoMixHash32)
	return h
}

// ClearAttestations removes attestations older than last state recalc slot.
func (b *BeaconState) ClearAttestations(lastStateRecalc uint64) {
	existing := b.data.PendingAttestations
	update := make([]*pb.Attestation, 0, len(existing))
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
func (b *BeaconState) CalculateNewBlockHashes(block *pb.BeaconBlock, parentSlot uint64) ([][]byte, error) {
	distance := block.GetSlot() - parentSlot
	existing := b.data.LatestBlockRootHash32S
	update := existing[distance:]
	for len(update) < 2*int(params.BeaconConfig().CycleLength) {
		update = append(update, block.GetParentRootHash32())
	}
	return update, nil
}

// SetCrossLinks updates the inner proto's cross link records.
func (b *BeaconState) SetCrossLinks(crossLinks []*pb.CrosslinkRecord) {
	b.data.LatestCrosslinks = crossLinks
}

// SetLatestPenalizedExitBalances updates the inner proto's penalized deposits.
func (b *BeaconState) SetLatestPenalizedExitBalances(penalizedDeposits []uint64) {
	b.data.LatestPenalizedExitBalances = penalizedDeposits
}

// SetLastJustifiedSlot updates the inner proto's last justified slot.
func (b *BeaconState) SetLastJustifiedSlot(justifiedSlot uint64) {
	b.data.JustifiedSlot = justifiedSlot
}

// SetLastFinalizedSlot updates the inner proto's last finalized slot.
func (b *BeaconState) SetLastFinalizedSlot(finalizedSlot uint64) {
	b.data.FinalizedSlot = finalizedSlot
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
func (b *BeaconState) SetPendingAttestations(pendingAttestations []*pb.Attestation) {
	b.data.PendingAttestations = pendingAttestations
}

// SetRandaoMix updates the inner proto's randao mix.
func (b *BeaconState) SetRandaoMix(randaoMix []byte) {
	b.data.RandaoMixHash32 = randaoMix
}

// SetLatestBlockHashes updates the inner proto's recent block hashes.
func (b *BeaconState) SetLatestBlockHashes(blockHashes [][]byte) {
	b.data.LatestBlockRootHash32S = blockHashes
}

// SetShardAndCommitteesForSlots updates the inner proto's shard and committees for slots.
func (b *BeaconState) SetShardAndCommitteesForSlots(shardAndCommitteesForSlot []*pb.ShardAndCommitteeArray) {
	b.data.ShardAndCommitteesForSlots = shardAndCommitteesForSlot
}

// SetShardAndCommitteesAtSlots updates the inner proto's shard and committees for slots.
func (b *BeaconState) SetShardAndCommitteesAtSlots(shardAndCommitteesAtSlot []*pb.ShardAndCommitteeArray) {
	b.data.ShardAndCommitteesAtSlots = shardAndCommitteesAtSlot
}

// SetValidatorRegistry updates the state's internal validator set.
func (b *BeaconState) SetValidatorRegistry(validators []*pb.ValidatorRecord) {
	b.data.ValidatorRegistry = validators
}

// SetValidatorRegistryLastChangeSlot updates the inner proto's validator set change slot.
func (b *BeaconState) SetValidatorRegistryLastChangeSlot(slot uint64) {
	b.data.ValidatorRegistryLastChangeSlot = slot
}

// SetValidatorRegistryExitCount sets the exit count of the
// validator registry.
func (b *BeaconState) SetValidatorRegistryExitCount(count uint64) {
	b.data.ValidatorRegistryExitCount = count
}

// SetValidatorRegistryDeltaChainTipHash32 sets the delta hash of the validator registry.
func (b *BeaconState) SetValidatorRegistryDeltaChainTipHash32(chainTipHash []byte) {
	b.data.ValidatorRegistryDeltaChainTipHash32 = chainTipHash
}

// SetNextSeedHash sets the next seed hash in the beacon state.
func (b *BeaconState) SetNextSeedHash(hash [32]byte) {
	b.data.NextSeedHash32 = hash[:]
}

// SetPersistentCommittees sets the persistent committees in the beacon state.
func (b *BeaconState) SetPersistentCommittees(committees []*pbcomm.Uint32List) {
	b.data.PersistentCommittees = committees
}

// SetPersistentCommitteeReassignments sets the committee reassignments.
func (b *BeaconState) SetPersistentCommitteeReassignments(assignments []*pb.ShardReassignmentRecord) {
	b.data.PersistentCommitteeReassignments = assignments
}

// SetPreviousJustifiedSlot sets the previous justified slot into the state.
func (b *BeaconState) SetPreviousJustifiedSlot(slot uint64) {
	b.data.PreviousJustifiedSlot = slot
}

// SetJustifiedSlotBitfield sets the bitfield of the last justified slot.
func (b *BeaconState) SetJustifiedSlotBitfield(field uint64) {
	b.data.JustifiedSlotBitfield = field
}

// SetLatestAttestations sets the latest pending attestations into the state.
func (b *BeaconState) SetLatestAttestations(attestations []*pb.PendingAttestationRecord) {
	b.data.LatestAttestations = attestations
}

// SetProcessedPowReceiptHash saves the POW receipts which have
// been processed by the POW chain.
func (b *BeaconState) SetProcessedPowReceiptHash(hash [32]byte) {
	b.data.ProcessedPowReceiptRootHash32 = hash[:]
}

// SetCandidatePowReceiptRoots saves the latest roots of POW receipts that have
// yet to be processed.
func (b *BeaconState) SetCandidatePowReceiptRoots(records []*pb.CandidatePoWReceiptRootRecord) {
	b.data.CandidatePowReceiptRoots = records
}

// SetGenesisTime saves the genesis time of the genesis
// block into the state.
func (b *BeaconState) SetGenesisTime(time uint64) {
	b.data.GenesisTime = time
}

// SetForkData sets data relating to the fork into the state.
func (b *BeaconState) SetForkData(data *pb.ForkData) {
	b.data.ForkData = data
}

// SetSlot saves the slot of the last processed block to
// the beacon state.
func (b *BeaconState) SetSlot(slot uint64) {
	b.data.Slot = slot
}
