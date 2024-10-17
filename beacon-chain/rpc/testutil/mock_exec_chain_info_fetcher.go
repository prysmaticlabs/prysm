package testutil

import (
	"context"
	"math/big"
)

// MockExecutionChainInfoFetcher is a fake implementation of the powchain.ChainInfoFetcher
type MockExecutionChainInfoFetcher struct {
	CurrEndpoint string
	CurrError    error
	Syncing      bool
	Connected    bool
}

func (*MockExecutionChainInfoFetcher) GenesisExecutionChainInfo() (uint64, *big.Int) {
	return uint64(0), &big.Int{}
}

func (m *MockExecutionChainInfoFetcher) ExecutionClientConnected() bool {
	return m.Connected
}

func (m *MockExecutionChainInfoFetcher) ExecutionClientEndpoint() string {
	return m.CurrEndpoint
}

func (m *MockExecutionChainInfoFetcher) ExecutionClientConnectionErr() error {
	return m.CurrError
}

func (m *MockExecutionChainInfoFetcher) IsExecutionClientSyncing(_ context.Context) (bool, error) {
	return m.Syncing, nil
}
