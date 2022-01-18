package testutil

import (
	"math/big"
)

// MockGenesisTimeFetcher is a fake implementation of the powchain.ChainInfoFetcher
type MockPOWChainInfoFetcher struct {
	CurrEndpoint string
	CurrError    error
	Endpoints    []string
	Errors       []error
}

func (m *MockPOWChainInfoFetcher) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return uint64(0), &big.Int{}
}

func (m *MockPOWChainInfoFetcher) IsConnectedToETH1() bool {
	return true
}

func (m *MockPOWChainInfoFetcher) CurrentETH1Endpoint() string {
	return m.CurrEndpoint
}

func (m *MockPOWChainInfoFetcher) CurrentETH1ConnectionError() error {
	return m.CurrError
}

func (m *MockPOWChainInfoFetcher) ETH1Endpoints() []string {
	return m.Endpoints
}

func (m *MockPOWChainInfoFetcher) ETH1ConnectionErrors() []error {
	return m.Errors
}
