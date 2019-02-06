package backend

import "testing"

func TestSimulatedBackendStartAndStop(t *testing.T) {

	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}

	if err := backend.InitializeChain(); err != nil {
		t.Fatalf("Could not initialize simulated backend %v", err)
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

	if err := backend.InitializeChain(); err != nil {
		t.Fatalf("Could not initialize simulated backend %v", err)
	}

	slotLimit := 250

	for i := 0; i < slotLimit; i++ {
		if err := backend.GenerateBlockAndAdvanceChain(&SimulatedObjects{}); err != nil {
			t.Fatalf("Could not generate block and transition state successfully %v for slot %d", err, backend.state.Slot+1)
		}
		if backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1].Slot != backend.state.Slot {
			t.Errorf("In memory Blocks do not have the same last slot as the state, expected %d but got %d",
				backend.state.Slot, backend.inMemoryBlocks[len(backend.inMemoryBlocks)-1])
		}
	}

	if backend.state.Slot != uint64(slotLimit) {
		t.Errorf("Unequal state slot and expected slot %d %d", backend.state.Slot, slotLimit)
	}

}

func TestGenerateNilBlocks(t *testing.T) {
	backend, err := NewSimulatedBackend()
	if err != nil {
		t.Fatalf("Could not create a new simulated backedn %v", err)
	}

	if err := backend.InitializeChain(); err != nil {
		t.Fatalf("Could not initialize simulated backend %v", err)
	}

	slotLimit := 100

	for i := 0; i < slotLimit; i++ {
		if err := backend.GenerateNilBlockAndAdvanceChain(); err != nil {
			t.Fatalf("Could not generate block and transition state successfully %v for slot %d", err, backend.state.Slot+1)
		}
	}

	if backend.state.Slot != uint64(slotLimit) {
		t.Errorf("Unequal state slot and expected slot %d %d", backend.state.Slot, slotLimit)
	}

}
