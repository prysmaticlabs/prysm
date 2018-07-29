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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

var stateLookupKey = "beaconchainstate"

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
	has, err := db.Has([]byte(stateLookupKey))
	if err != nil {
		return nil, err
	}
	if !has {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		active, crystallized := types.NewGenesisStates()
		beaconChain.state.ActiveState = active
		beaconChain.state.CrystallizedState = crystallized
		return beaconChain, nil
	}
	enc, err := db.Get([]byte(stateLookupKey))
	if err != nil {
		return nil, err
	}
	// Deserializes the encoded object into a beacon chain.
	if err := rlp.DecodeBytes(enc, &beaconChain.state); err != nil {
		return nil, fmt.Errorf("could not deserialize chainstate from disk: %v", err)
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

// ActiveValidatorCount exposes a getter to total number of active validator.
func (b *BeaconChain) ActiveValidatorCount() int {
	return len(b.state.CrystallizedState.ActiveValidators)
}

// QueuedValidatorCount exposes a getter to total number of queued validator.
func (b *BeaconChain) QueuedValidatorCount() int {
	return len(b.state.CrystallizedState.QueuedValidators)
}

// ExitedValidatorCount exposes a getter to total number of exited validator.
func (b *BeaconChain) ExitedValidatorCount() int {
	return len(b.state.CrystallizedState.ExitedValidators)
}

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	return types.NewGenesisBlock()
}

// isEpochTransition checks if the current slotNumber divided by the epoch length(64 slots)
// is greater than the current epoch.
func (b *BeaconChain) isEpochTransition(slotNumber uint64) bool {
	currentEpoch := b.state.CrystallizedState.CurrentEpoch
	isTransition := (slotNumber / params.EpochLength) > currentEpoch
	return isTransition
}

// MutateActiveState allows external services to modify the active state.
func (b *BeaconChain) MutateActiveState(activeState *types.ActiveState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState = activeState
	return b.persist()
}

// MutateCrystallizedState allows external services to modify the crystallized state.
func (b *BeaconChain) MutateCrystallizedState(crystallizedState *types.CrystallizedState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.CrystallizedState = crystallizedState
	return b.persist()
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
	slotDuration := time.Duration(block.SlotNumber()*params.SlotLength) * time.Second
	genesis, err := b.GenesisBlock()
	if err != nil {
		return false, err
	}

	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return false, err
	}

	// Verify state hashes from the block are correct
	hash, err := hashActiveState(b.ActiveState())
	if err != nil {
		return false, err
	}

	blockActiveStateHash := block.ActiveStateHash()

	if blockActiveStateHash != hash {
		return false, fmt.Errorf("active state hash mismatched, wanted: %v, got: %v", blockActiveStateHash, hash)
	}

	hash, err = hashCrystallizedState(b.CrystallizedState())
	if err != nil {
		return false, err
	}

	blockCrystallizedStateHash := block.CrystallizedStateHash()

	if blockCrystallizedStateHash != hash {
		return false, fmt.Errorf("crystallized state hash mismatched, wanted: %v, got: %v", blockCrystallizedStateHash, hash)
	}

	validTime := time.Now().After(genesisTime.Add(slotDuration))

	return validTime, nil
}

// RotateValidatorSet is called  every dynasty transition. It's primary function is
// to go through queued validators and induct them to be active, and remove bad
// active validator whose balance is below threshold to the exit set. It also cross checks
// every validator's switch dynasty before induct or remove.
func (b *BeaconChain) RotateValidatorSet() ([]types.ValidatorRecord, []types.ValidatorRecord, []types.ValidatorRecord) {

	var newExitedValidators = b.CrystallizedState().ExitedValidators
	var newActiveValidators []types.ValidatorRecord
	upperbound := b.ActiveValidatorCount()/30 + 1
	exitCount := 0

	// Loop through active validator set, remove validator whose balance is below 50% and switch dynasty > current dynasty.
	for _, validator := range b.state.CrystallizedState.ActiveValidators {
		if validator.Balance < params.DefaultBalance/2 {
			newExitedValidators = append(newExitedValidators, validator)
		} else if validator.SwitchDynasty == b.CrystallizedState().Dynasty+1 && exitCount < upperbound {
			newExitedValidators = append(newExitedValidators, validator)
			exitCount++
		} else {
			newActiveValidators = append(newActiveValidators, validator)
		}
	}
	// Get the total number of validator we can induct.
	inductNum := upperbound
	if b.QueuedValidatorCount() < inductNum {
		inductNum = b.QueuedValidatorCount()
	}

	// Induct queued validator to active validator set until the switch dynasty is greater than current number.
	for i := 0; i < inductNum; i++ {
		if b.CrystallizedState().QueuedValidators[i].SwitchDynasty > b.CrystallizedState().Dynasty+1 {
			inductNum = i
			break
		}
		newActiveValidators = append(newActiveValidators, b.CrystallizedState().QueuedValidators[i])
	}
	newQueuedValidators := b.CrystallizedState().QueuedValidators[inductNum:]

	return newQueuedValidators, newActiveValidators, newExitedValidators
}

