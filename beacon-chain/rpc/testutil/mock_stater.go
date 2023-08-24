package testutil

import (
	"context"
	"strconv"
	"strings"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// MockStater is a fake implementation of lookup.Stater.
type MockStater struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
	StatesBySlot    map[primitives.Slot]state.BeaconState
	StatesByRoot    map[[32]byte]state.BeaconState
}

// State --
func (m *MockStater) State(_ context.Context, id []byte) (state.BeaconState, error) {
	stateIdString := strings.ToLower(string(id))
	switch stateIdString {
	case "head", "genesis", "finalized", "justified":
		return m.BeaconState, nil
	default:
		if len(id) == 32 {
			return m.StatesByRoot[bytesutil.ToBytes32(id)], nil
		} else {
			_, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				e := lookup.NewStateIdParseError(parseErr)
				return nil, &e
			}
			return m.BeaconState, nil
		}
	}
}

// StateRoot --
func (m *MockStater) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

// StateBySlot --
func (m *MockStater) StateBySlot(_ context.Context, s primitives.Slot) (state.BeaconState, error) {
	return m.StatesBySlot[s], nil
}
