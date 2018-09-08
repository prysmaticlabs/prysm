package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

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
func NewGenesisCrystallizedState() (*CrystallizedState, error) {
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
		return nil, err
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

	return &CrystallizedState{
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

// LastStateRecalc returns when the last time crystallized state recalculated.
func (c *CrystallizedState) LastStateRecalc() uint64 {
	return c.data.LastStateRecalc
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (c *CrystallizedState) JustifiedStreak() uint64 {
	return c.data.JustifiedStreak
}

// CrosslinkingStartShard returns next shard that crosslinking assignment will start from.
func (c *CrystallizedState) CrosslinkingStartShard() uint64 {
	return c.data.CrosslinkingStartShard
}

// LastJustifiedSlot return the last justified slot of the beacon chain.
func (c *CrystallizedState) LastJustifiedSlot() uint64 {
	return c.data.LastJustifiedSlot
}

// LastFinalizedSlot returns the last finalized Slot of the beacon chain.
func (c *CrystallizedState) LastFinalizedSlot() uint64 {
	return c.data.LastFinalizedSlot
}

// CurrentDynasty returns the current dynasty of the beacon chain.
func (c *CrystallizedState) CurrentDynasty() uint64 {
	return c.data.CurrentDynasty
}

// TotalDeposits returns total balance of deposits.
func (c *CrystallizedState) TotalDeposits() uint64 {
	return c.data.TotalDeposits
}

// DynastySeed is used to select the committee for each shard.
func (c *CrystallizedState) DynastySeed() [32]byte {
	var h [32]byte
	copy(h[:], c.data.DynastySeed)
	return h
}

// DynastySeedLastReset is the last finalized Slot that the crosslink seed was reset.
func (c *CrystallizedState) DynastySeedLastReset() uint64 {
	return c.data.DynastySeedLastReset
}

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// ValidatorsLength returns the number of total validators.
func (c *CrystallizedState) ValidatorsLength() int {
	return len(c.data.Validators)
}

// SetValidators sets the validator set.
func (c *CrystallizedState) SetValidators(validators []*pb.ValidatorRecord) {
	c.data.Validators = validators
}

// IndicesForSlots returns what active validators are part of the attester set
// at what slot, and in what shard.
func (c *CrystallizedState) IndicesForSlots() []*pb.ShardAndCommitteeArray {
	return c.data.IndicesForSlots
}

// CrosslinkRecords returns records about the most recent cross link or each shard.
func (c *CrystallizedState) CrosslinkRecords() []*pb.CrosslinkRecord {
	return c.data.CrosslinkRecords
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= c.LastStateRecalc()+params.CycleLength
}

// GetAttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) GetAttesterIndices(attestation *pb.AttestationRecord) ([]uint32, error) {
	lastStateRecalc := c.LastStateRecalc()
	// TODO: IndicesForSlots will return default value because the spec for dynasty transition is not finalized.
	shardCommitteeArray := c.IndicesForSlots()
	shardCommittee := shardCommitteeArray[attestation.Slot-lastStateRecalc].ArrayShardAndCommittee
	for i := 0; i < len(shardCommittee); i++ {
		if attestation.ShardId == shardCommittee[i].ShardId {
			return shardCommittee[i].Committee, nil
		}
	}
	return nil, fmt.Errorf("unable to find attestation based on slot: %v, shardID: %v", attestation.Slot, attestation.ShardId)
}

// DeriveCrystallizedState computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
func (c *CrystallizedState) DeriveCrystallizedState(aState *ActiveState) (*CrystallizedState, error) {
	var blockVoteBalance uint64
	justifiedStreak := c.JustifiedStreak()
	justifiedSlot := c.LastJustifiedSlot()
	finalizedSlot := c.LastFinalizedSlot()
	lastStateRecalc := c.LastStateRecalc()
	blockVoteCache := aState.GetBlockVoteCache()

	// walk through all the slots from LastStateRecalc - cycleLength to LastStateRecalc - 1.
	recentBlockHashes := aState.RecentBlockHashes()
	for i := uint64(0); i < params.CycleLength; i++ {
		slot := lastStateRecalc - params.CycleLength + i
		blockHash := recentBlockHashes[i]
		if _, ok := blockVoteCache[blockHash]; ok {
			blockVoteBalance = blockVoteCache[blockHash].VoteTotalDeposit
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

		if justifiedStreak >= params.CycleLength+1 && slot-params.CycleLength > finalizedSlot {
			finalizedSlot = slot - params.CycleLength
		}
	}

	// TODO: Utilize this value in the fork choice rule.
	newIndicesForSlots, err := casper.ShuffleValidatorsToCommittees(
		c.DynastySeed(),
		c.Validators(),
		c.CurrentDynasty(),
		c.CrosslinkingStartShard(),
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to get validators by slot and by shard: %v", err)
	}

	// TODO: Process Crosslink records here.
	newCrossLinkRecords := []*pb.CrosslinkRecord{}

	// TODO: Full rewards and penalties design is not finalized according to the spec.
	rewardedValidators, _ := casper.CalculateRewards(
		aState.PendingAttestations(),
		c.Validators(),
		c.CurrentDynasty(),
		c.TotalDeposits())

	// Get all active validators and calculate total balance for next cycle.
	var nextCycleBalance uint64
	nextCycleValidators := casper.ActiveValidatorIndices(c.Validators(), c.CurrentDynasty())
	for _, index := range nextCycleValidators {
		nextCycleBalance += c.Validators()[index].Balance
	}

	// Construct new crystallized state for cycle transition.
	newCrystallizedState := NewCrystallizedState(&pb.CrystallizedState{
		Validators:             rewardedValidators, // TODO: Stub. Static validator set because dynasty transition is not finalized according to the spec.
		LastStateRecalc:        lastStateRecalc + params.CycleLength,
		IndicesForSlots:        newIndicesForSlots,
		LastJustifiedSlot:      justifiedSlot,
		JustifiedStreak:        justifiedStreak,
		LastFinalizedSlot:      finalizedSlot,
		CrosslinkingStartShard: 0, // TODO: Stub. Need to see where this epoch left off.
		CrosslinkRecords:       newCrossLinkRecords,
		DynastySeedLastReset:   c.DynastySeedLastReset(), // TODO: Stub. Dynasty transition is not finalized according to the spec.
		TotalDeposits:          nextCycleBalance,
	})

	return newCrystallizedState, nil
}
