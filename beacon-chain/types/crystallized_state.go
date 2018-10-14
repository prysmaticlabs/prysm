package types

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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

func initialValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.GetConfig().BootstrappedValidatorsCount; i++ {
		validator := &pb.ValidatorRecord{
			Status:            uint64(params.Active),
			Balance:           uint64(params.GetConfig().DepositSize * params.GetConfig().Gwei),
			WithdrawalAddress: []byte{},
			Pubkey:            []byte{},
		}
		validators = append(validators, validator)
	}
	return validators
}

func initialValidatorsFromJSON(genesisJSONPath string) ([]*pb.ValidatorRecord, error) {
	// #nosec G304
	// genesisJSONPath is a user input for the path of genesis.json.
	// Ex: /path/to/my/genesis.json.
	f, err := os.Open(genesisJSONPath)
	if err != nil {
		return nil, err
	}

	cState := &pb.CrystallizedState{}
	if err := jsonpb.Unmarshal(f, cState); err != nil {
		return nil, fmt.Errorf("error converting JSON to proto: %v", err)
	}

	return cState.Validators, nil
}

func initialShardAndCommitteesForSlots(validators []*pb.ValidatorRecord) ([]*pb.ShardAndCommitteeArray, error) {
	seed := make([]byte, 0, 32)
	committees, err := casper.ShuffleValidatorsToCommittees(common.BytesToHash(seed), validators, 1)
	if err != nil {
		return nil, err
	}

	// Starting with 2 cycles (128 slots) with the same committees.
	return append(committees, committees...), nil
}

