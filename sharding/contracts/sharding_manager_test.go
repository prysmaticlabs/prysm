package contracts

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type SMCConfig struct {
	collatorLockupLenght *big.Int
	proposerLockupLength *big.Int
}

var (
	mainKey, _               = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	accountBalance1001Eth, _ = new(big.Int).SetString("1001000000000000000000", 10)
	collatorDeposit, _       = new(big.Int).SetString("1000000000000000000000", 10)
	smcConfig                = SMCConfig{
		collatorLockupLenght: new(big.Int).SetInt64(1),
		proposerLockupLength: new(big.Int).SetInt64(1),
	}
)

func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend, smcConfig.collatorLockupLenght, smcConfig.proposerLockupLength)
}

// Test creating the SMC contract
func TestContractCreation(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	_, _, _, err := deploySMCContract(backend, mainKey)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// Test register collator
func TestCollatorDeposit(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Test register_collator() function
	// Deposit 1000 Eth
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

// Test collator deregister from pool
func TestCollatorDeregister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

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
}

func TestGetEligibleCollator(t *testing.T) {
	const numberCollators = 20
	var collatorPoolAddr [numberCollators]common.Address
	var collatorPoolPrivKeys [numberCollators]*ecdsa.PrivateKey
	var transactOpts [numberCollators]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	for i := 0; i < numberCollators; i++ {
		key, _ := crypto.GenerateKey()
		collatorPoolPrivKeys[i] = key
		collatorPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		transactOpts[i] = bind.NewKeyedTransactor(key)
		transactOpts[i].Value = collatorDeposit

		genesis[collatorPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance1001Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, collatorPoolPrivKeys[0])

	// Register collator
	for i := 0; i < numberCollators; i++ {
		smc.Register_collator(transactOpts[i])
	}

	// Move blockchain 4 blocks further (Head is at 5 after) : period 1
	for i := 0; i < 4; i++ {
		backend.Commit()
	}

	_, err := smc.Get_eligible_collator(&bind.CallOpts{}, big.NewInt(0), big.NewInt(4))
	if err != nil {
		t.Fatalf("Cannot get eligible collator: %v", err)
	}
	// after: Head is 11, period 2
	for i := 0; i < 6; i++ {
		backend.Commit()
	}
	_, err = smc.Get_eligible_collator(&bind.CallOpts{}, big.NewInt(1), big.NewInt(6))
	if err != nil {
		t.Fatalf("Cannot get eligible collator: %v\n", err)
	}
}
