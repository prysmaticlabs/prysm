package depositcontract

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	amount32Eth        = "32000000000000000000"
	amountLessThan1Eth = "500000000000000000"
)

// TestAccount represents a test account in the simulated backend,
// through which we can perform actions on the eth1.0 chain.
type TestAccount struct {
	Addr         common.Address
	ContractAddr common.Address
	Contract     *DepositContract
	Backend      *backends.SimulatedBackend
	TxOpts       *bind.TransactOpts
}

// Setup creates the simulated backend with the deposit contract deployed
func Setup() (*TestAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// strip off the 0x and the first 2 characters 04 which is always the EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey[:], publicKeyBytes)

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("100000000000000000000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 210000000000)

	contractAddr, _, contract, err := DeployDepositContract(txOpts, backend, addr)
	if err != nil {
		return nil, err
	}
	backend.Commit()

	return &TestAccount{addr, contractAddr, contract, backend, txOpts}, nil
}

// Amount32Eth returns 32Eth(in wei) in terms of the big.Int type.
func Amount32Eth() *big.Int {
	amount, _ := new(big.Int).SetString(amount32Eth, 10)
	return amount
}

// LessThan1Eth returns less than 1 Eth(in wei) in terms of the big.Int type.
func LessThan1Eth() *big.Int {
	amount, _ := new(big.Int).SetString(amountLessThan1Eth, 10)
	return amount
}
