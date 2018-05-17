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

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	txOpts  *bind.TransactOpts
}

type testAcconts []testAccount

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

// initAccounts is a helper function to initialize backend with the n accounts,
// returns an array of testAccount struct.
func initAccounts(n int) (testAcconts, map[common.Address]core.GenesisAccount) {
	genesis := make(core.GenesisAlloc)
	var testAccounts []testAccount

	for i := 0; i < n; i++ {
		privKey, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		txOpts := bind.NewKeyedTransactor(privKey)
		testAccounts = append(testAccounts, testAccount{addr, privKey, txOpts})
		genesis[addr] = core.GenesisAccount{
			Balance: accountBalance2000Eth,
		}
	}
	return testAccounts, genesis
}

// deploySMCContract is a helper function for deploying SMC.
func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// registerNotaries is a helper function register notaries in batch,
// it also verifies notary registration is successful.
func (a testAcconts) registerNotaries(t *testing.T, backend *backends.SimulatedBackend, smc *SMC, deposit *big.Int, params ...int) error {
	for i := params[0]; i < params[1]; i++ {
		a[i].txOpts.Value = deposit
		_, err := smc.RegisterNotary(a[i].txOpts)
		if err != nil {
			return err
		}
		backend.Commit()

		notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, a[i].addr)
		if !notary.Deposited ||
			notary.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 ||
			notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
			t.Errorf("Incorrect notary registry. Want - deposited:true, index:%v, period:0"+
				"Got - deposited:%v, index:%v, period:%v ", i, notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
		}
	}
	return nil
}

// deregisterNotaries is a helper function that deregister notaries in batch,
// it also verifies notary deregistration is successful.
func (a testAcconts) deregisterNotaries(t *testing.T, backend *backends.SimulatedBackend, smc *SMC, params ...int) {
	for i := params[0]; i < params[1]; i++ {
		a[i].txOpts.Value = big.NewInt(0)
		_, err := smc.DeregisterNotary(a[i].txOpts)
		if err != nil {
			t.Fatalf("Failed to deregister notary: %v", err)
		}
		backend.Commit()
		notary, _ := smc.NotaryRegistry(&bind.CallOpts{}, a[i].addr)
		if notary.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {
			t.Errorf("Degistered period can not be 0 right after deregistration")
		}
	}
}

// addHeader is a helper function to add header to smc,
// it also verifies header is correctly added to the SMC.
func (a testAccount) addHeader(t *testing.T, backend *backends.SimulatedBackend, smc *SMC, shard *big.Int, period *big.Int, chunkRoot uint8) error {
	_, err := smc.AddHeader(a.txOpts, shard, period, [32]byte{chunkRoot})
	if err != nil {
		return err
	}
	backend.Commit()

	p, err := smc.LastSubmittedCollation(&bind.CallOpts{}, shard)
	if err != nil {
		t.Fatalf("Can't get last submitted collation's period number: %v", err)
	}
	if p.Cmp(period) != 0 {
		t.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}

	cr, err := smc.CollationRecords(&bind.CallOpts{}, shard, period)
	if cr.ChunkRoot != [32]byte{chunkRoot} {
		t.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}
	return nil
}

// addHeader is a helper function for notary to submit vote on a given header,
// it also verifies vote is properly submitted and casted.
func (a testAccount) submitVote(t *testing.T, backend *backends.SimulatedBackend, smc *SMC, shard *big.Int, period *big.Int, index *big.Int, chunkRoot uint8) error {
	_, err := smc.SubmitVote(a.txOpts, shard, period, index, [32]byte{chunkRoot})
	if err != nil {
		return err
	}
	backend.Commit()

	v, err := smc.HasVoted(&bind.CallOpts{}, shard, index)
	if err != nil {
		t.Fatalf("Check notary's vote failed: %v", err)
	}
	if !v {
		t.Errorf("Notary's indexd bit did not cast to 1 in index %v", index)
	}
	return nil
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
	// Initializes 3 accounts to register as notaries.
	const notaryCount = 3
	notaryPool, genesis := initAccounts(notaryCount)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Verify notary 0 has not registered.
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPool[0].addr)
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}
	if notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	// Test notary 0 has registered.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)

	// Test notary 1 and 2 have registered.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 1, 3)

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
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)
	if err := notaryPool.registerNotaries(t, backend, smc, notaryDepositInsufficient, 0, 1); err == nil {
		t.Errorf("Notary register should have failed with insufficient deposit")
	}
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

// TestNotaryDoubleRegisters tests notary registers twice.
func TestNotaryDoubleRegisters(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Notary 0 registers again. This time should fail
	if err := notaryPool.registerNotaries(t, backend, smc, big.NewInt(0), 0, 1); err == nil {
		t.Errorf("Notary register should have failed with double registers")
	}
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

}

