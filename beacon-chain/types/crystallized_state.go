package types

import (
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

var shardCount = params.GetConfig().ShardCount

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
func NewGenesisCrystallizedState(genesisJSONPath string) (*CrystallizedState, error) {
	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	var genesisValidators []*pb.ValidatorRecord
	var err error
	if genesisJSONPath != "" {
		log.Infof("Initializing crystallized state from %s", genesisJSONPath)
		genesisValidators, err = casper.InitialValidatorsFromJSON(genesisJSONPath)
		if err != nil {
			return nil, err
		}
	} else {
		genesisValidators = casper.InitialValidators()
	}

	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := casper.InitialShardAndCommitteesForSlots(genesisValidators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinks []*pb.CrosslinkRecord
	for i := 0; i < shardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			Dynasty:        0,
			ShardBlockHash: make([]byte, 0, 32),
			Slot:           0,
		})
	}

	// Calculate total deposit from boot strapped validators.
	var totalDeposit uint64
	for _, v := range genesisValidators {
		totalDeposit += v.Balance
	}

	return &CrystallizedState{
		data: &pb.CrystallizedState{
			LastStateRecalculationSlot: 0,
			JustifiedStreak:            0,
			LastJustifiedSlot:          0,
			LastFinalizedSlot:          0,
			Dynasty:                    1,
			DynastySeed:                []byte{},
			DynastyStartSlot:           0,
			Crosslinks:                 crosslinks,
			Validators:                 genesisValidators,
			ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
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
	var hash [32]byte
	h := blake2b.Sum512(data)
	copy(hash[:], h[:32])
	return hash, nil
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

// Dynasty returns the current dynasty of the beacon chain.
func (c *CrystallizedState) Dynasty() uint64 {
	return c.data.Dynasty
}

// TotalDeposits returns total balance of the deposits of the active validators.
func (c *CrystallizedState) TotalDeposits() uint64 {
	validators := c.data.Validators
	totalDeposit := casper.TotalActiveValidatorDeposit(validators)
	return totalDeposit
}

// DynastyStartSlot returns the last dynasty start number.
func (c *CrystallizedState) DynastyStartSlot() uint64 {
	return c.data.DynastyStartSlot
}

// ShardAndCommitteesForSlots returns the shard committee object.
func (c *CrystallizedState) ShardAndCommitteesForSlots() []*pb.ShardAndCommitteeArray {
	return c.data.ShardAndCommitteesForSlots
}

// Crosslinks returns the cross link records of the all the shards.
func (c *CrystallizedState) Crosslinks() []*pb.CrosslinkRecord {
	return c.data.Crosslinks
}

// DynastySeed is used to select the committee for each shard.
func (c *CrystallizedState) DynastySeed() [32]byte {
	var h [32]byte
	copy(h[:], c.data.DynastySeed)
	return h
}

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// DepositsPenalizedInPeriod returns total deposits penalized in the given withdrawal period.
func (c *CrystallizedState) DepositsPenalizedInPeriod() []uint32 {
	return c.data.DepositsPenalizedInPeriod
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	if c.LastStateRecalculationSlot() == 0 && slotNumber == params.GetConfig().CycleLength-1 {
		return true
	}
	return slotNumber >= c.LastStateRecalculationSlot()+params.GetConfig().CycleLength-1
}

// isDynastyTransition checks if a dynasty transition can be processed. At that point,
// validator shuffle will occur.
func (c *CrystallizedState) isDynastyTransition(slotNumber uint64) bool {
	if c.LastFinalizedSlot() <= c.DynastyStartSlot() {
		return false
	}
	if slotNumber-c.DynastyStartSlot() < params.GetConfig().MinDynastyLength {
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
		if c.DynastyStartSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
}

// getAttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) getAttesterIndices(attestation *pb.AggregatedAttestation) ([]uint32, error) {
	slotsStart := c.LastStateRecalculationSlot() - params.GetConfig().CycleLength
	slotIndex := (attestation.Slot - slotsStart) % params.GetConfig().CycleLength
	return casper.CommitteeInShardAndSlot(slotIndex, attestation.GetShard(), c.data.GetShardAndCommitteesForSlots())
}

// NewStateRecalculations computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
// We also check for dynasty transition and compute for a new dynasty if necessary during this transition.
func (c *CrystallizedState) NewStateRecalculations(aState *ActiveState, block *Block, enableCrossLinks bool, enableRewardChecking bool) (*CrystallizedState, error) {
	var LastStateRecalculationSlotCycleBack uint64
	var newValidators []*pb.ValidatorRecord
	var newCrosslinks []*pb.CrosslinkRecord
	var err error

	justifiedStreak := c.JustifiedStreak()
	justifiedSlot := c.LastJustifiedSlot()
	finalizedSlot := c.LastFinalizedSlot()
	LastStateRecalculationSlot := c.LastStateRecalculationSlot()
	Dynasty := c.Dynasty()
	DynastyStartSlot := c.DynastyStartSlot()
	blockVoteCache := aState.GetBlockVoteCache()
	ShardAndCommitteesForSlots := c.ShardAndCommitteesForSlots()
	timeSinceFinality := block.SlotNumber() - c.LastFinalizedSlot()
	recentBlockHashes := aState.RecentBlockHashes()

	if LastStateRecalculationSlot < params.GetConfig().CycleLength {
		LastStateRecalculationSlotCycleBack = 0
	} else {
		LastStateRecalculationSlotCycleBack = LastStateRecalculationSlot - params.GetConfig().CycleLength
	}

	// If reward checking is disabled, the new set of validators for the cycle
	// will remain the same.
	if !enableRewardChecking {
		newValidators = c.data.Validators
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.GetConfig().CycleLength; i++ {
		var blockVoteBalance uint64

		slot := LastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]

		blockVoteBalance, newValidators = casper.TallyVoteBalances(blockHash, slot,
			blockVoteCache, c.Validators(), timeSinceFinality, enableRewardChecking)

		justifiedSlot, finalizedSlot, justifiedStreak = casper.FinalizeAndJustifySlots(slot, justifiedSlot, finalizedSlot,
			justifiedStreak, blockVoteBalance, c.TotalDeposits())

		if enableCrossLinks {
			newCrosslinks, err = c.processCrosslinks(aState.PendingAttestations(), slot, block.SlotNumber())
			if err != nil {
				return nil, err
			}
		}
	}

	c.data.LastFinalizedSlot = finalizedSlot
	// Entering new dynasty transition.
	if c.isDynastyTransition(block.SlotNumber()) {
		log.Info("Entering dynasty transition")
		DynastyStartSlot = LastStateRecalculationSlot
		Dynasty, ShardAndCommitteesForSlots, err = c.newDynastyRecalculations(block.ParentHash())
		if err != nil {
			return nil, err
		}

		period := block.SlotNumber() / params.GetConfig().WithdrawalPeriod
		totalPenalties := c.penalizedETH(period)
		casper.ChangeValidators(block.SlotNumber(), totalPenalties, newValidators)
	}

	// Construct new crystallized state after cycle and dynasty transition.
	newCrystallizedState := NewCrystallizedState(&pb.CrystallizedState{
		DynastySeed:                c.data.DynastySeed,
		ShardAndCommitteesForSlots: ShardAndCommitteesForSlots,
		Validators:                 newValidators,
		LastStateRecalculationSlot: LastStateRecalculationSlot + params.GetConfig().CycleLength,
		LastJustifiedSlot:          justifiedSlot,
		JustifiedStreak:            justifiedStreak,
		LastFinalizedSlot:          finalizedSlot,
		Crosslinks:                 newCrosslinks,
		DynastyStartSlot:           DynastyStartSlot,
		Dynasty:                    Dynasty,
	})

	return newCrystallizedState, nil
}

// newDynastyRecalculations recomputes the validator set. This method is called during a dynasty transition.
func (c *CrystallizedState) newDynastyRecalculations(seed [32]byte) (uint64, []*pb.ShardAndCommitteeArray, error) {
	lastSlot := len(c.data.ShardAndCommitteesForSlots) - 1
	lastCommitteeFromLastSlot := len(c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee) - 1
	crosslinkLastShard := c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee[lastCommitteeFromLastSlot].Shard
	crosslinkNextShard := (crosslinkLastShard + 1) % uint64(shardCount)
	nextDynasty := c.Dynasty() + 1

	newShardCommitteeArray, err := casper.ShuffleValidatorsToCommittees(
		seed,
		c.data.Validators,
		crosslinkNextShard,
	)
	if err != nil {
		return 0, nil, err
	}

	return nextDynasty, append(c.data.ShardAndCommitteesForSlots[:params.GetConfig().CycleLength], newShardCommitteeArray...), nil
}

func copyCrosslinks(existing []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {
	new := make([]*pb.CrosslinkRecord, len(existing))
	for i := 0; i < len(existing); i++ {
		oldCL := existing[i]
		newBlockhash := make([]byte, len(oldCL.ShardBlockHash))
		copy(newBlockhash, oldCL.ShardBlockHash)
		newCL := &pb.CrosslinkRecord{
			Dynasty:        oldCL.Dynasty,
			ShardBlockHash: newBlockhash,
			Slot:           oldCL.Slot,
		}
		new[i] = newCL
	}

	return new
}

// processCrosslinks checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest dynasty and slot numbers.
func (c *CrystallizedState) processCrosslinks(pendingAttestations []*pb.AggregatedAttestation, slot uint64, currentSlot uint64) ([]*pb.CrosslinkRecord, error) {
	validators := c.data.Validators
	dynasty := c.data.Dynasty
	crosslinkRecords := copyCrosslinks(c.data.Crosslinks)

	for _, attestation := range pendingAttestations {
		indices, err := c.getAttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		totalBalance, voteBalance := casper.VotedBalanceInAttestation(validators, indices, attestation)

		casper.ApplyCrosslinkRewardsAndPenalties(crosslinkRecords, currentSlot, indices, attestation,
			dynasty, validators, totalBalance, voteBalance)

		crosslinkRecords = casper.ProcessBalancesInCrosslink(slot, voteBalance, totalBalance, dynasty, attestation, crosslinkRecords)

	}
	return crosslinkRecords, nil
}

// penalizedETH calculates penalized total ETH during the last 3 withdrawal periods.
func (c *CrystallizedState) penalizedETH(periodIndex uint64) uint64 {
	var penalties uint64

	depositsPenalizedInPeriod := c.DepositsPenalizedInPeriod()
	penalties += uint64(depositsPenalizedInPeriod[periodIndex])

	if periodIndex >= 1 {
		penalties += uint64(depositsPenalizedInPeriod[periodIndex-1])
	}

	if periodIndex >= 2 {
		penalties += uint64(depositsPenalizedInPeriod[periodIndex-2])
	}

	return penalties
}
