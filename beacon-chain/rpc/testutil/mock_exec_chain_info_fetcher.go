package testutil

import (
	"math/big"
)

// MockExecutionChainInfoFetcher is a fake implementation of the powchain.ChainInfoFetcher
type MockExecutionChainInfoFetcher struct {
	CurrEndpoint string
	CurrError    error
	Endpoints    []string
	Errors       []error
}

func (*MockExecutionChainInfoFetcher) GenesisExecutionChainInfo() (uint64, *big.Int) {
	return uint64(0), &big.Int{}
}

func (*MockExecutionChainInfoFetcher) IsConnectedToETH1() bool {
	return true
}

func (m *MockExecutionChainInfoFetcher) CurrentETH1Endpoint() string {
	return m.CurrEndpoint
}

func (m *MockExecutionChainInfoFetcher) CurrentETH1ConnectionError() error {
	return m.CurrError
}

func (m *MockExecutionChainInfoFetcher) ETH1Endpoints() []string {
	return m.Endpoints
}

func (m *MockExecutionChainInfoFetcher) ETH1ConnectionErrors() []error {
	return m.Errors
}
