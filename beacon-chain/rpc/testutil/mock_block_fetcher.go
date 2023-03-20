package testutil

import (
	"context"
	"strconv"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// MockBlockFetcher is a fake implementation of blockfetcher.Fetcher.
type MockBlockFetcher struct {
	BlockToReturn interfaces.ReadOnlySignedBeaconBlock
	ErrorToReturn error
	SlotBlockMap  map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock
}

func (m *MockBlockFetcher) Block(_ context.Context, b []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	slotNumber, parseErr := strconv.ParseUint(string(b), 10, 64)
	if parseErr != nil {
		return m.BlockToReturn, nil
	}
	return m.SlotBlockMap[primitives.Slot(slotNumber)], nil
}
