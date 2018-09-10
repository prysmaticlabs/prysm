package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
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

func initialValidators() []*pb.ValidatorRecord {
	var validators []*pb.ValidatorRecord
	for i := 0; i < params.BootstrappedValidatorsCount; i++ {
		validator := &pb.ValidatorRecord{StartDynasty: 0, EndDynasty: params.DefaultEndDynasty, Balance: params.DefaultBalance, WithdrawalAddress: []byte{}, PublicKey: 0}
		validators = append(validators, validator)
	}
	return validators
}

func initialShardAndCommitteesForSlots(validators []*pb.ValidatorRecord) ([]*pb.ShardAndCommitteeArray, error) {
	seed := make([]byte, 0, 32)
	committees, err := casper.ShuffleValidatorsToCommittees(common.BytesToHash(seed), validators, 1, 0)
	if err != nil {
		return nil, err
	}

	// Starting with 2 cycles (128 slots) with the same committees.
	return append(committees, committees...), nil
}

// NewGenesisCrystallizedState initializes the crystallized state for slot 0.
func NewGenesisCrystallizedState() (*CrystallizedState, error) {
	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	// TODO(#493): Perform this task from some sort of genesis state json config instead.
	validators := initialValidators()

	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := initialShardAndCommitteesForSlots(validators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinkRecords []*pb.CrosslinkRecord
	for i := 0; i < params.ShardCount; i++ {
		crosslinkRecords = append(crosslinkRecords, &pb.CrosslinkRecord{
			Dynasty:   0,
			Blockhash: make([]byte, 0, 32),
			Slot:      0,
		})
	}

	// Calculate total deposit from boot strapped validators.
	var totalDeposit uint64
	for _, v := range validators {
		totalDeposit += v.Balance
	}

	return &CrystallizedState{
		data: &pb.CrystallizedState{
			LastStateRecalc:            0,
			JustifiedStreak:            0,
			LastJustifiedSlot:          0,
			LastFinalizedSlot:          0,
			CurrentDynasty:             1,
			CrosslinkingStartShard:     0,
			TotalDeposits:              totalDeposit,
			DynastySeed:                []byte{},
			DynastySeedLastReset:       0,
			CrosslinkRecords:           crosslinkRecords,
			Validators:                 validators,
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

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= c.LastStateRecalc()+params.CycleLength
}

// getAttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) getAttesterIndices(attestation *pb.AttestationRecord) ([]uint32, error) {
	slotsStart := int64(c.LastStateRecalc()) - params.CycleLength
	slotIndex := int64(attestation.Slot) - slotsStart
	// TODO(#267): ShardAndCommitteesForSlots will return default value because the spec for dynasty transition is not finalized.
	shardCommitteeArray := c.data.ShardAndCommitteesForSlots
	shardCommittee := shardCommitteeArray[slotIndex].ArrayShardAndCommittee
	for i := 0; i < len(shardCommittee); i++ {
		if attestation.ShardId == shardCommittee[i].ShardId {
			return shardCommittee[i].Committee, nil
		}
	}
	return nil, fmt.Errorf("unable to find attestation based on slot: %v, shardID: %v", attestation.Slot, attestation.ShardId)
}

// CalculateNewCrystallizedState computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
func (c *CrystallizedState) CalculateNewCrystallizedState(aState *ActiveState, slotNumber uint64) (*CrystallizedState, error) {
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

	newShardAndCommitteesForSlots, err := c.calculateNewShardAndCommitteesForSlots()
	if err != nil {
		return nil, fmt.Errorf("Unable to get validators by slot and by shard: %v", err)
	}

	newCrossLinkRecords, err := c.processCrosslinks(aState.PendingAttestations(), slotNumber)
	if err != nil {
		return nil, err
	}

	// TODO(471): Update rewards and penalties scheme to align with latest spec.
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
		Validators:                 rewardedValidators,
		LastStateRecalc:            lastStateRecalc + params.CycleLength,
		ShardAndCommitteesForSlots: newShardAndCommitteesForSlots,
		LastJustifiedSlot:          justifiedSlot,
		JustifiedStreak:            justifiedStreak,
		LastFinalizedSlot:          finalizedSlot,
		CrosslinkingStartShard:     0, // TODO(#494): Stub. Need to see where this epoch left off.
		CrosslinkRecords:           newCrossLinkRecords,
		DynastySeedLastReset:       c.data.DynastySeedLastReset, // TODO(#267): Stub. Need to implement dynasty transition.
		TotalDeposits:              nextCycleBalance,
	})

	return newCrystallizedState, nil
}

func (c *CrystallizedState) calculateNewShardAndCommitteesForSlots() ([]*pb.ShardAndCommitteeArray, error) {
	newCommittees, err := casper.ShuffleValidatorsToCommittees(
		c.DynastySeed(),
		c.Validators(),
		c.CurrentDynasty(),
		c.CrosslinkingStartShard(),
	)
	if err != nil {
		return nil, err
	}

	return append(c.data.ShardAndCommitteesForSlots[:params.CycleLength], newCommittees...), nil
}

type shardAttestation struct {
	shardID        uint64
	shardBlockHash [32]byte
}

func copyCrosslinks(existing []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {
	new := make([]*pb.CrosslinkRecord, len(existing))
	for i := 0; i < len(existing); i++ {
		oldCL := existing[i]
		newBlockhash := make([]byte, len(oldCL.Blockhash))
		copy(newBlockhash, oldCL.Blockhash)
		newCL := &pb.CrosslinkRecord{
			Dynasty:   oldCL.Dynasty,
			Blockhash: newBlockhash,
			Slot:      oldCL.Slot,
		}
		new[i] = newCL
	}

	return new
}

// processCrosslinks checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest dynasty and slot numbers.
func (c *CrystallizedState) processCrosslinks(pendingAttestations []*pb.AttestationRecord, slot uint64) ([]*pb.CrosslinkRecord, error) {
	validators := c.data.Validators
	dynasty := c.data.CurrentDynasty
	crosslinkRecords := copyCrosslinks(c.data.CrosslinkRecords)

	shardAttestationBalance := map[shardAttestation]uint64{}
	for _, attestation := range pendingAttestations {
		indices, err := c.getAttesterIndices(attestation)
		if err != nil {
			return nil, err
		}

		shardBlockHash := [32]byte{}
		copy(shardBlockHash[:], attestation.ShardBlockHash)
		sa := shardAttestation{
			shardID:        attestation.ShardId,
			shardBlockHash: shardBlockHash,
		}
		if _, ok := shardAttestationBalance[sa]; !ok {
			shardAttestationBalance[sa] = 0
		}

		// find the total balance of the shard committee.
		var totalBalance uint64
		for _, attesterIndex := range indices {
			totalBalance += validators[attesterIndex].Balance
		}

		// find the balance of votes cast in shard attestation.
		var voteBalance uint64
		for i, attesterIndex := range indices {
			if utils.CheckBit(attestation.AttesterBitfield, i) {
				voteBalance += validators[attesterIndex].Balance
			}
		}
		shardAttestationBalance[sa] += voteBalance

		// if 2/3 of committee voted on this crosslink, update the crosslink
		// with latest dynasty number, shard block hash, and slot number.
		if 3*voteBalance >= 2*totalBalance && dynasty > crosslinkRecords[attestation.ShardId].Dynasty {
			crosslinkRecords[attestation.ShardId] = &pb.CrosslinkRecord{
				Dynasty:   dynasty,
				Blockhash: attestation.ShardBlockHash,
				Slot:      slot,
			}
		}
	}
	return crosslinkRecords, nil
}
