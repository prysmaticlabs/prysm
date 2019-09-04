package mock

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
)

type MockPOWChain struct {
	beaconDB db.Database
}

func (m *MockPOWChain) HasChainStarted() bool {
	return true
}

func (m *MockPOWChain) ChainStartDeposits() []*ethpb.Deposit {
	return 0
}

func (m *MockPOWChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

func (m *MockPOWChain) ChainStartFeed() *event.Feed {
	return new(event.Feed)
}
