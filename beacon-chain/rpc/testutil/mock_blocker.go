package testutil

import (
	"context"
	"strconv"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// MockBlocker is a fake implementation of lookup.Blocker.
type MockBlocker struct {
	BlockToReturn interfaces.ReadOnlySignedBeaconBlock
	ErrorToReturn error
	SlotBlockMap  map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock
}

// Block --
func (m *MockBlocker) Block(_ context.Context, b []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	slotNumber, parseErr := strconv.ParseUint(string(b), 10, 64)
	if parseErr != nil {
		//nolint:nilerr
		return m.BlockToReturn, nil
	}
	return m.SlotBlockMap[primitives.Slot(slotNumber)], nil
}