// TestNotaryDeregister tests notary deregisters in a normal condition.
func TestNotaryDeregister(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward 100 blocks to check notary's deregistered period field is set correctly.
	for i := 0; i < FastForward100Blocks; i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
	notaryPool.deregisterNotaries(t, backend, smc, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

// TestNotaryDeregisterThenRegister tests notary deregisters then registers before lock up ends.
func TestNotaryDeregisterThenRegister(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward to the next period to deregister.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
	notaryPool.deregisterNotaries(t, backend, smc, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Notary 0 re-registers again.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}
}

// TestNotaryRelease tests notary releases in a normal condition.
func TestNotaryRelease(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward to the next period to deregister.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
	notaryPool.deregisterNotaries(t, backend, smc, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Fast forward until lockup ends.
	for i := 0; i < int(sharding.NotaryLockupLength*sharding.PeriodLength+1); i++ {
		backend.Commit()
	}

	// Notary 0 releases.
	_, err := smc.ReleaseNotary(notaryPool[0].txOpts)
	if err != nil {
		t.Fatalf("Failed to release notary: %v", err)
	}
	backend.Commit()
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPool[0].addr)
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}
	if notary.Deposited {
		t.Errorf("Notary deposit flag should be false after released")
	}
	balance, err := backend.BalanceAt(ctx, notaryPool[0].addr, nil)
	if err != nil {
		t.Errorf("Can't get account balance, err: %s", err)
	}
	if balance.Cmp(notaryDeposit) < 0 {
		t.Errorf("Notary did not receive deposit after lock up ends")
	}
}

// TestNotaryInstantRelease tests notary releases before lockup ends.
func TestNotaryInstantRelease(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Fast forward to the next period to deregister.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Notary 0 deregisters.
	notaryPool.deregisterNotaries(t, backend, smc, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 0, Got: %v", numNotaries)
	}

	// Notary 0 tries to release before lockup ends.
	_, err := smc.ReleaseNotary(notaryPool[0].txOpts)
	backend.Commit()
	notary, err := smc.NotaryRegistry(&bind.CallOpts{}, notaryPool[0].addr)
	if err != nil {
		t.Fatalf("Can't get notary registry info: %v", err)
	}
	if !notary.Deposited {
		t.Errorf("Notary deposit flag should be true before released")
	}
	balance, err := backend.BalanceAt(ctx, notaryPool[0].addr, nil)
	if balance.Cmp(notaryDeposit) > 0 {
		t.Errorf("Notary received deposit before lockup ends")
	}
}

// TestCommitteeListsAreDifferent tests different shards have different notary committee.
func TestCommitteeListsAreDifferent(t *testing.T) {
	notaryCount := 1000
	notaryPool, genesis := initAccounts(notaryCount)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Register 1000 notaries to SMC.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1000)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(int64(notaryCount))) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1000, Got: %v", numNotaries)
	}

	// Compare sampled first 5 notaries of shard 0 to shard 1, they should not be identical.
	for i := 0; i < 5; i++ {
		addr0, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		addr1, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(1))
		if addr0 == addr1 {
			t.Errorf("Shard 0 committee list is identical to shard 1's committee list")
		}
	}

}

// TestGetCommitteeWithNonMember tests unregistered notary tries to be in the committee.
func TestGetCommitteeWithNonMember(t *testing.T) {
	const notaryCount = 11
	notaryPool, genesis := initAccounts(notaryCount)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Register 10 notaries to SMC, leave 1 address free.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 10)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("Incorrect count from notary pool. Want: 10, Got: %v", numNotaries)
	}

	// Verify the unregistered account is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if notaryPool[10].addr == addr {
			t.Errorf("Account %s is not a notary", notaryPool[10].addr.String())
		}
	}
}

// TestGetCommitteeWithinSamePeriod tests notary registers and samples within the same period
func TestGetCommitteeWithinSamePeriod(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 1, Got: %v", numNotaries)
	}

	// Notary 0 samples for itself within the same period after registration
	sampledAddr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
	if notaryPool[0].addr != sampledAddr {
		t.Errorf("Unable to sample notary address within same period of registration, got addr: %v", sampledAddr)
	}
}

// TestGetCommitteeAfterDeregisters tests notary tries to be in committee after deregistered.
func TestGetCommitteeAfterDeregisters(t *testing.T) {
	const notaryCount = 10
	notaryPool, genesis := initAccounts(notaryCount)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Register 10 notaries to SMC.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, notaryCount)
	numNotaries, _ := smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 10, Got: %v", numNotaries)
	}

	// Deregister notary 0 from SMC.
	notaryPool.deregisterNotaries(t, backend, smc, 0, 1)
	numNotaries, _ = smc.NotaryPoolLength(&bind.CallOpts{})
	if numNotaries.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("Incorrect count from notary pool. Want: 9, Got: %v", numNotaries)
	}

	// Verify degistered notary 0 is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if notaryPool[0].addr == addr {
			t.Errorf("Account %s is not a notary", notaryPool[0].addr.String())
		}
	}
}

