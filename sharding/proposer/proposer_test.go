package proposer

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding/contracts"
	"math/big"
	"testing"
)

var (
	key, _         = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr           = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance = big.NewInt(1001000000000000000000)
)

type mockClient struct {
	smc *contracts.SMC
	t   *testing.T
}

func (m *mockClient) SMCCaller() *contracts.SMCCaller {
	return &m.smc.SMCCaller
}

func (m *mockClient) SMCTransactor() *contracts.SMCTransactor {
	return &m.smc.SMCTransactor
}

func transactOpts() *bind.TransactOpts {
	return bind.NewKeyedTransactor(key)
}

func setup() (*backends.SimulatedBackend, *contracts.SMC) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance}})
	_, _, smc, _ := contracts.DeploySMC(transactOpts(), backend)
	backend.Commit()
	return backend, smc
}
