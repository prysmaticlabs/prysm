package testutil

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

// MockBlockFetcher is a fake implementation of blockfetcher.Fetcher.
type MockBlockFetcher struct {
	BlockToReturn interfaces.ReadOnlySignedBeaconBlock
	ErrorToReturn error
}

func (m *MockBlockFetcher) Block(_ context.Context, _ []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return m.BlockToReturn, nil
}
