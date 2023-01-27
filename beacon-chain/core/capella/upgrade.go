package capella

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
)

// UpgradeToCapella updates a generic state to return the version Capella state.
func UpgradeToCapella(state state.BeaconState) (state.BeaconState, error) {
	return state.UpgradeToCapella()
}
