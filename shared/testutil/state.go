package testutil

import (
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var st, _ = stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{
	BlockRoots:  make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
	StateRoots:  make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
	Slashings:   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
	RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
})

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState() *stateTrie.BeaconState {
	return st.Copy()
}
