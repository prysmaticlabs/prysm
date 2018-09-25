package blockchain

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
)

// BeaconChain represents the core PoS blockchain object containing
// both a crystallized and active state.
type BeaconChain struct {
	state *beaconState
	db    *db.DB
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
func NewBeaconChain(db *db.DB) (*BeaconChain, error) {
	beaconChain := &BeaconChain{
		db:    db,
		state: &beaconState{},
	}

	var aState *types.ActiveState
	var cState *types.CrystallizedState
	var err error

	hasInitialState := db.HasInitialState()
	if !hasInitialState {
		log.Info("No state found on disk, initializing genesis block and state")

		block, err := types.NewGenesisBlock()
		if err != nil {
			return nil, err
		}

		aState = types.NewGenesisActiveState()
		cState, err = types.NewGenesisCrystallizedState()
		if err != nil {
			return nil, err
		}

		db.SaveInitialState(block, aState, cState)
	} else {
		aState, err = db.GetActiveState()
		if err != nil {
			return nil, err
		}

		cState, err = db.GetCrystallizedState()
		if err != nil {
			return nil, err
		}
	}

	beaconChain.state.ActiveState = aState
	beaconChain.state.CrystallizedState = cState

	return beaconChain, nil
}

// GenesisBlock returns the canonical, genesis block.
func (b *BeaconChain) GenesisBlock() (*types.Block, error) {
	return b.db.GetBlockBySlot(0)
}

// CanonicalHead fetches the latest head stored in persistent storage.
func (b *BeaconChain) CanonicalHead() (*types.Block, error) {
	return b.db.GetChainTip()
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
func (b *BeaconChain) SetActiveState(activeState *types.ActiveState) {
	b.state.ActiveState = activeState
}

// SetCrystallizedState is a convenience method which sets and persists the crystallized state on the beacon chain.
func (b *BeaconChain) SetCrystallizedState(crystallizedState *types.CrystallizedState) {
	b.state.CrystallizedState = crystallizedState
}
