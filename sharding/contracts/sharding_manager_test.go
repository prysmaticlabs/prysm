package contracts

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
)

type SMCConfig struct {
	notaryLockupLenght   *big.Int
	proposerLockupLength *big.Int
}

var (
	mainKey, _                   = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	accountBalance2000Eth, _     = new(big.Int).SetString("2000000000000000000000", 10)
	notaryDepositInsufficient, _ = new(big.Int).SetString("999000000000000000000", 10)
	notaryDeposit, _             = new(big.Int).SetString("1000000000000000000000", 10)
	FastForward100Blocks         = 100
	smcConfig                    = SMCConfig{
		notaryLockupLenght:   new(big.Int).SetInt64(1),
		proposerLockupLength: new(big.Int).SetInt64(1),
	}
	ctx = context.Background()
)

func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// Test creating the SMC contract
func TestContractCreation(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	_, _, _, err := deploySMCContract(backend, mainKey)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

func TestNotaryRegister(t *testing.T) {
	const notaryCount = 3
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// initializes back end with 3 accounts and each with 2000 eth balances
	for i := 0; i < notaryCount; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		txOpts[i] = bind.NewKeyedTransactor(key)
		txOpts[i].Value = notaryDeposit
		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance2000Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// Notary 0 has not registered
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[0])
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}

	if notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	// Test notary 0 has registered
	if _, err := smc.RegisterNotary(txOpts[0]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[0])

	if !notary.Deposited ||
		notary.PoolIndex.Cmp(big.NewInt(0)) != 0 ||
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary registry. Want - deposited:true, index:0, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	// Test notary 1 and 2 have registered
	if _, err := smc.RegisterNotary(txOpts[1]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	if _, err := smc.RegisterNotary(txOpts[2]); err != nil {
		t.Fatalf("Registering notary has failed: %v", err)
	}
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[1])

	if !notary.Deposited ||
		notary.PoolIndex.Cmp(big.NewInt(1)) != 0 ||
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary registry. Want - deposited:true, index:1, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[2])

	if !notary.Deposited ||
		notary.PoolIndex.Cmp(big.NewInt(2)) != 0 ||
		notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary registry. Want - deposited:true, index:2, period:0"+
			"Got - deposited:%v, index:%v, period:%v ", notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
	}

	// Check total numbers of notaries in pool
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get notary pool length: %v", err)
	}
	if numNotaries.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 3, Got: %v", numNotaries)
	}
}

func TestNotaryRegisterInsufficientEther(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDepositInsufficient
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	_, err := smc.RegisterNotary(txOpts)
	if err == nil {
		t.Errorf("Notary register should have failed with insufficient deposit")
	}

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if notary.Deposited {
		t.Errorf("Notary deposited with insufficient fund")
	}

	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

}

func TestNotaryDoubleRegisters(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if !notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Notary 0 registers again
	_, err := smc.RegisterNotary(txOpts)
	if err == nil {
		t.Errorf("Notary register should have failed with double registers")
	}
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

}

func TestNotaryDeregister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if !notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward 100 blocks to check notary's deregistered period field is set correctly
	for i := 0; i < FastForward100Blocks; i++ {
		backend.Commit()
	}

	// Notary 0 deregisters
	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	// Verify notary has saved the deregistered period as: current block number / period length
	notary, _ = smc.NotaryRegistry(&bind.CallOpts{}, addr)
	currentPeriod := big.NewInt(int64(FastForward100Blocks) / sharding.PeriodLength)
	if currentPeriod.Cmp(notary.DeregisteredPeriod) != 0 {
		t.Errorf("Incorrect notary degister period. Want: %v, Got: %v ", currentPeriod, notary.DeregisteredPeriod)
	}

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

func TestNotaryDeregisterThenRegister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if !notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Notary 0 deregisters
	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Notary 0 re-registers again
	smc.RegisterNotary(txOpts)
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

