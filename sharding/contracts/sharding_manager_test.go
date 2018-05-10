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

// deploySMCContract is a helper function for deploying SMC.
func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// TestContractCreation tests SMC smart contract can successfully be deployed.
func TestContractCreation(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	_, _, _, err := deploySMCContract(backend, mainKey)
	backend.Commit()
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// TestNotaryRegister tests notary registers in a normal condition.
func TestNotaryRegister(t *testing.T) {
	const notaryCount = 3
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// Initializes back end with 3 accounts and each with 2000 eth balances.
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

	// Notary 0 has not registered.
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPoolAddr[0])
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}

	if notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	// Test notary 0 has registered.
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

	// Test notary 1 and 2 have registered.
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

	// Check total numbers of notaries in pool.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Fatalf("Failed to get notary pool length: %v", err)
	}
	if numNotaries.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 3, Got: %v", numNotaries)
	}
}

// TestNotaryRegisterInsufficientEther tests notary registers with insufficient deposit.
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

// TestNotaryDoubleRegisters tests notary registers twice.
func TestNotaryDoubleRegisters(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
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

	// Notary 0 registers again.
	_, err := smc.RegisterNotary(txOpts)
	if err == nil {
		t.Errorf("Notary register should have failed with double registers")
	}
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

}

// TestNotaryDeregister tests notary deregisters in a normal condition.
func TestNotaryDeregister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
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

	// Fast forward 100 blocks to check notary's deregistered period field is set correctly.
	for i := 0; i < FastForward100Blocks; i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
	txOpts = bind.NewKeyedTransactor(mainKey)
	_, err := smc.DeregisterNotary(txOpts)
	if err != nil {
		t.Fatalf("Failed to deregister notary: %v", err)
	}
	backend.Commit()

	// Verify notary has saved the deregistered period as: current block number / period length.
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

// TestNotaryDeregisterThenRegister tests notary deregisters then registers before lock up ends.
func TestNotaryDeregisterThenRegister(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
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

	// Notary 0 deregisters.
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

	// Notary 0 re-registers again.
	smc.RegisterNotary(txOpts)
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

// TestNotaryRelease tests notary releases in a normal condition.
func TestNotaryRelease(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
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

	// Fast forward to the next period to deregister.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
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

	// Fast forward until lockup ends.
	for i := 0; i < int(sharding.NotaryLockupLength*sharding.PeriodLength+1); i++ {
		backend.Commit()
	}

	// Notary 0 releases.
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

// TestNotaryInstantRelease tests notary releases before lockup ends.
func TestNotaryInstantRelease(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
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

	// Fast forward to the next period to deregister.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
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

	// Notary 0 tries to release before lockup ends.
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

// TestCommitteeListsAreDifferent tests different shards have different notary committee.
func TestCommitteeListsAreDifferent(t *testing.T) {
	const notaryCount = 1000
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// initializes back end with 10000 accounts and each with 2000 eth balances.
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

	// Register 1000 notaries to SMC.
	for i := 0; i < notaryCount; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(notaryCount)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1000, Got: %v", numNotaries)
	}

	// Compare sampled first 5 notaries of shard 0 to shard 1, they should not be identical.
	for i := 0; i < 5; i++ {
		addr0, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		addr1, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(1), big.NewInt(int64(i)))
		if addr0 == addr1 {
			t.Errorf("Shard 0 committee list is identical to shard 1's committee list")
		}
	}

}

// TestGetCommitteeWithNonMember tests unregistered notary tries to be in the committee.
func TestGetCommitteeWithNonMember(t *testing.T) {
	const notaryCount = 11
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// Initialize back end with 11 accounts and each with 2000 eth balances.
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

	// Register 10 notaries to SMC, leave 1 address free.
	for i := 0; i < 10; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 135, Got: %v", numNotaries)
	}

	// Verify the unregistered account is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		if notaryPoolAddr[10].String() == addr.String() {
			t.Errorf("Account %s is not a notary", notaryPoolAddr[10].String())
		}
	}

}

