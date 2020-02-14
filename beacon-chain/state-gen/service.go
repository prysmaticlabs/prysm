package stategen

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
)

// State represents an object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB  db.NoHeadAccessDatabase
	splitSlot uint64
	slotsPerArchivePoint uint64
}

func New(db db.NoHeadAccessDatabase)*State {
	return &State{
		beaconDB: db,
		slotsPerArchivePoint: uint64(flags.Get().SlotsPerArchivePoint),
	}
}
