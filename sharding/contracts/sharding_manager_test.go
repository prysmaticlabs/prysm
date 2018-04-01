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
	accountBalance1001Eth, _ = new(big.Int).SetString("1000000000000000000001", 10)
	collatorDeposit, _       = new(big.Int).SetString("1000000000000000000000", 10)
)

func deploySMCContract(backend *backends.SimulatedBackend) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// Test creating the SMC contract
func TestContractCreation(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	_, _, _, err := deploySMCContract(backend)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// Test getting the collation gas limit
func TestGetCollationGasLimit(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	_, _, smc, _ := deploySMCContract(backend)
	gasLimit, err := smc.GetCollationGasLimit(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Error getting collationGasLimit: %v", err)
	}
	if gasLimit.Cmp(big.NewInt(10000000)) != 0 {
		t.Fatalf("collation gas limit should be 10000000 gas")
	}
}

// Test collator deposit
func TestCollatorDeposit(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, smc, _ := deploySMCContract(backend)

	// Test deposit() function
	// Deposit 100 Eth
	transactOpts.Value = collatorDeposit

	if _, err := smc.Deposit(transactOpts); err != nil {
		t.Fatalf("Collator cannot deposit: %v", err)
	}
	backend.Commit()

	// Check updated number of collators
	numCollators, err := smc.NumCollators(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get number of collators: %v", err)
	}
	if numCollators.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Failed to update number of collators")
	}

	// Check collator structure
	collatorStruct, err := smc.Collators(&bind.CallOpts{}, big.NewInt(0))
	if err != nil {
		t.Fatalf("Failed to get collator structure: %v", err)
	}
	if collatorStruct.Addr != addr {
		t.Fatalf("Wrong collator address, %v should be %v", collatorStruct.Addr, addr)
	}
	if collatorStruct.Deposit.Cmp(collatorDeposit) != 0 {
		t.Fatalf("Wrong collator deposit, %v should be %v", collatorStruct.Deposit, collatorDeposit)
	}

	// Check for the Deposit event
	depositsEventsIterator, err := smc.FilterDeposit(&bind.FilterOpts{})
	if err != nil {
		t.Fatalf("Failed to get Deposit event: %v", err)
	}
	if !depositsEventsIterator.Next() {
		t.Fatal("No Deposit event found")
	}
	if depositsEventsIterator.Event.Collator != addr {
		t.Fatalf("Collator address mismatch: %x should be %x", depositsEventsIterator.Event.Collator, addr)
	}
	if depositsEventsIterator.Event.Index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Collator index mismatch: %d should be 0", depositsEventsIterator.Event.Index)
	}
}

// Test collator withdraw
func TestCollatorWithdraw(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, smc, _ := deploySMCContract(backend)

	transactOpts.Value = collatorDeposit
	smc.Deposit(transactOpts)

	transactOpts.Value = big.NewInt(0)
	_, err := smc.Withdraw(transactOpts, big.NewInt(0))
	if err != nil {
		t.Fatalf("Failed to withdraw: %v", err)
	}
	backend.Commit()

	// Check for the Withdraw event
	withdrawsEventsIterator, err := smc.FilterWithdraw(&bind.FilterOpts{Start: 0})
	if err != nil {
		t.Fatalf("Failed to get withdraw event: %v", err)
	}
	if !withdrawsEventsIterator.Next() {
		t.Fatal("No withdraw event found")
	}
	if withdrawsEventsIterator.Event.Index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Collator index mismatch: %d should be 0", withdrawsEventsIterator.Event.Index)
	}
}
