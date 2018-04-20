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
	notaryLockupLenght *big.Int
	proposerLockupLength *big.Int
}

var (
	mainKey, _               = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	accountBalance1001Eth, _ = new(big.Int).SetString("5000000000000000000000", 10)
	notaryDeposit, _       = new(big.Int).SetString("1000000000000000000000", 10)
	smcConfig                = SMCConfig{
		notaryLockupLenght: new(big.Int).SetInt64(1),
		proposerLockupLength: new(big.Int).SetInt64(1),
	}
)

func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
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

// Test register notary
func TestNotaryDeposit(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Test register_notary() function
	// Deposit 1000 Eth
	transactOpts.Value = notaryDeposit

	t.Log(transactOpts.)
	if _, err := smc.RegisterNotary(transactOpts); err != nil {
		t.Fatalf("Notary cannot deposit: %v", err)
	}
	backend.Commit()

	// Check updated number of notaries
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get notary pool length: %v", err)
	}
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("Failed to update number of notaries")
	}

	// Check deposited is true
	tx, err := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatalf("Failed to update notary registry: %v", err)
	}
	if tx.Deposited != true {
		t.Fatalf("Notary registry not updated")
	}

	// Check for the RegisterNotary event
	depositsEventsIterator, err := smc.FilterNotaryRegistered(&bind.FilterOpts{})
	if err != nil {
		t.Fatalf("Failed to get Deposit event: %v", err)
	}
	if !depositsEventsIterator.Next() {
		t.Fatal("No Deposit event found")
	}
	if depositsEventsIterator.Event.Notary != addr {
		t.Fatalf("Notary address mismatch: %x should be %x", depositsEventsIterator.Event.Notary, addr)
	}
	if depositsEventsIterator.Event.PoolIndex.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Notary index mismatch: %d should be 0", depositsEventsIterator.Event.PoolIndex)
	}
}

// Test notary deregister from pool
func TestNotaryDeregister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance1001Eth}})
	transactOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	transactOpts.Value = notaryDeposit
	// Register notary
	smc.RegisterNotary(transactOpts)

	transactOpts.Value = big.NewInt(0)
	// Deregister notary
	_, err := smc.DeregisterNotary(transactOpts)
	backend.Commit()
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}

	// Check for the NotaryDeregistered event
	withdrawsEventsIterator, err := smc.FilterNotaryDeregistered(&bind.FilterOpts{Start: 0})
	if err != nil {
		t.Fatalf("Failed to get NotaryDeregistered event: %v", err)
	}
	if !withdrawsEventsIterator.Next() {
		t.Fatal("No NotaryDeregistered event found")
	}
	if withdrawsEventsIterator.Event.PoolIndex.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Notary index mismatch: %d should be 0", withdrawsEventsIterator.Event.PoolIndex)
	}
}

func TestGetEligibleNotary(t *testing.T) {
	const numberNotaries = 20
	var notaryPoolAddr [numberNotaries]common.Address
	var notaryPoolPrivKeys [numberNotaries]*ecdsa.PrivateKey
	var transactOpts [numberNotaries]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	for i := 0; i < numberNotaries; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		transactOpts[i] = bind.NewKeyedTransactor(key)
		transactOpts[i].Value = notaryDeposit

		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance1001Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// Register notary
	for i := 0; i < numberNotaries; i++ {
		smc.RegisterNotary(transactOpts[i])
	}

	// Move blockchain 4 blocks further (Head is at 5 after) : period 1
	for i := 0; i < 4; i++ {
		backend.Commit()
	}

	_, err := smc.IsNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(4))
	if err != nil {
		t.Fatalf("Cannot get eligible notary: %v", err)
	}
	// after: Head is 11, period 2
	for i := 0; i < 6; i++ {
		backend.Commit()
	}
	_, err = smc.IsNotaryInCommittee(&bind.CallOpts{}, big.NewInt(1), big.NewInt(6))
	if err != nil {
		t.Fatalf("Cannot get eligible notary: %v\n", err)
	}
}
