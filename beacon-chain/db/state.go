package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
)

// GetActiveState contains the current state of attestations and changes every block.
func (db *BeaconDB) GetActiveState() *types.ActiveState {
	return db.state.aState
}

// GetCrystallizedState contains cycle dependent validator information, changes every cycle.
func (db *BeaconDB) GetCrystallizedState() *types.CrystallizedState {
	return db.state.cState
}

// SaveActiveState is a convenience method which sets and persists the active state on the beacon chain.
func (db *BeaconDB) SaveActiveState(activeState *types.ActiveState) error {
	db.state.aState = activeState
	encodedState, err := db.GetCrystallizedState().Marshal()
	if err != nil {
		return err
	}
	return db.put(activeStateLookupKey, encodedState)
}

// SaveCrystallizedState is a convenience method which sets and persists the crystallized state on the beacon chain.
func (db *BeaconDB) SaveCrystallizedState(crystallizedState *types.CrystallizedState) error {
	db.state.cState = crystallizedState
	encodedState, err := db.GetActiveState().Marshal()
	if err != nil {
		return err
	}
	return db.put(crystallizedStateLookupKey, encodedState)
}