func TestNotaryRelease(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if !notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward to the next period to deregister
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters
	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Fast forward until lockup ends
	for i := 0; i < int(sharding.NotaryLockupLength*sharding.PeriodLength+1); i++ {
		backend.Commit()
	}

	// Notary 0 releases
	_, err = smc.ReleaseNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to release notary: %v", err)
	}
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}

	if notary.Deposited {
		t.Errorf("Notary deposit flag should be false after released")
	}

	balance, err := backend.BalanceAt(ctx, addr, nil)
	if err != nil {
		t.Errorf("Can't get account balance, err: %s", err)
	}

	if balance.Cmp(notaryDeposit) < 0 {
		t.Errorf("Notary did not receive deposit after lock up ends")
	}
}

func TestNotaryInstantRelease(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})

	if !notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward to the next period to deregister
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters
	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Notary 0 tries to release before lockup ends
	_, err = smc.ReleaseNotary(txOpts)
	backend.Commit()

	notary, err = smc.NotaryRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}

	if !notary.Deposited {
		t.Errorf("Notary deposit flag should be true before released")
	}

	balance, err := backend.BalanceAt(ctx, addr, nil)

	if balance.Cmp(notaryDeposit) > 0 {
		t.Errorf("Notary received deposit before lockup ends")
	}
}

func TestCommitteeListsAreDifferent(t *testing.T) {
	const notaryCount = 10000
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// initializes back end with 10000 accounts and each with 2000 eth balances
	for i := 0; i < notaryCount; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		txOpts[i] = bind.NewKeyedTransactor(key)
		txOpts[i].Value = notaryDeposit
		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance2000Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// register 10000 notaries to SMC
	for i := 0; i < notaryCount; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1000, Got: %v", numNotaries)
	}

	// get a list of sampled notaries from shard 0
	var shard0CommitteeList []string
	for i := 0; i < int(sharding.NotaryCommitSize); i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		shard0CommitteeList = append(shard0CommitteeList, addr.String())
	}

	// get a list of sampled notaries from shard 1, verify it's not identical to shard 0
	for i := 0; i < int(sharding.NotaryCommitSize); i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(1), big.NewInt(int64(i)))
		if shard0CommitteeList[i] == addr.String() {
			t.Errorf("Shard 0 committee list is identical to shard 1's committee list")
		}
	}
}

func TestGetCommitteeWithNonMember(t *testing.T) {
	const notaryCount = 11
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// initialize back end with 11 accounts and each with 2000 eth balances
	for i := 0; i < notaryCount; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		txOpts[i] = bind.NewKeyedTransactor(key)
		txOpts[i].Value = notaryDeposit
		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance2000Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// register 10 notaries to SMC, leave 1 address free
	for i := 0; i < 10; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 135, Got: %v", numNotaries)
	}

	// verify the unregistered account is not in the notary pool list
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		if notaryPoolAddr[10].String() == addr.String() {
			t.Errorf("Account %s is not a notary", notaryPoolAddr[10].String())
		}
	}

}

func TestGetCommitteeAfterDeregisters(t *testing.T) {
	const notaryCount = 10
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// initialize back end with 10 accounts and each with 2000 eth balances
	for i := 0; i < notaryCount; i++ {
		key, _ := crypto.GenerateKey()
		notaryPoolPrivKeys[i] = key
		notaryPoolAddr[i] = crypto.PubkeyToAddress(key.PublicKey)
		txOpts[i] = bind.NewKeyedTransactor(key)
		txOpts[i].Value = notaryDeposit
		genesis[notaryPoolAddr[i]] = core.GenesisAccount{
			Balance: accountBalance2000Eth,
		}
	}

	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPoolPrivKeys[0])

	// register 10 notaries to SMC
	for i := 0; i < 10; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 10, Got: %v", numNotaries)
	}

	// deregister notary 0 from SMC
	txOpts[0].Value = big.NewInt(0)
	smc.DeregisterNotary(txOpts[0])
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 9, Got: %v", numNotaries)
	}

	// verify degistered notary 0 is not in the notary pool list
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		if notaryPoolAddr[0].String() == addr.String() {
			t.Errorf("Account %s is not a notary", notaryPoolAddr[0].String())
		}
	}
}
