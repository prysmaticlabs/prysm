package genesis

import (
	_ "embed"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

var (
	//go:embed mainnet.ssz.snappy
	mainnetRawSSZCompressed []byte // 1.8Mb
)

// State returns a copy of the genesis state from a hardcoded value.
func State(name string) (types.BeaconState, error) {
	switch name {
	case params.MainnetName:
		return load(mainnetRawSSZCompressed)
	default:
		// No state found.
		return nil, nil
	}
}

// load a compressed ssz state file into a beacon state struct.
func load(b []byte) (types.BeaconState, error) {
	st := &ethpb.BeaconState{}
	b, err := snappy.Decode(nil /*dst*/, b)
	if err != nil {
		return nil, err
	}
	if err := st.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return state.InitializeFromProtoUnsafePhase0(st)
}
