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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	return types.NewGenesisBlock()
}

// ActiveState exposes a getter to external services.
func (b *BeaconChain) ActiveState() *types.ActiveState {
	return b.state.ActiveState
}

// CrystallizedState exposes a getter to external services.
func (b *BeaconChain) CrystallizedState() *types.CrystallizedState {
	return b.state.CrystallizedState
}

// MutateActiveState allows external services to modify the active state.
func (b *BeaconChain) MutateActiveState(activeState *types.ActiveState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState = activeState
	return b.PersistActiveState()
}

// MutateCrystallizedState allows external services to modify the crystallized state.
func (b *BeaconChain) MutateCrystallizedState(crystallizedState *types.CrystallizedState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.CrystallizedState = crystallizedState
	return b.PersistCrystallizedState()
}

// PersistActiveState stores proto encoding of the latest beacon chain active state into the db.
func (b *BeaconChain) PersistActiveState() error {
	encodedState, err := b.ActiveState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(activeStateLookupKey), encodedState)
}

// PersistCrystallizedState stores proto encoding of the latest beacon chain crystallized state into the db.
func (b *BeaconChain) PersistCrystallizedState() error {
	encodedState, err := b.CrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(crystallizedStateLookupKey), encodedState)
}

// IsEpochTransition checks if the current slotNumber divided by the epoch length(64 slots)
// is greater than the current epoch.
func (b *BeaconChain) IsEpochTransition(slotNumber uint64) bool {
	currentEpoch := b.state.CrystallizedState.CurrentEpoch()
	isTransition := (slotNumber / params.EpochLength) > currentEpoch
	return isTransition
}

// CanProcessBlock decides if an incoming p2p block can be processed into the chain's block trie.
func (b *BeaconChain) CanProcessBlock(fetcher types.POWBlockFetcher, block *types.Block) (bool, error) {
	if _, err := fetcher.BlockByHash(context.Background(), block.MainChainRef()); err != nil {
		return false, fmt.Errorf("fetching PoW block corresponding to mainchain reference failed: %v", err)
	}

	// Check if the parentHash pointed by the beacon block is in the beaconDB.
	parentHash := block.ParentHash()
	val, err := b.db.Get(parentHash[:])
	if err != nil {
		return false, err
	}
	if val == nil {
		return false, errors.New("parent hash points to nil in beaconDB")
	}

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

// rotateValidatorSet is called every dynasty transition. It's primary function is
// to go through queued validators and induct them to be active, and remove bad
// active validator whose balance is below threshold to the exit set. It also cross checks
// every validator's switch dynasty before induct or remove.
func (b *BeaconChain) rotateValidatorSet() ([]*pb.ValidatorRecord, []*pb.ValidatorRecord, []*pb.ValidatorRecord) {

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

// getAttestersProposer returns lists of random sampled attesters and proposer indices.
func (b *BeaconChain) getAttestersProposer(seed common.Hash) ([]int, int, error) {
	attesterCount := math.Min(params.AttesterCount, float64(b.CrystallizedState().ActiveValidatorsLength()))

	indices, err := utils.ShuffleIndices(seed, b.CrystallizedState().ActiveValidatorsLength())
	if err != nil {
		return nil, -1, err
	}
	return indices[:int(attesterCount)], indices[len(indices)-1], nil
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
	activeValidators := b.state.CrystallizedState.ActiveValidators()
	attesterDeposits := b.state.ActiveState.TotalAttesterDeposits()
	totalDeposit := b.state.CrystallizedState.TotalDeposits()

	attesterFactor := attesterDeposits * 3
	totalFactor := uint64(totalDeposit * 2)

	if attesterFactor >= totalFactor {
		log.Info("Setting justified epoch to current epoch: %v", b.CrystallizedState().CurrentEpoch())
		b.state.CrystallizedState.UpdateJustifiedEpoch()

		log.Info("Applying rewards and penalties for the validators from last epoch")
		for i := range activeValidators {
			voted, err := b.voted(i)
			if err != nil {
				return fmt.Errorf("exiting calculate rewards FFG due to %v", err)
			}
			if voted {
				activeValidators[i].Balance += params.AttesterReward
			} else {
				activeValidators[i].Balance -= params.AttesterReward
			}
		}

		log.Info("Resetting attester bit field to all zeros")
		b.resetAttesterBitfield()

		log.Info("Resetting total attester deposit to zero")
		b.ActiveState().SetTotalAttesterDeposits(0)

		b.CrystallizedState().UpdateActiveValidators(activeValidators)
		err := b.PersistActiveState()
		if err != nil {
			return err
		}
		err = b.PersistCrystallizedState()
		if err != nil {
			return err
		}
	}
	return nil
}

// voted checks if a validator has voted by comparing its bit field.
func (b *BeaconChain) voted(index int) (bool, error) {
	bitfield := b.state.ActiveState.AttesterBitfield()
	attesterBlock := (index + 1) / 8
	attesterFieldIndex := (index + 1) % 8
	if attesterFieldIndex == 0 {
		attesterFieldIndex = 8
	} else {
		attesterBlock++
	}

	if len(bitfield) < attesterBlock {
		return false, errors.New("attester index does not exist")
	}

	field := bitfield[attesterBlock-1] >> (8 - uint(attesterFieldIndex))
	if field%2 != 0 {
		return true, nil
	}

	return false, nil
}

// resetAttesterBitfield resets the attester bit field of active state to zeros.
func (b *BeaconChain) resetAttesterBitfield() {
	newbitfields := make([]byte, b.CrystallizedState().ActiveValidatorsLength()/8)
	b.state.ActiveState.SetAttesterBitfield(newbitfields)
}
