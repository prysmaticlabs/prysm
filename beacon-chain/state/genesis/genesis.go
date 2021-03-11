package genesis

import (
	_ "embed"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	//go:embed prater.ssz
	praterRawSSZ []byte
	//go:embed pyrmont.ssz
	pyrmontRawSSZ []byte
	//go:embed mainnet.ssz
	mainnetRawSSZ []byte
)

// GenesisState returns a copy of the genesis state from a hardcoded value.
func GenesisState(name string) (*state.BeaconState, error) {
	switch name {
	case params.ConfigNames[params.Prater]:
		return load(praterRawSSZ)
	case params.ConfigNames[params.Pyrmont]:
		return load(pyrmontRawSSZ)
	case params.ConfigNames[params.Mainnet]:
		return load(mainnetRawSSZ)
	default:
		// No state found.
		return nil, nil
	}
}

func load(b []byte) (*state.BeaconState, error) {
	st := &pbp2p.BeaconState{}
	if err := st.UnmarshalSSZ(b); err != nil {
		return nil, err
	}
	return state.InitializeFromProtoUnsafe(st)
}