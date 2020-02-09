package state_gen

import "github.com/prysmaticlabs/prysm/beacon-chain/db"

// Service represents a service that handles the internal
// logic of maintaining both hot and cold states in DB.
type Service struct {
	beaconDB               db.NoHeadAccessDatabase
	splitSlot uint64
}

