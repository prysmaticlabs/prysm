package p2p

import (
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type mockChain struct {
	currentFork     *ethpb.Fork
	genesisValsRoot [32]byte
	genesisTime     time.Time
	currentSlot     types.Slot
}

func (m *mockChain) ForkChoicer() forkchoice.ForkChoicer {
	return nil
}

func (m *mockChain) CurrentFork() *ethpb.Fork {
	return m.currentFork
}

func (m *mockChain) GenesisValidatorsRoot() [32]byte {
	return m.genesisValsRoot
}

func (m *mockChain) GenesisTime() time.Time {
	return m.genesisTime
}

func (m *mockChain) CurrentSlot() types.Slot {
	return m.currentSlot
}