// TestGetCommitteeAfterDeregisters tests notary tries to be in committee after deregistered.
func TestGetCommitteeAfterDeregisters(t *testing.T) {
	const notaryCount = 10
	var notaryPoolAddr [notaryCount]common.Address
	var notaryPoolPrivKeys [notaryCount]*ecdsa.PrivateKey
	var txOpts [notaryCount]*bind.TransactOpts
	genesis := make(core.GenesisAlloc)

	// Initialize back end with 10 accounts and each with 2000 eth balances.
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

	// Register 10 notaries to SMC.
	for i := 0; i < 10; i++ {
		smc.RegisterNotary(txOpts[i])
		backend.Commit()
	}

	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 10, Got: %v", numNotaries)
	}

	// Deregister notary 0 from SMC.
	txOpts[0].Value = big.NewInt(0)
	smc.DeregisterNotary(txOpts[0])
	backend.Commit()

	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 9, Got: %v", numNotaries)
	}

	// Verify degistered notary 0 is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0), big.NewInt(int64(i)))
		if notaryPoolAddr[0].String() == addr.String() {
			t.Errorf("Account %s is not a notary", notaryPoolAddr[0].String())
		}
	}
}

// TestNormalAddHeader tests proposer add header in normal condition.
func TestNormalAddHeader(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	p, err := smc.LastSubmittedCollation(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Can't get last submitted collation's period number: %v", err)
	}
	if p.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}

	cr, err := smc.CollationRecords(&bind.CallOpts{}, shard0, period1)
	if cr.ChunkRoot != chunkRoot {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Fast forward to the next period. Period 2.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 2 and chunkroot 0xB.
	period2 := big.NewInt(2)
	chunkRoot = [32]byte{'B'}
	_, err = smc.AddHeader(txOpts, shard0, period2, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	p, err = smc.LastSubmittedCollation(&bind.CallOpts{}, shard0)
	if p.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}

	cr, err = smc.CollationRecords(&bind.CallOpts{}, shard0, period2)
	if cr.ChunkRoot != chunkRoot {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Proposer adds header consists shard 1, period 2 and chunkroot 0xC.
	shard1 := big.NewInt(1)
	chunkRoot = [32]byte{'C'}
	_, err = smc.AddHeader(txOpts, shard1, period2, chunkRoot)
	backend.Commit()

	p, err = smc.LastSubmittedCollation(&bind.CallOpts{}, shard1)
	if p.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}

	cr, err = smc.CollationRecords(&bind.CallOpts{}, shard1, period2)
	if cr.ChunkRoot != chunkRoot {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", cr, chunkRoot)
	}
}

// TestAddTwoHeadersAtSamePeriod tests we can't add two headers within the same period.
func TestAddTwoHeadersAtSamePeriod(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	p, err := smc.LastSubmittedCollation(&bind.CallOpts{}, shard0)
	if p.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}
	cr, err := smc.CollationRecords(&bind.CallOpts{}, shard0, period1)
	if cr.ChunkRoot != chunkRoot {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Proposer attempts to add another header chunkroot 0xB on the same period for the same shard.
	chunkRoot = [32]byte{'B'}
	_, err = smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err == nil {
		t.Errorf("Proposer is not allowed to add 2 headers within same period")
	}
}

// TestAddHeadersAtWrongPeriod tests proposer adds header in the wrong period.
func TestAddHeadersAtWrongPeriod(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header at wrong period, shard 0, period 0 and chunkroot 0xA.
	period0 := big.NewInt(0)
	shard0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	chunkRootEmpty := [32]byte{}
	_, err := smc.AddHeader(txOpts, shard0, period0, chunkRoot)
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}

	cr, err := smc.CollationRecords(&bind.CallOpts{}, shard0, period0)
	if cr.ChunkRoot != chunkRootEmpty {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Proposer adds header at wrong period, shard 0, period 2 and chunkroot 0xA.
	period2 := big.NewInt(2)
	_, err = smc.AddHeader(txOpts, shard0, period2, chunkRoot)
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}

	cr, err = smc.CollationRecords(&bind.CallOpts{}, shard0, period2)
	if cr.ChunkRoot != chunkRootEmpty {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Proposer adds header at correct period, shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	_, err = smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	cr, err = smc.CollationRecords(&bind.CallOpts{}, shard0, period1)
	if cr.ChunkRoot != chunkRootEmpty {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}
}