// TestNormalAddHeader tests proposer add header in normal condition.
func TestNormalAddHeader(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	err := notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Fast forward to the next period. Period 2.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 2 and chunkroot 0xB.
	err = notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(2), 'B')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
	// Proposer adds header consists shard 1, period 2 and chunkroot 0xC.
	err = notaryPool[0].addHeader(t, backend, smc, big.NewInt(1), big.NewInt(2), 'C')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
}

// TestAddTwoHeadersAtSamePeriod tests we can't add two headers within the same period.
func TestAddTwoHeadersAtSamePeriod(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	err := notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Proposer attempts to add another header chunkroot 0xB on the same period for the same shard.
	err = notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(1), 'B')
	if err == nil {
		t.Errorf("Proposer is not allowed to add 2 headers within same period")
	}
}

// TestAddHeadersAtWrongPeriod tests proposer adds header in the wrong period.
func TestAddHeadersAtWrongPeriod(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header at wrong period, shard 0, period 0 and chunkroot 0xA.
	err := notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(0), 'A')
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}
	// Proposer adds header at wrong period, shard 0, period 2 and chunkroot 0xA.
	err = notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(2), 'A')
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}
	// Proposer adds header at correct period, shard 0, period 1 and chunkroot 0xA.
	err = notaryPool[0].addHeader(t, backend, smc, big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}
}

// TestSubmitVote tests notary submit votes in normal condition.
func TestSubmitVote(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	notaryPool[0].txOpts.Value = big.NewInt(0)
	err := notaryPool[0].addHeader(t, backend, smc, shard0, period1, 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Notary 0 votes on header.
	c, err := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get notary vote count failed: %v", err)
	}
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}

	// Notary votes on the header that was submitted
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period1, index0, 'A')
	if err != nil {
		t.Fatalf("Notary submits vote failed: %v", err)
	}
	c, err = smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}

	// Check header's approved with the current period, should be period 0.
	p, err := smc.LastApprovedCollation(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get period of last approved header failed: %v", err)
	}
	if p.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect period submitted, want: 0, got: %v", p)
	}

}

// TestSubmitVoteTwice tests notary tries to submit same vote twice.
func TestSubmitVoteTwice(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	notaryPool[0].txOpts.Value = big.NewInt(0)
	err := notaryPool[0].addHeader(t, backend, smc, shard0, period1, 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Notary 0 votes on header.
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period1, index0, 'A')
	if err != nil {
		t.Fatalf("Notary submits vote failed: %v", err)
	}

	// Notary 0 votes on header again, it should fail.
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("notary voting twice should have failed")
	}

	// Check notary's vote count is correct in shard.
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteByNonEligibleNotary tests a non-eligible notary tries to submit vote.
func TestSubmitVoteByNonEligibleNotary(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	err := notaryPool[0].addHeader(t, backend, smc, shard0, period1, 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Unregistered Notary 0 votes on header, it should fail.
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("Non registered notary submits vote should have failed")
	}

	// Check notary's vote count is correct in shard
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}
}

// TestSubmitVoteWithOutAHeader tests a notary tries to submit vote before header gets added.
func TestSubmitVoteWithOutAHeader(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	notaryPool[0].txOpts.Value = big.NewInt(0)

	// Notary 0 votes on header, it should fail because no header has added.
	err := notaryPool[0].submitVote(t, backend, smc, shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("Notary votes should have failed due to missing header")
	}

	// Check notary's vote count is correct in shard
	c, _ := smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteWithInvalidArgs tests notary submits vote using wrong chunkroot and period.
func TestSubmitVoteWithInvalidArgs(t *testing.T) {
	notaryPool, genesis := initAccounts(1)
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, _ := deploySMCContract(backend, notaryPool[0].privKey)

	// Notary 0 registers.
	notaryPool.registerNotaries(t, backend, smc, notaryDeposit, 0, 1)

	// Fast forward to the next period to submit header. Period 1.
	for i := 0; i < int(sharding.PeriodLength); i++ {
		backend.Commit()
	}

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	notaryPool[0].txOpts.Value = big.NewInt(0)
	err := notaryPool[0].addHeader(t, backend, smc, shard0, period1, 'A')
	if err != nil {
		t.Fatalf("Proposer adds header failed: %v", err)
	}

	// Notary voting with incorrect period.
	period2 := big.NewInt(2)
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period2, index0, 'A')
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect period")
	}

	// Notary voting with incorrect chunk root.
	err = notaryPool[0].submitVote(t, backend, smc, shard0, period2, index0, 'B')
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect chunk root")
	}
}
