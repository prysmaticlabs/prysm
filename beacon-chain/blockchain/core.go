package blockchain

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"golang.org/x/crypto/blake2s"
	"math"
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
	enc, err := db.Get([]byte(stateLookupKey))
	if err != nil && err.Error() == leveldberrors.ErrNotFound.Error() {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		active, crystallized := types.NewGenesisStates()
		beaconChain.state.ActiveState = active
		beaconChain.state.CrystallizedState = crystallized
		return beaconChain, nil
	}
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

// MutateActiveState allows external services to modify the active state.
func (b *BeaconChain) MutateActiveState(activeState *types.ActiveState) error {
	defer b.lock.Unlock()
	b.lock.Lock()
	b.state.ActiveState = activeState
	return b.persist()
}

// MutateCrystallizedState allows external services to modify the crystallized state.
func (b *BeaconChain) MutateCrystallizedState(crystallizedState *types.CrystallizedState) error {
	defer b.lock.Unlock()
	b.lock.Lock()
	b.state.CrystallizedState = crystallizedState
	return b.persist()
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

	_, _, err := b.getAttestersProposer(seed)
	if err != nil {
		return nil, err
	}
	// TODO: Verify attestations from attesters

	// TODO: Verify main signature from proposer

	// TODO: Update crosslink records (post Ruby release)

	// TODO: Track reward for the proposer that just proposed the latest beacon block

	// TODO: Verify randao reveal from validator's hash pre image

	return &types.ActiveState{
		TotalAttesterDeposits: 0,
		AttesterBitfields:     []byte{},
	}, nil
}

// getAttestersProposer returns lists of random sampled attesters and proposer indices.
func (b *BeaconChain) getAttestersProposer(seed common.Hash) ([]int, int, error) {
	attesterCount := math.Min(params.AttesterCount, float64(len(b.CrystallizedState().ActiveValidators)))
	indices, err := Shuffle(seed, len(b.CrystallizedState().ActiveValidators))
	if err != nil {
		return nil, -1, err
	}
	return indices[:int(attesterCount)], indices[len(indices)-1], nil
}

// Shuffle returns a list of pseudorandomly sampled
// indices to use to select attesters and proposers.
func Shuffle(seed common.Hash, validatorCount int) ([]int, error) {
	if validatorCount > params.MaxValidators {
		return nil, errors.New("Validator count has exceeded MaxValidator Count")
	}

	// construct a list of indices up to MaxValidators
	validatorList := make([]int, validatorCount)
	for i := range validatorList {
		validatorList[i] = i
	}

	hashSeed, err := blake2s.New256(seed[:])
	if err != nil {
		return nil, err
	}

	hashSeedByte := hashSeed.Sum(nil)

	// shuffle stops at the second to last index
	for i := 0; i < validatorCount-1; i++ {
		// convert every 3 bytes to random number, replace validator index with that number
		for j := 0; j+3 < len(hashSeedByte); j += 3 {
			swapNum := int(hashSeedByte[j] + hashSeedByte[j+1] + hashSeedByte[j+2])
			remaining := validatorCount - i
			swapPos := swapNum%remaining + i
			validatorList[i], validatorList[swapPos] = validatorList[swapPos], validatorList[i]
		}
	}
	return validatorList, nil
}