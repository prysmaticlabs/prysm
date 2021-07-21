package testutil

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// MockFetcher is a fake implementation of utils.BlockProducer.
type MockBlockProducer struct {
	Block *ethpb.BeaconBlock
}

// ProduceBlock --
func (m *MockBlockProducer) ProduceBlock(context.Context, types.Slot, []byte, []byte) (*ethpb.BeaconBlock, error) {
	return m.Block, nil
}
