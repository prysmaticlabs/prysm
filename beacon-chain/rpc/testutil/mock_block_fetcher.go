package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

// MockBlockFetcher is a fake implementation of blockfetcher.Fetcher.
type MockBlockFetcher struct {
	BlockToReturn interfaces.ReadOnlySignedBeaconBlock
}

func (m *MockBlockFetcher) Block(_ context.Context, _ []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	return m.BlockToReturn, nil
}
