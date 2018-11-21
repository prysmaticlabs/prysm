package types

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var shardCount = params.BeaconConfig().ShardCount

// CrystallizedState contains fields of every Slot state,
// it changes every Slot.
type CrystallizedState struct {
	data *pb.CrystallizedState
}

// NewCrystallizedState creates a new crystallized state with a explicitly set data field.
func NewCrystallizedState(data *pb.CrystallizedState) *CrystallizedState {
	return &CrystallizedState{data: data}
}

// NewGenesisCrystallizedState initializes the crystallized state for slot 0.
func NewGenesisCrystallizedState(genesisValidators []*pb.ValidatorRecord) (*CrystallizedState, error) {
	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	var err error
	if genesisValidators == nil {
		genesisValidators = casper.InitialValidators()

	}
	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := casper.InitialShardAndCommitteesForSlots(genesisValidators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinks []*pb.CrosslinkRecord
	for i := uint64(0); i < shardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			ShardBlockHash: make([]byte, 0, 32),
			Slot:           0,
		})
	}

	return &CrystallizedState{
		data: &pb.CrystallizedState{
			LastStateRecalculationSlot: 0,
			JustifiedStreak:            0,
			LastJustifiedSlot:          0,
			LastFinalizedSlot:          0,
			ValidatorSetChangeSlot:     0,
			ForkSlotNumber:             0,
			Crosslinks:                 crosslinks,
			Validators:                 genesisValidators,
			ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
			ValidatorSetDeltaHashChain: make([]byte, 0, 32),
			PreForkVersion:             params.BeaconConfig().InitialForkVersion,
			PostForkVersion:            params.BeaconConfig().InitialForkVersion,
		},
	}, nil
}

// Proto returns the underlying protobuf data within a state primitive.
func (c *CrystallizedState) Proto() *pb.CrystallizedState {
	return c.data
}

// Marshal encodes crystallized state object into the wire format.
func (c *CrystallizedState) Marshal() ([]byte, error) {
	return proto.Marshal(c.data)
}

// Hash serializes the crystallized state object then uses
// blake2b to hash the serialized object.
func (c *CrystallizedState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(c.data)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.Hash(data), nil
}