// TestSubmitVote tests notary submit votes in normal condition.
func TestSubmitVote(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
	smc.RegisterNotary(txOpts)
	backend.Commit()

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	txOpts.Value = big.NewInt(0)
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	// Notary 0 votes on header.
	c, err := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get notary vote count failed: %v", err)
	}
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}

	_, err = smc.SubmitVote(txOpts, shard0, period1, index0, chunkRoot)
	backend.Commit()
	if err != nil {
		t.Fatalf("Notary submits vote failed: %v", err)
	}

	// Check notary 0's vote is correctly casted.
	v, err := smc.HasVoted(&bind.CallOpts{}, shard0, index0)
	if err != nil {
		t.Fatalf("Check notary's vote failed: %v", err)
	}
	if !v {
		t.Errorf("Notary's indexd bit did not cast to 1 in index %v", index0)
	}
	c, err = smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}

	// Check header's submitted with the current period, should be period 1.
	p, err := smc.LastSubmittedCollation(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get period of last submitted header failed: %v", err)
	}
	if p.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect period submitted, want: 1, got: %v", p)
	}

	// Check header's approved with the current period, should be period 0.
	p, err = smc.LastApprovedCollation(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get period of last approved header failed: %v", err)
	}
	if p.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect period submitted, want: 0, got: %v", p)
	}

}

// TestSubmitVoteTwice tests notary tries to submit same vote twice.
func TestSubmitVoteTwice(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
	smc.RegisterNotary(txOpts)
	backend.Commit()

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	txOpts.Value = big.NewInt(0)
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	// Notary 0 votes on header.
	smc.SubmitVote(txOpts, shard0, period1, index0, chunkRoot)
	backend.Commit()

	// Check notary 0's vote is correctly casted.
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}

	// Notary 0 votes on header again, it should fail.
	_, err = smc.SubmitVote(txOpts, shard0, period1, index0, chunkRoot)
	if err == nil {
		t.Errorf("notary voting twice should have failed")
	}
	backend.Commit()

	// Check notary 0's vote is correctly casted.
	c, _ = smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteByNonEligibleNotary tests a non-eligible notary tries to submit vote.
func TestSubmitVoteByNonEligibleNotary(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	txOpts.Value = big.NewInt(0)
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	// Unregistered Notary 0 votes on header, it should fail.
	_, err = smc.SubmitVote(txOpts, shard0, period1, index0, chunkRoot)
	backend.Commit()
	if err == nil {
		t.Errorf("Non registered notary submits vote should have failed")
	}
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}
}

// TestSubmitVoteWithOutAHeader tests a notary tries to submit vote before header gets added.
func TestSubmitVoteWithOutAHeader(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers.
	smc.RegisterNotary(txOpts)
	backend.Commit()

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	txOpts.Value = big.NewInt(0)

	// Notary 0 votes on header, it should fail because no header has added.
	_, err := smc.SubmitVote(txOpts, shard0, period1, index0, chunkRoot)
	if err == nil {
		t.Errorf("Notary votes should have failed due to missing header")
	}
	backend.Commit()

	// Check notary 0's vote is correctly casted.
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteWithInvalidArgs tests notary submits vote using wrong chunkroot and period.
func TestSubmitVoteWithInvalidArgs(t *testing.T) {
	addr := crypto.PubkeyToAddress(mainKey.PublicKey)
	backend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: accountBalance2000Eth}})
	txOpts := bind.NewKeyedTransactor(mainKey)
	txOpts.Value = notaryDeposit
	_, _, smc, _ := deploySMCContract(backend, mainKey)

	// Notary 0 registers
	smc.RegisterNotary(txOpts)
	backend.Commit()

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	chunkRoot := [32]byte{'A'}
	txOpts.Value = big.NewInt(0)
	_, err := smc.AddHeader(txOpts, shard0, period1, chunkRoot)
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	backend.Commit()

	// Notary voting with incorrect period.
	period2 := big.NewInt(2)
	_, err = smc.SubmitVote(txOpts, shard0, period2, index0, chunkRoot)
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect period")
	}

	// Notary voting with incorrect chunk root.
	chunkRootWrong := [32]byte{'B'}
	_, err = smc.SubmitVote(txOpts, shard0, period1, index0, chunkRootWrong)
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect chunk root")
	}
}
