package blockchain

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/sirupsen/logrus"
)

var activeStateLookupKey = "beacon-active-state"
var crystallizedStateLookupKey = "beacon-crystallized-state"

// BeaconChain represents the core PoS blockchain object containing
// both a crystallized and active state.
type BeaconChain struct {
	state *beaconState
	lock  sync.Mutex
	db    ethdb.Database
}

type beaconState struct {
	ActiveState       *types.ActiveState
	CrystallizedState *types.CrystallizedState
}

// NewBeaconChain initializes an instance using genesis state parameters if
// none provided.
func NewBeaconChain(db ethdb.Database) (*BeaconChain, error) {
	beaconChain := &BeaconChain{
		db:    db,
		state: &beaconState{},
	}
	hasActive, err := db.Has([]byte(activeStateLookupKey))
	if err != nil {
		return nil, err
	}
	hasCrystallized, err := db.Has([]byte(crystallizedStateLookupKey))
	if err != nil {
		return nil, err
	}
	if !hasActive && !hasCrystallized {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		active, crystallized := types.NewGenesisStates()
		beaconChain.state.ActiveState = active
		beaconChain.state.CrystallizedState = crystallized
		return beaconChain, nil
	}

	if hasActive {
		enc, err := db.Get([]byte(activeStateLookupKey))
		if err != nil {
			return nil, err
		}

		activeData := &pb.ActiveStateResponse{}
		err = proto.Unmarshal(enc, activeData)
		if err != nil {
			return nil, err
		}
		beaconChain.state.ActiveState = types.NewActiveState(activeData)
	}
	if hasCrystallized {
		enc, err := db.Get([]byte(crystallizedStateLookupKey))
		if err != nil {
			return nil, err
		}

		crystallizedData := &pb.CrystallizedStateResponse{}
		err = proto.Unmarshal(enc, crystallizedData)
		if err != nil {
			return nil, err
		}
		beaconChain.state.CrystallizedState = types.NewCrystallizedState(crystallizedData)
	}
	return beaconChain, nil
}

// ActiveState exposes a getter to external services.
func (b *BeaconChain) ActiveState() *types.ActiveState {
	return b.state.ActiveState
}

// CrystallizedState exposes a getter to external services.
func (b *BeaconChain) CrystallizedState() *types.CrystallizedState {
	return b.state.CrystallizedState
}

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	return types.NewGenesisBlock()
}

// isEpochTransition checks if the current slotNumber divided by the epoch length(64 slots)
// is greater than the current epoch.
func (b *BeaconChain) isEpochTransition(slotNumber uint64) bool {
	currentEpoch := b.state.CrystallizedState.CurrentEpoch()
	isTransition := (slotNumber / params.EpochLength) > currentEpoch
	return isTransition
}

// MutateActiveState allows external services to modify the active state.
func (b *BeaconChain) MutateActiveState(activeState *types.ActiveState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState = activeState
	return b.persistActiveState()
}

// MutateCrystallizedState allows external services to modify the crystallized state.
func (b *BeaconChain) MutateCrystallizedState(crystallizedState *types.CrystallizedState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.CrystallizedState = crystallizedState
	return b.persistCrystallizedState()
}

// CanProcessBlock decides if an incoming p2p block can be processed into the chain's block trie.
func (b *BeaconChain) CanProcessBlock(fetcher types.POWBlockFetcher, block *types.Block) (bool, error) {
	mainchainBlock, err := fetcher.BlockByHash(context.Background(), block.MainChainRef())
	if err != nil {
		return false, err
	}
	// There needs to be a valid mainchain block for the reference hash in a beacon block.
	if mainchainBlock == nil {
		return false, nil
	}
	// TODO: check if the parentHash pointed by the beacon block is in the beaconDB.

	// Calculate the timestamp validity condition.
	slotDuration := time.Duration(block.SlotNumber()*params.SlotDuration) * time.Second
	genesis, err := b.GenesisBlock()
	if err != nil {
		return false, err
	}

	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return false, err
	}

	if time.Now().Before(genesisTime.Add(slotDuration)) {
		return false, nil
	}

	// Verify state hashes from the block are correct
	hash, err := b.ActiveState().Hash()
	if err != nil {
		return false, err
	}

	if block.ActiveStateHash() != hash {
		return false, fmt.Errorf("active state hash mismatched, wanted: %v, got: %v", block.ActiveStateHash(), hash)
	}

	hash, err = b.CrystallizedState().Hash()
	if err != nil {
		return false, err
	}

	if block.CrystallizedStateHash() != hash {
		return false, fmt.Errorf("crystallized state hash mismatched, wanted: %v, got: %v", block.CrystallizedStateHash(), hash)
	}

	return true, nil
}