// NewGenesisCrystallizedState initializes the crystallized state for slot 0.
func NewGenesisCrystallizedState(genesisJSONPath string) (*CrystallizedState, error) {
	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	var genesisValidators []*pb.ValidatorRecord
	var err error
	if genesisJSONPath != "" {
		log.Infof("Initializing crystallized state from %s", genesisJSONPath)
		genesisValidators, err = initialValidatorsFromJSON(genesisJSONPath)
		if err != nil {
			return nil, err
		}
	} else {
		genesisValidators = initialValidators()
	}

	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := initialShardAndCommitteesForSlots(genesisValidators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinks []*pb.CrosslinkRecord
	for i := 0; i < shardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			RecentlyChanged: false,
			ShardBlockHash:  make([]byte, 0, 32),
			Slot:            0,
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
			ValidatorSetChangeSlot:     0,
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
	return hashutil.Hash(data), nil
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

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	if c.LastStateRecalculationSlot() == 0 && slotNumber == params.GetConfig().CycleLength-1 {
		return true
	}
	return slotNumber >= c.LastStateRecalculationSlot()+params.GetConfig().CycleLength-1
}

// isValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func (c *CrystallizedState) isValidatorSetChange(slotNumber uint64) bool {
	if c.LastFinalizedSlot() <= c.ValidatorSetChangeSlot() {
		return false
	}
	if slotNumber-c.ValidatorSetChangeSlot() < params.GetConfig().MinValidatorSetChangeInterval {
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

// getAttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) getAttesterIndices(attestation *pb.AggregatedAttestation) ([]uint32, error) {
	slotsStart := c.LastStateRecalculationSlot() - params.GetConfig().CycleLength
	slotIndex := (attestation.Slot - slotsStart) % params.GetConfig().CycleLength
	shardCommitteeArray := c.data.ShardAndCommitteesForSlots
	shardCommittee := shardCommitteeArray[slotIndex].ArrayShardAndCommittee
	for i := 0; i < len(shardCommittee); i++ {
		if attestation.Shard == shardCommittee[i].Shard {
			return shardCommittee[i].Committee, nil
		}
	}
	return nil, fmt.Errorf("unable to find attestation based on slot: %v, Shard: %v", attestation.Slot, attestation.Shard)
}

// NewStateRecalculations computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
// We also check for validator set change transition and compute for new committees if necessary during this transition.
func (c *CrystallizedState) NewStateRecalculations(aState *ActiveState, block *Block, enableCrossLinks bool, enableRewardChecking bool) (*CrystallizedState, *ActiveState, error) {
	var blockVoteBalance uint64
	var LastStateRecalculationSlotCycleBack uint64
	var newValidators []*pb.ValidatorRecord
	var newCrosslinks []*pb.CrosslinkRecord
	var err error

	justifiedStreak := c.JustifiedStreak()
	justifiedSlot := c.LastJustifiedSlot()
	finalizedSlot := c.LastFinalizedSlot()
	lastStateRecalculationSlot := c.LastStateRecalculationSlot()
	validatorSetChangeSlot := c.ValidatorSetChangeSlot()
	blockVoteCache := aState.GetBlockVoteCache()
	ShardAndCommitteesForSlots := c.ShardAndCommitteesForSlots()
	timeSinceFinality := block.SlotNumber() - c.LastFinalizedSlot()
	recentBlockHashes := aState.RecentBlockHashes()

	if lastStateRecalculationSlot < params.GetConfig().CycleLength {
		LastStateRecalculationSlotCycleBack = 0
	} else {
		LastStateRecalculationSlotCycleBack = lastStateRecalculationSlot - params.GetConfig().CycleLength
	}

	// If reward checking is disabled, the new set of validators for the cycle
	// will remain the same.
	if !enableRewardChecking {
		newValidators = c.data.Validators
	}

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to LastStateRecalculationSlot - 1.
	for i := uint64(0); i < params.GetConfig().CycleLength; i++ {
		var voterIndices []uint32

		slot := LastStateRecalculationSlotCycleBack + i
		blockHash := recentBlockHashes[i]
		if _, ok := blockVoteCache[blockHash]; ok {
			blockVoteBalance = blockVoteCache[blockHash].VoteTotalDeposit
			voterIndices = blockVoteCache[blockHash].VoterIndices

			// Apply Rewards for each slot.
			if enableRewardChecking {
				newValidators = casper.CalculateRewards(
					slot,
					voterIndices,
					c.Validators(),
					blockVoteBalance,
					timeSinceFinality)
			}
		} else {
			blockVoteBalance = 0
		}

		if 3*blockVoteBalance >= 2*c.TotalDeposits() {
			if slot > justifiedSlot {
				justifiedSlot = slot
			}
			justifiedStreak++
		} else {
			justifiedStreak = 0
		}

		if slot > params.GetConfig().CycleLength && justifiedStreak >= params.GetConfig().CycleLength+1 && slot-params.GetConfig().CycleLength-1 > finalizedSlot {
			finalizedSlot = slot - params.GetConfig().CycleLength - 1
		}

		if enableCrossLinks {
			newCrosslinks, err = c.processCrosslinks(aState.PendingAttestations(), slot, block.SlotNumber())
			if err != nil {
				return nil, nil, err
			}
		}
	}

	// Clean up old attestations.
	newPendingAttestations := aState.cleanUpAttestations(lastStateRecalculationSlot)

	c.data.LastFinalizedSlot = finalizedSlot
	// Entering new validator set change transition.
	if c.isValidatorSetChange(block.SlotNumber()) {
		log.Info("Entering validator set change transition")
		validatorSetChangeSlot = lastStateRecalculationSlot
		ShardAndCommitteesForSlots, err = c.newValidatorSetRecalculations(block.ParentHash())
		if err != nil {
			return nil, nil, err
		}

		period := block.SlotNumber() / params.GetConfig().WithdrawalPeriod
		totalPenalties := c.penalizedETH(period)
		casper.ChangeValidators(block.SlotNumber(), totalPenalties, newValidators)
	}

	// Construct new crystallized state after cycle and validator set changed.
	newCrystallizedState := NewCrystallizedState(&pb.CrystallizedState{
		ShardAndCommitteesForSlots: ShardAndCommitteesForSlots,
		Validators:                 newValidators,
		LastStateRecalculationSlot: lastStateRecalculationSlot + params.GetConfig().CycleLength,
		LastJustifiedSlot:          justifiedSlot,
		JustifiedStreak:            justifiedStreak,
		LastFinalizedSlot:          finalizedSlot,
		Crosslinks:                 newCrosslinks,
		ValidatorSetChangeSlot:     validatorSetChangeSlot,
	})

	// Construct new active state after clean up pending attestations.
	newActiveState := NewActiveState(&pb.ActiveState{
		PendingAttestations: newPendingAttestations,
		RecentBlockHashes:   aState.data.RecentBlockHashes,
	}, aState.blockVoteCache)

	return newCrystallizedState, newActiveState, nil
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

	return append(c.data.ShardAndCommitteesForSlots[:params.GetConfig().CycleLength], newShardCommitteeArray...), nil
}

type shardAttestation struct {
	Shard          uint64
	shardBlockHash [32]byte
}

func copyCrosslinks(existing []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {
	new := make([]*pb.CrosslinkRecord, len(existing))
	for i := 0; i < len(existing); i++ {
		oldCL := existing[i]
		newBlockhash := make([]byte, len(oldCL.ShardBlockHash))
		copy(newBlockhash, oldCL.ShardBlockHash)
		newCL := &pb.CrosslinkRecord{
			RecentlyChanged: oldCL.RecentlyChanged,
			ShardBlockHash:  newBlockhash,
			Slot:            oldCL.Slot,
		}
		new[i] = newCL
	}

	return new
}

// processCrosslinks checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest beacon chain slot numbers.
func (c *CrystallizedState) processCrosslinks(pendingAttestations []*pb.AggregatedAttestation, slot uint64, currentSlot uint64) ([]*pb.CrosslinkRecord, error) {
	validators := c.data.Validators
	crosslinkRecords := copyCrosslinks(c.data.Crosslinks)
	rewardQuotient := casper.RewardQuotient(validators)

	shardAttestationBalance := map[shardAttestation]uint64{}
	for _, attestation := range pendingAttestations {
		indices, err := c.getAttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		shardBlockHash := [32]byte{}
		copy(shardBlockHash[:], attestation.ShardBlockHash)
		shardAtt := shardAttestation{
			Shard:          attestation.Shard,
			shardBlockHash: shardBlockHash,
		}
		if _, ok := shardAttestationBalance[shardAtt]; !ok {
			shardAttestationBalance[shardAtt] = 0
		}

		// find the total and vote balance of the shard committee.
		var totalBalance uint64
		var voteBalance uint64
		for _, attesterIndex := range indices {
			// find balance of validators who voted.
			if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
				voteBalance += validators[attesterIndex].Balance
			}
			// add to total balance of the committee.
			totalBalance += validators[attesterIndex].Balance
		}

		for _, attesterIndex := range indices {
			timeSinceLastConfirmation := currentSlot - crosslinkRecords[attestation.Shard].GetSlot()

			if !crosslinkRecords[attestation.Shard].RecentlyChanged {
				if bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex)) {
					casper.RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
				} else {
					casper.PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
				}
			}
		}

		shardAttestationBalance[shardAtt] += voteBalance

		// if 2/3 of committee voted on this crosslink, update the crosslink
		// with latest shard block hash, and slot number.
		if 3*voteBalance >= 2*totalBalance && !crosslinkRecords[attestation.Shard].RecentlyChanged {
			crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
				RecentlyChanged: true,
				ShardBlockHash:  attestation.ShardBlockHash,
				Slot:            slot,
			}
		}
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
