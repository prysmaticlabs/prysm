package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var canonicalHeadKey = "latest-canonical-head"
var activeStateLookupKey = "beacon-active-state"
var crystallizedStateLookupKey = "beacon-crystallized-state"

var clock utils.Clock = &utils.RealClock{}

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
	hasActive, err := db.Has([]byte(activeStateLookupKey))
	if err != nil {
		return nil, err
	}
	hasCrystallized, err := db.Has([]byte(crystallizedStateLookupKey))
	if err != nil {
		return nil, err
	}
	hasGenesis, err := db.Has([]byte("genesis"))
	if err != nil {
		return nil, err
	}
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
		if err := beaconChain.db.Put([]byte("genesis"), genesisMarshall); err != nil {
			return nil, err
		}
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
		activeData := &pb.ActiveState{}
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
		crystallizedData := &pb.CrystallizedState{}
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
	genesisExists, err := b.db.Has([]byte("genesis"))
	if err != nil {
		return nil, err
	}
	if genesisExists {
		bytes, err := b.db.Get([]byte("genesis"))
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
	headExists, err := b.db.Has([]byte(canonicalHeadKey))
	if err != nil {
		return nil, err
	}
	if headExists {
		bytes, err := b.db.Get([]byte(canonicalHeadKey))
		if err != nil {
			return nil, err
		}
		block := &pb.BeaconBlock{}
		if err := proto.Unmarshal(bytes, block); err != nil {
			return nil, err
		}
		return types.NewBlock(block), nil
	}
	return nil, nil
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
	return b.db.Put([]byte(activeStateLookupKey), encodedState)
}

// PersistCrystallizedState stores proto encoding of the current beacon chain crystallized state into the db.
func (b *BeaconChain) PersistCrystallizedState() error {
	encodedState, err := b.CrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(crystallizedStateLookupKey), encodedState)
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new state transition will occur.
func (b *BeaconChain) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= b.CrystallizedState().LastStateRecalc()+params.CycleLength
}

// CanProcessBlock is called to decide if an incoming p2p block can be processed into the chain's block trie,
// it checks time stamp, beacon chain parent block hash. It also checks pow chain reference hash if it's a validator.
func (b *BeaconChain) CanProcessBlock(fetcher types.POWBlockFetcher, block *types.Block, isValidator bool) (bool, error) {
	if isValidator {
		if _, err := fetcher.BlockByHash(context.Background(), block.PowChainRef()); err != nil {
			return false, fmt.Errorf("fetching PoW block corresponding to mainchain reference failed: %v", err)
		}
	}

	canProcess, err := b.verifyBlockTimeStamp(block)
	if err != nil {
		return false, fmt.Errorf("unable to process block: %v", err)
	}
	if !canProcess {
		return false, fmt.Errorf("time stamp verification for beacon block %v failed", block.SlotNumber())
	}

	canProcess, err = b.verifyBlockActiveHash(block)
	if err != nil {
		return false, fmt.Errorf("unable to process block: %v", err)
	}
	if !canProcess {
		return false, fmt.Errorf("active state verification for beacon block %v failed", block.SlotNumber())
	}

	canProcess, err = b.verifyBlockCrystallizedHash(block)
	if err != nil {
		return false, fmt.Errorf("unable to process block: %v", err)
	}
	if !canProcess {
		return false, fmt.Errorf("crystallized verification for beacon block %v failed", block.SlotNumber())
	}
	return canProcess, nil
}

// verifyBlockTimeStamp verifies node's local time is greater than or equal to
// min timestamp as computed by GENESIS_TIME + slot_number * SLOT_DURATION.
func (b *BeaconChain) verifyBlockTimeStamp(block *types.Block) (bool, error) {
	slotDuration := time.Duration(block.SlotNumber()*params.SlotDuration) * time.Second
	genesis, err := b.GenesisBlock()
	if err != nil {
		return false, err
	}
	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return false, err
	}
	if clock.Now().Before(genesisTime.Add(slotDuration)) {
		return false, nil
	}
	return true, nil
}

// verifyBlockActiveHash verifies block's active state hash equal to
// node's computed active state hash.
func (b *BeaconChain) verifyBlockActiveHash(block *types.Block) (bool, error) {
	hash, err := b.ActiveState().Hash()
	if err != nil {
		return false, err
	}
	if block.ActiveStateHash() != hash {
		return false, nil
	}
	return true, nil
}

// verifyBlockCrystallizedHash verifies block's crystallized state hash equal to
// node's computed crystallized state hash.
func (b *BeaconChain) verifyBlockCrystallizedHash(block *types.Block) (bool, error) {
	hash, err := b.CrystallizedState().Hash()
	if err != nil {
		return false, err
	}
	if block.CrystallizedStateHash() != hash {
		return false, nil
	}
	return true, nil
}

// computeNewActiveState for every newly processed beacon block.
func (b *BeaconChain) computeNewActiveState(seed common.Hash) (*types.ActiveState, error) {
	newActiveState := types.NewActiveState(&pb.ActiveState{
		PendingAttestations: []*pb.AttestationRecord{},
		RecentBlockHashes:   [][]byte{},
	})
	attesters, proposer, err := casper.SampleAttestersAndProposers(seed, b.CrystallizedState())
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

	return newActiveState, nil
}

// computeNewCrystallizedState for every newly processed beacon block at a cycle transition.
func (b *BeaconChain) computeNewCrystallizedState(active *types.ActiveState, block *types.Block) (*types.CrystallizedState, error) {
	newCrystallized := b.CrystallizedState()
	newCrystallized.SetStateRecalc(block.SlotNumber())
	if err := casper.CalculateRewardsFFG(active, newCrystallized, block); err != nil {
		return newCrystallized, fmt.Errorf("could not calculate ffg rewards/penalties: %v", err)
	}
	return newCrystallized, nil
}

// saveBlock puts the passed block into the beacon chain db.
func (b *BeaconChain) saveBlock(block *types.Block) error {
	encodedState, err := block.Marshal()
	if err != nil {
		return err
	}
	hash, err := block.Hash()
	if err != nil {
		return err
	}

	return b.db.Put(hash[:], encodedState)
}

// saveCanonical puts the passed block into the beacon chain db
// and also saves a "latest-head" key mapping to the block in the db.
func (b *BeaconChain) saveCanonical(block *types.Block) error {
	if err := b.saveBlock(block); err != nil {
		return err
	}
	enc, err := block.Marshal()
	if err != nil {
		return err
	}
	return b.db.Put([]byte(canonicalHeadKey), enc)
}