// RotateValidatorSet is called  every dynasty transition. It's primary function is
// to go through queued validators and induct them to be active, and remove bad
// active validator whose balance is below threshold to the exit set. It also cross checks
// every validator's switch dynasty before induct or remove.
func (b *BeaconChain) RotateValidatorSet() ([]*pb.ValidatorRecord, []*pb.ValidatorRecord, []*pb.ValidatorRecord) {

	var newExitedValidators = b.CrystallizedState().ExitedValidators()
	var newActiveValidators []*pb.ValidatorRecord
	upperbound := b.CrystallizedState().ActiveValidatorsLength()/30 + 1
	exitCount := 0

	// Loop through active validator set, remove validator whose balance is below 50% and switch dynasty > current dynasty.
	for _, validator := range b.CrystallizedState().ActiveValidators() {
		if validator.Balance < params.DefaultBalance/2 {
			newExitedValidators = append(newExitedValidators, validator)
		} else if validator.SwitchDynasty == b.CrystallizedState().CurrentDynasty()+1 && exitCount < upperbound {
			newExitedValidators = append(newExitedValidators, validator)
			exitCount++
		} else {
			newActiveValidators = append(newActiveValidators, validator)
		}
	}
	// Get the total number of validator we can induct.
	inductNum := upperbound
	if b.CrystallizedState().QueuedValidatorsLength() < inductNum {
		inductNum = b.CrystallizedState().QueuedValidatorsLength()
	}

	// Induct queued validator to active validator set until the switch dynasty is greater than current number.
	for i := 0; i < inductNum; i++ {
		if b.CrystallizedState().QueuedValidators()[i].SwitchDynasty > b.CrystallizedState().CurrentDynasty()+1 {
			inductNum = i
			break
		}
		newActiveValidators = append(newActiveValidators, b.CrystallizedState().QueuedValidators()[i])
	}
	newQueuedValidators := b.CrystallizedState().QueuedValidators()[inductNum:]

	return newQueuedValidators, newActiveValidators, newExitedValidators
}

// persistActiveState stores proto encoding of the latest beacon chain active state into the db.
func (b *BeaconChain) persistActiveState() error {
	encodedState, err := b.ActiveState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(activeStateLookupKey), encodedState)
}

// persistCrystallizedState stores proto encoding of the latest beacon chain crystallized state into the db.
func (b *BeaconChain) persistCrystallizedState() error {
	encodedState, err := b.CrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(crystallizedStateLookupKey), encodedState)
}

// computeNewActiveState computes a new active state for every beacon block.
func (b *BeaconChain) computeNewActiveState(seed common.Hash) (*types.ActiveState, error) {
	attesters, proposer, err := b.getAttestersProposer(seed)
	if err != nil {
		return nil, err
	}
	// TODO: Verify attestations from attesters.
	log.WithFields(logrus.Fields{"attestersIndices": attesters}).Debug("Attester indices")

	// TODO: Verify main signature from proposer.
	log.WithFields(logrus.Fields{"proposerIndex": proposer}).Debug("Proposer index")

	// TODO: Update crosslink records (post Ruby release).

	// TODO: Track reward for the proposer that just proposed the latest beacon block.

	// TODO: Verify randao reveal from validator's hash pre image.

	return types.NewActiveState(&pb.ActiveStateResponse{
		TotalAttesterDeposits: 0,
		AttesterBitfield:      []byte{},
	}), nil
}

// getAttestersProposer returns lists of random sampled attesters and proposer indices.
func (b *BeaconChain) getAttestersProposer(seed common.Hash) ([]int, int, error) {
	attesterCount := math.Min(params.AttesterCount, float64(b.CrystallizedState().ActiveValidatorsLength()))

	indices, err := utils.ShuffleIndices(seed, b.CrystallizedState().ActiveValidatorsLength())
	if err != nil {
		return nil, -1, err
	}
	return indices[:int(attesterCount)], indices[len(indices)-1], nil
}

// GetCutoffs is used to split up validators into groups at the start
// of every epoch. It determines at what height validators can make
// attestations and crosslinks. It returns lists of cutoff indices.
func GetCutoffs(validatorCount int) []int {
	var heightCutoff = []int{0}
	var heights []int
	var heightCount float64

	// Skip heights if there's not enough validators to fill in a min sized committee.
	if validatorCount < params.EpochLength*params.MinCommiteeSize {
		heightCount = math.Floor(float64(validatorCount) / params.MinCommiteeSize)
		for i := 0; i < int(heightCount); i++ {
			heights = append(heights, (i*params.Cofactor)%params.EpochLength)
		}
		// Enough validators, fill in all the heights.
	} else {
		heightCount = params.EpochLength
		for i := 0; i < int(heightCount); i++ {
			heights = append(heights, i)
		}
	}

	filled := 0
	appendHeight := false
	for i := 0; i < params.EpochLength-1; i++ {
		appendHeight = false
		for _, height := range heights {
			if i == height {
				appendHeight = true
			}
		}
		if appendHeight {
			filled++
			heightCutoff = append(heightCutoff, filled*validatorCount/int(heightCount))
		} else {
			heightCutoff = append(heightCutoff, heightCutoff[len(heightCutoff)-1])
		}
	}
	heightCutoff = append(heightCutoff, validatorCount)

	// TODO: For the validators assigned to each height, split them up into
	// committees for different shards. Do not assign the last END_EPOCH_GRACE_PERIOD
	// heights in a epoch to any shards.
	return heightCutoff
}

