package stategen

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB db.NoHeadAccessDatabase
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB: db,
	}
}
