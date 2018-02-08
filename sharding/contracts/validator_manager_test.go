package contracts

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	key, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr                     = crypto.PubkeyToAddress(key.PublicKey)
	accountBalance1000Eth, _ = new(big.Int).SetString("1000000000000000000000", 10)
)

func deployVMCContract(backend *backends.SimulatedBackend) (common.Address, *types.Transaction, *VMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeployVMC(transactOpts, backend)
}

// Test creating the VMC contract
func TestContractCreation(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1000Eth}})
	_, _, _, err := deployVMCContract(backend)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy VMC: %v", err)
	}
}

// Test getting the collation gas limit
func TestGetCollationGasLimit(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1000Eth}})
	_, _, vmc, _ := deployVMCContract(backend)
	gasLimit, err := vmc.GetCollationGasLimit(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Error getting collationGasLimit: %v", err)
	}
	if gasLimit.Cmp(big.NewInt(10000000)) != 0 {
		t.Fatalf("collation gas limit should be 10000000 gas")
	}
}

// Test validator deposit
func TestValidatorDeposit(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1000Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, vmc, _ := deployVMCContract(backend)

	// Test deposit() function
	// Deposit 100 Eth
	transactOpts.Value, _ = new(big.Int).SetString("100000000000000000000", 10)

	if _, err := vmc.Deposit(transactOpts); err != nil {
		t.Fatalf("Validator cannot deposit: %v", err)
	}
	backend.Commit()
	//Check for the Deposit event
	depositsEventsIterator, err := vmc.FilterDeposit(&bind.FilterOpts{})
	if err != nil {
		t.Fatalf("Failed to get Deposit event: %v", err)
	}
	if depositsEventsIterator.Next() == false {
		t.Fatal("No Deposit event found")
	}
	if depositsEventsIterator.Event.Validator != addr {
		t.Fatalf("Validator address mismatch: %x should be %x", depositsEventsIterator.Event.Validator, addr)
	}
	if depositsEventsIterator.Event.Index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Validator index mismatch: %d should be 0", depositsEventsIterator.Event.Index)
	}
}

// Test validator withdraw
func TestValidatorWithdraw(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1000Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, vmc, _ := deployVMCContract(backend)

	transactOpts.Value, _ = new(big.Int).SetString("100000000000000000000", 10)
	vmc.Deposit(transactOpts)

	transactOpts.Value = big.NewInt(0)
	_, err := vmc.Withdraw(transactOpts, big.NewInt(0))
	if err != nil {
		t.Fatalf("Failed to withdraw: %v", err)
	}
	backend.Commit()

	//Check for the Withdraw event
	withdrawsEventsIterator, err := vmc.FilterWithdraw(&bind.FilterOpts{Start: 0})
	if err != nil {
		t.Fatalf("Failed to get withdraw event: %v", err)
	}
	if withdrawsEventsIterator.Next() == false {
		t.Fatal("No withdraw event found")
	}
	if withdrawsEventsIterator.Event.ValidatorIndex.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Validator index mismatch: %d should be 0", withdrawsEventsIterator.Event.ValidatorIndex)
	}
}
