package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
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
	mainchainBlock, err := fetcher.BlockByHash(context.Background(), block.Data().MainChainRef)
	if err != nil {
		return false, err
	}
	// There needs to be a valid mainchain block for the reference hash in a beacon block.
	if mainchainBlock == nil {
		return false, nil
	}
	// TODO: check if the parentHash pointed by the beacon block is in the beaconDB.

	// Calculate the timestamp validity condition.
	slotDuration := time.Duration(block.Data().SlotNumber*params.SlotLength) * time.Second
	validTime := time.Now().After(b.GenesisBlock().Data().Timestamp.Add(slotDuration))

	// Verify state hashes from the block are correct
	hash, err := hashActiveState(*b.ActiveState())
	if err != nil {
		return false, err
	}

	if !bytes.Equal(block.Data().ActiveStateHash.Sum(nil), hash.Sum(nil)) {
		return false, fmt.Errorf("Active state hash mismatched, wanted: %v, got: %v", hash.Sum(nil), block.Data().ActiveStateHash.Sum(nil))
	}
	hash, err = hashCrystallizedState(*b.CrystallizedState())
	if err != nil {
		return false, err
	}
	if !bytes.Equal(block.Data().CrystallizedStateHash.Sum(nil), hash.Sum(nil)) {
		return false, fmt.Errorf("Crystallized state hash mismatched, wanted: %v, got: %v", hash.Sum(nil), block.Data().CrystallizedStateHash.Sum(nil))
	}

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
func hashActiveState(state types.ActiveState) (hash.Hash, error) {
	serializedState, err := rlp.EncodeToBytes(state)
	if err != nil {
		return nil, err
	}
	return blake2b.New256(serializedState)
}

// hashCrystallizedState serializes the crystallized state object
// then uses blake2b to hash the serialized object.
func hashCrystallizedState(state types.CrystallizedState) (hash.Hash, error) {
	serializedState, err := rlp.EncodeToBytes(state)
	if err != nil {
		return nil, err
	}
	return blake2b.New256(serializedState)
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