// CopyState returns a deep copy of the current state.
func (c *CrystallizedState) CopyState() *CrystallizedState {
	crosslinks := make([]*pb.CrosslinkRecord, len(c.Crosslinks()))
	for index, crossLink := range c.Crosslinks() {
		crosslinks[index] = &pb.CrosslinkRecord{
			ShardBlockHash: crossLink.GetShardBlockHash(),
			Slot:           crossLink.GetSlot(),
		}
	}

	validators := make([]*pb.ValidatorRecord, len(c.Validators()))
	for index, validator := range c.Validators() {
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

	shardAndCommitteesForSlots := make([]*pb.ShardAndCommitteeArray, len(c.ShardAndCommitteesForSlots()))
	for index, shardAndCommitteesForSlot := range c.ShardAndCommitteesForSlots() {
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

	newState := CrystallizedState{&pb.CrystallizedState{
		LastStateRecalculationSlot: c.LastStateRecalculationSlot(),
		JustifiedStreak:            c.JustifiedStreak(),
		LastJustifiedSlot:          c.LastJustifiedSlot(),
		LastFinalizedSlot:          c.LastFinalizedSlot(),
		ValidatorSetChangeSlot:     c.ValidatorSetChangeSlot(),
		Crosslinks:                 crosslinks,
		Validators:                 validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
		DepositsPenalizedInPeriod:  c.DepositsPenalizedInPeriod(),
		ValidatorSetDeltaHashChain: c.data.ValidatorSetDeltaHashChain,
		PreForkVersion:             c.data.PreForkVersion,
		PostForkVersion:            c.data.PostForkVersion,
		ForkSlotNumber:             c.data.ForkSlotNumber,
	}}

	return &newState
}

// LastStateRecalculationSlot returns when the last time crystallized state recalculated.
func (c *CrystallizedState) LastStateRecalculationSlot() uint64 {
	return c.data.LastStateRecalculationSlot
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (c *CrystallizedState) JustifiedStreak() uint64 {
	return c.data.JustifiedStreak
}

// LastJustifiedSlot return the last justified slot of the beacon chain.
func (c *CrystallizedState) LastJustifiedSlot() uint64 {
	return c.data.LastJustifiedSlot
}

// LastFinalizedSlot returns the last finalized Slot of the beacon chain.
func (c *CrystallizedState) LastFinalizedSlot() uint64 {
	return c.data.LastFinalizedSlot
}

// TotalDeposits returns total balance of the deposits of the active validators.
func (c *CrystallizedState) TotalDeposits() uint64 {
	validators := c.data.Validators
	totalDeposit := casper.TotalActiveValidatorDeposit(validators)
	return totalDeposit
}

// ValidatorSetChangeSlot returns the slot of last time validator set changes.
func (c *CrystallizedState) ValidatorSetChangeSlot() uint64 {
	return c.data.ValidatorSetChangeSlot
}

// ShardAndCommitteesForSlots returns the shard committee object.
func (c *CrystallizedState) ShardAndCommitteesForSlots() []*pb.ShardAndCommitteeArray {
	return c.data.ShardAndCommitteesForSlots
}

// Crosslinks returns the cross link records of the all the shards.
func (c *CrystallizedState) Crosslinks() []*pb.CrosslinkRecord {
	return c.data.Crosslinks
}

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// DepositsPenalizedInPeriod returns total deposits penalized in the given withdrawal period.
func (c *CrystallizedState) DepositsPenalizedInPeriod() []uint32 {
	return c.data.DepositsPenalizedInPeriod
}

// ForkSlotNumber returns the slot of last fork.
func (c *CrystallizedState) ForkSlotNumber() uint64 {
	return c.data.ForkSlotNumber
}

// PreForkVersion returns the last pre fork version.
func (c *CrystallizedState) PreForkVersion() uint32 {
	return c.data.PreForkVersion
}

// PostForkVersion returns the last post fork version.
func (c *CrystallizedState) PostForkVersion() uint32 {
	return c.data.PostForkVersion
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= c.LastStateRecalculationSlot()+params.BeaconConfig().CycleLength
}

// GetShardsAndCommitteesForSlot returns the shard committees of a given slot.
func (c *CrystallizedState) GetShardsAndCommitteesForSlot(slotNumber uint64) (*pb.ShardAndCommitteeArray, error) {
	return casper.GetShardAndCommitteesForSlot(c.ShardAndCommitteesForSlots(), c.LastStateRecalculationSlot(), slotNumber)
}

// isValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func (c *CrystallizedState) isValidatorSetChange(slotNumber uint64) bool {
	if c.LastFinalizedSlot() <= c.ValidatorSetChangeSlot() {
		return false
	}
	if slotNumber-c.ValidatorSetChangeSlot() < params.BeaconConfig().MinValidatorSetChangeInterval {
		return false
	}

	shardProcessed := map[uint64]bool{}

	for _, shardAndCommittee := range c.ShardAndCommitteesForSlots() {
		for _, committee := range shardAndCommittee.ArrayShardAndCommittee {
			shardProcessed[committee.Shard] = true
		}
	}

	crosslinks := c.Crosslinks()
	for shard := range shardProcessed {
		if c.ValidatorSetChangeSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
}

// AttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) AttesterIndices(attestation *pb.AggregatedAttestation) ([]uint32, error) {
	shardCommittees, err := casper.GetShardAndCommitteesForSlot(
		c.ShardAndCommitteesForSlots(),
		c.LastStateRecalculationSlot(),
		attestation.GetSlot())
	if err != nil {
		return nil, fmt.Errorf("unable to fetch ShardAndCommittees for slot %d: %v", attestation.Slot, err)
	}

	shardCommitteesArray := shardCommittees.ArrayShardAndCommittee
	for _, shardCommittee := range shardCommitteesArray {
		if attestation.Shard == shardCommittee.Shard {
			return shardCommittee.Committee, nil
		}
	}

	return nil, fmt.Errorf("unable to find committee for shard %d", attestation.Shard)
}

// NewStateRecalculations computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
// We also check for validator set change transition and compute for new committees if necessary during this transition.
func (c *CrystallizedState) NewStateRecalculations(aState *ActiveState, block *Block, db beaconDB) (*CrystallizedState, error) {
	var lastStateRecalculationSlotCycleBack uint64
	var err error

	newState := c.CopyState()
	justifiedStreak := c.JustifiedStreak()
	justifiedSlot := c.LastJustifiedSlot()
	finalizedSlot := c.LastFinalizedSlot()
	timeSinceFinality := block.SlotNumber() - newState.LastFinalizedSlot()
	recentBlockHashes := aState.RecentBlockHashes()
	newState.data.Validators = casper.CopyValidators(newState.Validators())

	if c.LastStateRecalculationSlot() < params.BeaconConfig().CycleLength {
		lastStateRecalculationSlotCycleBack = 0
	} else {
		lastStateRecalculationSlotCycleBack = c.LastStateRecalculationSlot() - params.BeaconConfig().CycleLength
	}

	// TODO(711): Need to implement clean-up mechanism for block vote cache.
	blockVoteCache, err := db.ReadBlockVoteCache(recentBlockHashes[0:params.BeaconConfig().CycleLength])
	if err != nil {
		return nil, err
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.BeaconConfig().CycleLength; i++ {
		var blockVoteBalance uint64

		slot := lastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]

		blockVoteBalance, newState.data.Validators = casper.TallyVoteBalances(blockHash, blockVoteCache,
			newState.data.Validators, timeSinceFinality)

		justifiedSlot, finalizedSlot, justifiedStreak = casper.FinalizeAndJustifySlots(slot, justifiedSlot, finalizedSlot,
			justifiedStreak, blockVoteBalance, c.TotalDeposits())
	}

	newState.data.Crosslinks, err = newState.processCrosslinks(aState.PendingAttestations(), newState.Validators(), block.SlotNumber())
	if err != nil {
		return nil, err
	}

	newState.data.LastJustifiedSlot = justifiedSlot
	newState.data.LastFinalizedSlot = finalizedSlot
	newState.data.JustifiedStreak = justifiedStreak
	newState.data.LastStateRecalculationSlot = newState.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength

	// Process the pending special records gathered from last cycle.
	newState.data.Validators, err = casper.ProcessSpecialRecords(block.SlotNumber(), newState.Validators(), aState.PendingSpecials())
	if err != nil {
		return nil, err
	}

	// Exit the validators when their balance fall below min online deposit size.
	newState.data.Validators = casper.CheckValidatorMinDeposit(newState.Validators(), block.SlotNumber())

	newState.data.LastFinalizedSlot = finalizedSlot
	// Entering new validator set change transition.
	if newState.isValidatorSetChange(block.SlotNumber()) {
		log.Info("Entering validator set change transition")
		newState.data.ValidatorSetChangeSlot = newState.LastStateRecalculationSlot()
		newState.data.ShardAndCommitteesForSlots, err = newState.newValidatorSetRecalculations(block.ParentHash())
		if err != nil {
			return nil, err
		}

		period := uint32(block.SlotNumber() / params.BeaconConfig().MinWithdrawalPeriod)
		totalPenalties := newState.penalizedETH(period)
		newState.data.Validators = casper.ChangeValidators(block.SlotNumber(), totalPenalties, newState.Validators())
	}

	printCommittee(newState.data.ShardAndCommitteesForSlots)

	return newState, nil
}

func printCommittee(shardAndCommittees []*pb.ShardAndCommitteeArray) {
	log.Debug("Dumping shard committees")
	for slot, shardCommittees := range shardAndCommittees {
		for _, shardCommittee := range shardCommittees.ArrayShardAndCommittee {
			log.Debugf("Committee slot: %d, committee shard: %d committee validator indices: %v",
				slot, shardCommittee.Shard, shardCommittee.Committee)
		}
	}
}

// newValidatorSetRecalculations recomputes the validator set.
func (c *CrystallizedState) newValidatorSetRecalculations(seed [32]byte) ([]*pb.ShardAndCommitteeArray, error) {
	lastSlot := len(c.data.ShardAndCommitteesForSlots) - 1
	lastCommitteeFromLastSlot := len(c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee) - 1
	crosslinkLastShard := c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee[lastCommitteeFromLastSlot].Shard
	crosslinkNextShard := (crosslinkLastShard + 1) % uint64(shardCount)

	newShardCommitteeArray, err := casper.ShuffleValidatorsToCommittees(
		seed,
		c.data.Validators,
		crosslinkNextShard,
	)
	if err != nil {
		return nil, err
	}

	return append(c.data.ShardAndCommitteesForSlots[params.BeaconConfig().CycleLength:], newShardCommitteeArray...), nil
}

// processCrosslinks checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest beacon chain slot numbers.
func (c *CrystallizedState) processCrosslinks(pendingAttestations []*pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord, currentSlot uint64) ([]*pb.CrosslinkRecord, error) {
	crosslinkRecords := c.data.Crosslinks
	slot := c.LastStateRecalculationSlot() + params.BeaconConfig().CycleLength

	for _, attestation := range pendingAttestations {
		indices, err := c.AttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		totalBalance, voteBalance, err := casper.VotedBalanceInAttestation(validators, indices, attestation)
		if err != nil {
			return nil, err
		}

		err = casper.ApplyCrosslinkRewardsAndPenalties(crosslinkRecords, currentSlot, indices, attestation,
			validators, totalBalance, voteBalance)
		if err != nil {
			return nil, err
		}

		crosslinkRecords = casper.ProcessCrosslink(slot, voteBalance, totalBalance, attestation, crosslinkRecords)

	}
	return crosslinkRecords, nil
}

func getPenaltyForPeriod(penalties []uint32, period uint32) uint64 {
	numPeriods := uint32(len(penalties))
	if numPeriods < period+1 {
		return 0
	}

	return uint64(penalties[period])
}

// penalizedETH calculates penalized total ETH during the last 3 withdrawal periods.
func (c *CrystallizedState) penalizedETH(period uint32) uint64 {
	var totalPenalty uint64

	penalties := c.DepositsPenalizedInPeriod()

	totalPenalty += getPenaltyForPeriod(penalties, period)

	if period >= 1 {
		totalPenalty += getPenaltyForPeriod(penalties, period-1)
	}

	if period >= 2 {
		totalPenalty += getPenaltyForPeriod(penalties, period-2)
	}

	return totalPenalty
}
