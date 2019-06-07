package backend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func TestSimulatedBackendStop_ShutsDown(t *testing.T) {

	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}
	if err := backend.Shutdown(); err != nil {
		t.Errorf("Could not successfully shutdown simulated backend %v", err)
	}

	db.TeardownDB(backend.beaconDB)
}

func TestGenerateBlockAndAdvanceChain_IncreasesSlot(t *testing.T) {
	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backend %v", err)
	}

	privKeys, err := backend.SetupBackend(100)
	if err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}
	defer backend.Shutdown()
	defer db.TeardownDB(backend.beaconDB)

	slotLimit := params.BeaconConfig().SlotsPerEpoch + uint64(1)

	for i := uint64(0); i < slotLimit; i++ {
		if err := backend.GenerateBlockAndAdvanceChain(&SimulatedObjects{}, privKeys); err != nil {
			t.Fatalf("Could not generate block and transition state successfully %v for slot %d", err, backend.state.Slot+1)
		}
		if backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1].Slot != backend.state.Slot {
			t.Errorf("In memory Blocks do not have the same last slot as the state, expected %d but got %v",
				backend.state.Slot, backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1])
		}
	}

	if backend.state.Slot != params.BeaconConfig().GenesisSlot+uint64(slotLimit) {
		t.Errorf("Unequal state slot and expected slot %d %d", backend.state.Slot, slotLimit)
	}

}

func TestGenerateNilBlockAndAdvanceChain_IncreasesSlot(t *testing.T) {
	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}

	if _, err := backend.SetupBackend(100); err != nil {
		t.Fatalf("Could not set up backend %v", err)
	}
	defer backend.Shutdown()
	defer db.TeardownDB(backend.beaconDB)

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