// persist stores the RLP encoding of the latest beacon chain state into the db.
func (b *BeaconChain) persist() error {
	encodedState, err := rlp.EncodeToBytes(b.state)
	if err != nil {
		return err
	}
	return b.db.Put([]byte(stateLookupKey), encodedState)
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

	return &types.ActiveState{
		TotalAttesterDeposits: 0,
		AttesterBitfields:     []byte{},
	}, nil
}

// hashActiveState serializes the active state object then uses
// blake2b to hash the serialized object.
func hashActiveState(state *types.ActiveState) ([32]byte, error) {
	serializedState, err := rlp.EncodeToBytes(state)
	if err != nil {
		return [32]byte{}, err
	}
	return blake2b.Sum256(serializedState), nil
}

// hashCrystallizedState serializes the crystallized state object
// then uses blake2b to hash the serialized object.
func hashCrystallizedState(state *types.CrystallizedState) ([32]byte, error) {
	serializedState, err := rlp.EncodeToBytes(state)
	if err != nil {
		return [32]byte{}, err
	}
	return blake2b.Sum256(serializedState), nil
}

// getAttestersProposer returns lists of random sampled attesters and proposer indices.
func (b *BeaconChain) getAttestersProposer(seed common.Hash) ([]int, int, error) {
	attesterCount := math.Min(params.AttesterCount, float64(len(b.CrystallizedState().ActiveValidators)))
	indices, err := utils.ShuffleIndices(seed, len(b.CrystallizedState().ActiveValidators))
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

	if voted {
		b.state.CrystallizedState.ActiveValidators[index].Balance += params.AttesterReward
	} else {
		// TODO : Change this when penalties are specified for not voting
		b.state.CrystallizedState.ActiveValidators[index].Balance -= params.AttesterReward
	}

	return b.persist()
}

// resetAttesterBitfields resets the attester bitfields in the ActiveState to zero.
func (b *BeaconChain) resetAttesterBitfields() error {

	length := int(len(b.state.CrystallizedState.ActiveValidators) / 8)
	if len(b.state.CrystallizedState.ActiveValidators)%8 != 0 {
		length += 1
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	newbitfields := make([]byte, length)
	b.state.ActiveState.AttesterBitfields = newbitfields

	return b.persist()
}

// resetTotalDeposit clears and resets the total attester deposit to zero.
func (b *BeaconChain) resetTotalAttesterDeposit() error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState.TotalAttesterDeposits = 0

	return b.persist()
}

// setJustifiedEpoch sets the justified epoch during an epoch transition.
func (b *BeaconChain) updateJustifiedEpoch() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	justifiedEpoch := b.state.CrystallizedState.LastJustifiedEpoch
	b.state.CrystallizedState.LastJustifiedEpoch = b.state.CrystallizedState.CurrentEpoch

	if b.state.CrystallizedState.CurrentEpoch == (justifiedEpoch + 1) {
		b.state.CrystallizedState.LastFinalizedEpoch = justifiedEpoch
	}

	return b.persist()
}

// setRewardsAndPenalties checks if the attester has voted and then applies the
// rewards and penalties for them.
func (b *BeaconChain) updateRewardsAndPenalties(index int) error {
	bitfields := b.state.ActiveState.AttesterBitfields
	attesterBlock := (index + 1) / 8
	attesterFieldIndex := (index + 1) % 8
	if attesterFieldIndex == 0 {
		attesterFieldIndex = 8
	} else {
		attesterBlock += 1
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

// Slashing Condtions
// TODO: Implement all the conditions and add in the methods once the spec is updated

// computeValidatorRewardsAndPenalties is run every epoch transition and appropriates the
// rewards and penalties, resets the bitfield and deposits and also applies the slashing conditions.
func (b *BeaconChain) computeValidatorRewardsAndPenalties() error {
	activeValidatorSet := b.state.CrystallizedState.ActiveValidators
	attesterDeposits := b.state.ActiveState.TotalAttesterDeposits
	totalDeposit := b.state.CrystallizedState.TotalDeposits

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

			/*			if err := b.applySlashingConditions(i); err != nil {
						log.Error(err)
					}*/
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
