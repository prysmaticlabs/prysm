package backend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestSimulatedBackendStop(t *testing.T) {

	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}
	if err := backend.Shutdown(); err != nil {
		t.Errorf("Could not successfully shutdown simulated backend %v", err)
	}
}

func TestGenerateBlocks(t *testing.T) {
	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}

	initialDeposits, privKeys, err := generateInitialSimulatedDeposits(params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		t.Fatalf("Could not simulate initial validator deposits: %v", err)
	}
	if err := backend.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		t.Fatalf("Could not set up beacon state and initialize genesis block %v", err)
	}
	backend.depositTrie = trieutil.NewDepositTrie()

	slotLimit := params.BeaconConfig().SlotsPerEpoch + uint64(1)

	for i := uint64(0); i < slotLimit; i++ {
		if err := backend.GenerateBlockAndAdvanceChain(&SimulatedObjects{}, privKeys); err != nil {
			t.Fatalf("Could not generate block and transition state successfully %v for slot %d", err, backend.state.Slot+1)
		}
		if backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1].Slot != backend.state.Slot {
			t.Errorf("In memory Blocks do not have the same last slot as the state, expected %d but got %d",
				backend.state.Slot, backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1])
		}
	}

	if backend.state.Slot != params.BeaconConfig().GenesisSlot+uint64(slotLimit) {
		t.Errorf("Unequal state slot and expected slot %d %d", backend.state.Slot, slotLimit)
	}

}

func TestGenerateNilBlocks(t *testing.T) {
	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}

	initialDeposits, _, err := generateInitialSimulatedDeposits(params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		t.Fatalf("Could not simulate initial validator deposits: %v", err)
	}
	if err := backend.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		t.Fatalf("Could not set up beacon state and initialize genesis block %v", err)
	}
	backend.depositTrie = trieutil.NewDepositTrie()

	slotLimit := params.BeaconConfig().SlotsPerEpoch + uint64(1)

	for i := uint64(0); i < slotLimit; i++ {
		if err := backend.GenerateNilBlockAndAdvanceChain(); err != nil {
			t.Fatalf("Could not generate block and transition state successfully %v for slot %d", err, backend.state.Slot+1)
		}
	}

	if backend.state.Slot != params.BeaconConfig().GenesisSlot+uint64(slotLimit) {
		t.Errorf("Unequal state slot and expected slot %d %d", backend.state.Slot, slotLimit)
	}

}
