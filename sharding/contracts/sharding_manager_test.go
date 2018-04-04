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
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
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

// Test register collator
func TestCollatorDeposit(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, smc, _ := deploySMCContract(backend)

	// Test register_collator() function
	// Deposit 100 Eth
	transactOpts.Value = collatorDeposit

	if _, err := smc.Register_collator(transactOpts); err != nil {
		t.Fatalf("Collator cannot deposit: %v", err)
	}
	backend.Commit()

	// Check updated number of collators
	numCollators, err := smc.Collator_pool_len(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get collator pool length: %v", err)
	}
	if numCollators.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Failed to update number of collators")
	}

	// Check deposited is true
	tx, err := smc.Collator_registry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatalf("Failed to update collator registry: %v", err)
	}
	if tx.Deposited != true {
		t.Fatalf("Collator registry not updated")
	}

	// Check for the RegisterCollator event
	depositsEventsIterator, err := smc.FilterCollatorRegistered(&bind.FilterOpts{})
	if err != nil {
		t.Fatalf("Failed to get Deposit event: %v", err)
	}
	if !depositsEventsIterator.Next() {
		t.Fatal("No Deposit event found")
	}
	if depositsEventsIterator.Event.Collator != addr {
		t.Fatalf("Collator address mismatch: %x should be %x", depositsEventsIterator.Event.Collator, addr)
	}
	if depositsEventsIterator.Event.Pool_index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Collator index mismatch: %d should be 0", depositsEventsIterator.Event.Pool_index)
	}
}

// Test collator withdraw from the pool
func TestCollatorWithdraw(t *testing.T) {
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(key)
	_, _, smc, _ := deploySMCContract(backend)

	transactOpts.Value = collatorDeposit
	// Register collator
	smc.Register_collator(transactOpts)

	transactOpts.Value = big.NewInt(0)
	// Deregister collator
	_, err := smc.Deregister_collator(transactOpts)
	backend.Commit()
	if err != nil {
		t.Fatalf("Failed to deregister collator: %v", err)
	}

	// Check for the CollatorDeregistered event
	withdrawsEventsIterator, err := smc.FilterCollatorDeregistered(&bind.FilterOpts{Start: 0})
	if err != nil {
		t.Fatalf("Failed to get CollatorDeregistered event: %v", err)
	}
	if !withdrawsEventsIterator.Next() {
		t.Fatal("No CollatorDeregistered event found")
	}
	if withdrawsEventsIterator.Event.Pool_index.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Collator index mismatch: %d should be 0", withdrawsEventsIterator.Event.Pool_index)
	}
	// for i := 0; i < 16128*5+1; i++ {
	// 	backend.Commit()
	// }

	// Release collator
	// _, err = smc.Release_collator(transactOpts)
	// if err != nil {
	// 	t.Fatalf("Failed to release collator: %v", err)
	// }
}
