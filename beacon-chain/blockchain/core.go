package blockchain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"golang.org/x/crypto/blake2s"
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

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() *types.Block {
	return types.NewGenesisBlock()
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

// CanProcessBlock decides if an incoming p2p block can be processed into the chain's block trie.
func (b *BeaconChain) CanProcessBlock(fetcher powchain.POWBlockFetcher, block *types.Block) (bool, error) {
	mainchainBlock, err := fetcher.BlockByHash(context.Background(), block.Header().MainChainRef)
	if err != nil {
		return false, err
	}
	// There needs to be a valid mainchain block for the reference hash in a beacon block.
	if mainchainBlock != nil {
		return false, nil
	}
	// TODO: check if the parentHash pointed by the beacon block is in the beaconDB.

	// Calculate the timestamp validity condition.
	slotDuration := time.Duration(block.Header().SlotNumber*params.SlotLength) * time.Second
	validTime := time.Now().After(b.GenesisBlock().Header().Timestamp.Add(slotDuration))
	return validTime, nil
}

// persist stores the RLP encoding of the latest beacon chain state into the db.
func (b *BeaconChain) persist() error {
	encodedState, err := rlp.EncodeToBytes(b.state)
	if err != nil {
		return err
	}
	return b.db.Put([]byte(stateLookupKey), encodedState)
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
