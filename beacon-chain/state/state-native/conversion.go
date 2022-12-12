package state_native

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
)

func (b *BeaconState) ToGenesis() (state.GenesisBeaconState, error) {
	if b.slot > 0 {
		return nil, errors.New("state with non-zero slot cannot be a genesis state")
	}
	return &genesisBeaconState{b}, nil
}

func (b *genesisBeaconState) ToGenesis() (state.GenesisBeaconState, error) {
	return b, nil
}

func (*genesisBeaconState) Stub() {}
