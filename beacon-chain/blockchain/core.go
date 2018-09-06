package blockchain

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// BeaconChain represents the core PoS blockchain object containing
// both a crystallized and active state.
type BeaconChain struct {
	state *beaconState
	lock  sync.Mutex
	db    ethdb.Database
}

type beaconState struct {
	// ActiveState captures the beacon state at block processing level,
	// it focuses on verifying aggregated signatures and pending attestations.
	ActiveState *types.ActiveState
	// CrystallizedState captures the beacon state at cycle transition level,
	// it focuses on changes to the validator set, processing cross links and
	// setting up FFG checkpoints.
	CrystallizedState *types.CrystallizedState
}

// NewBeaconChain initializes a beacon chain using genesis state parameters if
// none provided.
func NewBeaconChain(db ethdb.Database) (*BeaconChain, error) {
	beaconChain := &BeaconChain{
		db:    db,
		state: &beaconState{},
	}
	hasCrystallized, err := db.Has(crystallizedStateLookupKey)
	if err != nil {
		return nil, err
	}
	hasGenesis, err := db.Has(genesisLookupKey)
	if err != nil {
		return nil, err
	}

	active, crystallized, err := types.NewGenesisStates()
	if err != nil {
		return nil, err
	}

	beaconChain.state.ActiveState = active

	if !hasGenesis {
		log.Info("No genesis block found on disk, initializing genesis block")
		genesisBlock, err := types.NewGenesisBlock()
		if err != nil {
			return nil, err
		}
		genesisMarshall, err := proto.Marshal(genesisBlock.Proto())
		if err != nil {
			return nil, err
		}
		if err := beaconChain.db.Put(genesisLookupKey, genesisMarshall); err != nil {
			return nil, err
		}
	}
	if !hasCrystallized {
		log.Info("No chainstate found on disk, initializing beacon from genesis")
		beaconChain.state.CrystallizedState = crystallized
		return beaconChain, nil
	}

	enc, err := db.Get(crystallizedStateLookupKey)
	if err != nil {
		return nil, err
	}
	crystallizedData := &pb.CrystallizedState{}
	err = proto.Unmarshal(enc, crystallizedData)
	if err != nil {
		return nil, err
	}
	beaconChain.state.CrystallizedState = types.NewCrystallizedState(crystallizedData)

	return beaconChain, nil
}

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	genesisExists, err := b.db.Has(genesisLookupKey)
	if err != nil {
		return nil, err
	}
	if genesisExists {
		bytes, err := b.db.Get(genesisLookupKey)
		if err != nil {
			return nil, err
		}
		block := &pb.BeaconBlock{}
		if err := proto.Unmarshal(bytes, block); err != nil {
			return nil, err
		}
		return types.NewBlock(block), nil
	}
	return types.NewGenesisBlock()
}

// CanonicalHead fetches the latest head stored in persistent storage.
func (b *BeaconChain) CanonicalHead() (*types.Block, error) {
	bytes, err := b.db.Get(canonicalHeadLookupKey)
	if err != nil {
		return nil, err
	}
	block := &pb.BeaconBlock{}
	if err := proto.Unmarshal(bytes, block); err != nil {
		return nil, fmt.Errorf("cannot unmarshal proto: %v", err)
	}
	return types.NewBlock(block), nil

}

// ActiveState contains the current state of attestations and changes every block.
func (b *BeaconChain) ActiveState() *types.ActiveState {
	return b.state.ActiveState
}

// CrystallizedState contains cycle dependent validator information, changes every cycle.
func (b *BeaconChain) CrystallizedState() *types.CrystallizedState {
	return b.state.CrystallizedState
}

// SetActiveState is a convenience method which sets and persists the active state on the beacon chain.
func (b *BeaconChain) SetActiveState(activeState *types.ActiveState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.ActiveState = activeState
	return b.PersistActiveState()
}

// SetCrystallizedState is a convenience method which sets and persists the crystallized state on the beacon chain.
func (b *BeaconChain) SetCrystallizedState(crystallizedState *types.CrystallizedState) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.state.CrystallizedState = crystallizedState
	return b.PersistCrystallizedState()
}

// PersistActiveState stores proto encoding of the current beacon chain active state into the db.
func (b *BeaconChain) PersistActiveState() error {
	encodedState, err := b.ActiveState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put(activeStateLookupKey, encodedState)
}

// PersistCrystallizedState stores proto encoding of the current beacon chain crystallized state into the db.
func (b *BeaconChain) PersistCrystallizedState() error {
	encodedState, err := b.CrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put(crystallizedStateLookupKey, encodedState)
}

func (b *BeaconChain) hasBlock(blockhash [32]byte) (bool, error) {
	return b.db.Has(blockKey(blockhash))
}

// saveBlock puts the passed block into the beacon chain db.
func (b *BeaconChain) saveBlock(block *types.Block) error {
	hash, err := block.Hash()
	if err != nil {
		return err
	}

	key := blockKey(hash)
	encodedState, err := block.Marshal()
	if err != nil {
		return err
	}
	return b.db.Put(key, encodedState)
}

// saveCanonicalSlotNumber saves the slotnumber and blockhash of a canonical block
// saved in the db. This will alow for canonical blocks to be retrieved from the db
// by using their slotnumber as a key, and then using the retrieved blockhash to get
// the block from the db.
// prefix + slotnumber -> blockhash
// prefix + blockhash -> block
func (b *BeaconChain) saveCanonicalSlotNumber(slotnumber uint64, hash [32]byte) error {
	return b.db.Put(canonicalBlockKey(slotnumber), hash[:])
}

// saveCanonicalBlock puts the passed block into the beacon chain db
// and also saves a "latest-head" key mapping to the block in the db.
func (b *BeaconChain) saveCanonicalBlock(block *types.Block) error {
	if err := b.saveBlock(block); err != nil {
		return err
	}
	enc, err := block.Marshal()
	if err != nil {
		return err
	}

	return b.db.Put(canonicalHeadLookupKey, enc)
}

// getBlock retrieves a block from the db using its hash.
func (b *BeaconChain) getBlock(hash [32]byte) (*types.Block, error) {
	key := blockKey(hash)
	enc, err := b.db.Get(key)
	if err != nil {
		return nil, err
	}

	block := &pb.BeaconBlock{}

	err = proto.Unmarshal(enc, block)

	return types.NewBlock(block), err
}

// removeBlock removes the block from the db.
func (b *BeaconChain) removeBlock(hash [32]byte) error {
	return b.db.Delete(blockKey(hash))
}

// hasCanonicalBlockForSlot checks the db if the canonical block for
// this slot exists.
func (b *BeaconChain) hasCanonicalBlockForSlot(slotnumber uint64) (bool, error) {
	return b.db.Has(canonicalBlockKey(slotnumber))
}

// getCanonicalBlockForSlot retrieves the canonical block which is saved in the db
// for that required slot number.
func (b *BeaconChain) getCanonicalBlockForSlot(slotNumber uint64) (*types.Block, error) {
	enc, err := b.db.Get(canonicalBlockKey(slotNumber))
	if err != nil {
		return nil, err
	}

	var blockhash [32]byte
	copy(blockhash[:], enc)

	block, err := b.getBlock(blockhash)

	return block, err
}