// hasVoted checks if the attester has voted by looking at the bitfield.
func hasVoted(bitfields []byte, attesterBlock int, attesterFieldIndex int) bool {
	voted := false

	fields := bitfields[attesterBlock-1]
	attesterField := fields >> (8 - uint(attesterFieldIndex))
	if attesterField%2 != 0 {
		voted = true
	}

	return voted
}

// applyRewardAndPenalty applies the appropriate rewards and penalties according to
// whether the attester has voted or not.
func (b *BeaconChain) applyRewardAndPenalty(index int, voted bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	validators := b.CrystallizedState().ActiveValidators()

	if voted {
		validators[index].Balance += params.AttesterReward
	} else {
		// TODO : Change this when penalties are specified for not voting
		validators[index].Balance -= params.AttesterReward
	}
	b.CrystallizedState().UpdateActiveValidators(validators)
	return b.persistCrystallizedState()
}

// resetAttesterBitfields resets the attester bitfields in the ActiveState to zero.
func (b *BeaconChain) resetAttesterBitfields() error {

	length := int(b.CrystallizedState().ActiveValidatorsLength() / 8)
	if b.CrystallizedState().ActiveValidatorsLength()%8 != 0 {
		length++
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	newbitfields := make([]byte, length)
	b.state.ActiveState.SetAttesterBitfield(newbitfields)

	return b.persistCrystallizedState()
}

// resetTotalAttesterDeposit clears and resets the total attester deposit to zero.
func (b *BeaconChain) resetTotalAttesterDeposit() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState.SetTotalAttesterDeposits(0)

	return b.persistCrystallizedState()
}

// updateJustifiedEpoch updates the justified epoch during an epoch transition.
func (b *BeaconChain) updateJustifiedEpoch() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	justifiedEpoch := b.state.CrystallizedState.LastJustifiedEpoch()
	b.state.CrystallizedState.SetLastJustifiedEpoch(b.state.CrystallizedState.CurrentEpoch())

	if b.state.CrystallizedState.CurrentEpoch() == (justifiedEpoch + 1) {
		b.state.CrystallizedState.SetLastFinalizedEpoch(justifiedEpoch)
	}

	return b.persistCrystallizedState()
}

// updateRewardsAndPenalties checks if the attester has voted and then applies the
// rewards and penalties for them.
func (b *BeaconChain) updateRewardsAndPenalties(index int) error {
	bitfields := b.state.ActiveState.AttesterBitfield()
	attesterBlock := (index + 1) / 8
	attesterFieldIndex := (index + 1) % 8
	if attesterFieldIndex == 0 {
		attesterFieldIndex = 8
	} else {
		attesterBlock++
	}

	if len(bitfields) < attesterBlock {
		return errors.New("attester index does not exist")
	}

	voted := hasVoted(bitfields, attesterBlock, attesterFieldIndex)
	if err := b.applyRewardAndPenalty(index, voted); err != nil {
		return fmt.Errorf("unable to apply rewards and penalties: %v", err)
	}

	return nil
}

// computeValidatorRewardsAndPenalties is run every epoch transition and appropriates the
// rewards and penalties, resets the bitfield and deposits and also applies the slashing conditions.
func (b *BeaconChain) computeValidatorRewardsAndPenalties() error {
	activeValidatorSet := b.state.CrystallizedState.ActiveValidators()
	attesterDeposits := b.state.ActiveState.TotalAttesterDeposits()
	totalDeposit := b.state.CrystallizedState.TotalDeposits()

	attesterFactor := attesterDeposits * 3
	totalFactor := uint64(totalDeposit * 2)

	if attesterFactor >= totalFactor {
		log.Info("Justified epoch in the crystallised state is set to the current epoch")

		if err := b.updateJustifiedEpoch(); err != nil {
			return fmt.Errorf("error setting justified epoch: %v", err)
		}

		for i := range activeValidatorSet {
			if err := b.updateRewardsAndPenalties(i); err != nil {
				log.Error(err)
			}
		}

		if err := b.resetAttesterBitfields(); err != nil {
			return fmt.Errorf("error resetting bitfields: %v", err)
		}
		if err := b.resetTotalAttesterDeposit(); err != nil {
			return fmt.Errorf("error resetting total deposits: %v", err)
		}
	}
	return nil
}

// Slashing Condtions
// TODO: Implement all the conditions and add in the methods once the spec is updated
