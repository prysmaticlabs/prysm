package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
)

type smcTestHelper struct {
	testAccounts []testAccount
	backend      *backends.SimulatedBackend
	smc          *SMC
}

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	txOpts  *bind.TransactOpts
}

var (
	accountBalance2000Eth, _     = new(big.Int).SetString("2000000000000000000000", 10)
	notaryDepositInsufficient, _ = new(big.Int).SetString("999000000000000000000", 10)
	notaryDeposit, _             = new(big.Int).SetString("1000000000000000000000", 10)
	FastForward100Blocks         = 100
	ctx                          = context.Background()
)

// newSMCTestHelper is a helper function to initialize backend with the n test accounts,
// a smc struct to interact with smc, and a simulated back end to for deployment.
func newSMCTestHelper(n int) (*smcTestHelper, error) {
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
	backend := backends.NewSimulatedBackend(genesis)
	_, _, smc, err := deploySMCContract(backend, testAccounts[0].privKey)
	if err != nil {
		return nil, err
	}
	return &smcTestHelper{testAccounts: testAccounts, backend: backend, smc: smc}, nil
}

// deploySMCContract is a helper function for deploying SMC.
func deploySMCContract(backend *backends.SimulatedBackend, key *ecdsa.PrivateKey) (common.Address, *types.Transaction, *SMC, error) {
	transactOpts := bind.NewKeyedTransactor(key)
	defer backend.Commit()
	return DeploySMC(transactOpts, backend)
}

// fastForward is a helper function to skip through n period.
func (s *smcTestHelper) fastForward(p int) {
	for i := 0; i < p*int(sharding.PeriodLength); i++ {
		s.backend.Commit()
	}
}

// registerNotaries is a helper function register notaries in batch.
func (s *smcTestHelper) registerNotaries(deposit *big.Int, params ...int) error {
	for i := params[0]; i < params[1]; i++ {
		s.testAccounts[i].txOpts.Value = deposit
		_, err := s.smc.RegisterNotary(s.testAccounts[i].txOpts)
		if err != nil {
			return err
		}
		s.backend.Commit()

		notary, _ := s.smc.NotaryRegistry(&bind.CallOpts{}, s.testAccounts[i].addr)
		if !notary.Deposited ||
			notary.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 ||
			notary.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
			return fmt.Errorf("Incorrect notary registry. Want - deposited:true, index:%v, period:0"+
				"Got - deposited:%v, index:%v, period:%v ", i, notary.Deposited, notary.PoolIndex, notary.DeregisteredPeriod)
		}
	}
	// Filter SMC logs by notaryRegistered.
	log, err := s.smc.FilterNotaryRegistered(&bind.FilterOpts{})
	if err != nil {
		return err
	}
	// Iterate notaryRegistered logs, compare each address and poolIndex.
	for i := 0; i < params[1]; i++ {
		log.Next()
		if log.Event.Notary != s.testAccounts[i].addr {
			return fmt.Errorf("incorrect address in notaryRegistered log. Want: %v Got: %v", s.testAccounts[i].addr, log.Event.Notary)
		}
		// Verify notaryPoolIndex is incremental starting from 1st registered Notary.
		if log.Event.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 {
			return fmt.Errorf("incorrect index in notaryRegistered log. Want: %v Got: %v", i, log.Event.Notary)
		}
	}
	return nil
}

// deregisterNotaries is a helper function that deregister notaries in batch.
func (s *smcTestHelper) deregisterNotaries(params ...int) error {
	for i := params[0]; i < params[1]; i++ {
		s.testAccounts[i].txOpts.Value = big.NewInt(0)
		_, err := s.smc.DeregisterNotary(s.testAccounts[i].txOpts)
		if err != nil {
			return fmt.Errorf("Failed to deregister notary: %v", err)
		}
		s.backend.Commit()
		notary, _ := s.smc.NotaryRegistry(&bind.CallOpts{}, s.testAccounts[i].addr)
		if notary.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {
			return fmt.Errorf("Degistered period can not be 0 right after deregistration")
		}
	}
	// Filter SMC logs by notaryDeregistered.
	log, err := s.smc.FilterNotaryDeregistered(&bind.FilterOpts{})
	if err != nil {
		return err
	}
	// Iterate notaryDeregistered logs, compare each address, poolIndex and verify period is set.
	for i := 0; i < params[1]; i++ {
		log.Next()
		if log.Event.Notary != s.testAccounts[i].addr {
			return fmt.Errorf("incorrect address in notaryDeregistered log. Want: %v Got: %v", s.testAccounts[i].addr, log.Event.Notary)
		}
		if log.Event.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 {
			return fmt.Errorf("incorrect index in notaryDeregistered log. Want: %v Got: %v", i, log.Event.Notary)
		}
		if log.Event.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {
			return fmt.Errorf("incorrect period in notaryDeregistered log. Got: %v", log.Event.DeregisteredPeriod)
		}
	}
	return nil
}

// addHeader is a helper function to add header to smc.
func (s *smcTestHelper) addHeader(a *testAccount, shard *big.Int, period *big.Int, chunkRoot uint8) error {
	_, err := s.smc.AddHeader(a.txOpts, shard, period, [32]byte{chunkRoot})
	if err != nil {
		return err
	}
	s.backend.Commit()

	p, err := s.smc.LastSubmittedCollation(&bind.CallOpts{}, shard)
	if err != nil {
		return fmt.Errorf("Can't get last submitted collation's period number: %v", err)
	}
	if p.Cmp(period) != 0 {
		return fmt.Errorf("Incorrect last period, when header was added. Got: %v", p)
	}

	cr, err := s.smc.CollationRecords(&bind.CallOpts{}, shard, period)
	if cr.ChunkRoot != [32]byte{chunkRoot} {
		return fmt.Errorf("Chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Filter SMC logs by headerAdded.
	shardIndex := []*big.Int{shard}
	logPeriod := uint64(period.Int64() * sharding.PeriodLength)
	log, err := s.smc.FilterHeaderAdded(&bind.FilterOpts{Start: logPeriod}, shardIndex)
	if err != nil {
		return err
	}
	log.Next()
	if log.Event.ProposerAddress != s.testAccounts[0].addr {
		return fmt.Errorf("incorrect proposer address in addHeader log. Want: %v Got: %v", s.testAccounts[0].addr, log.Event.ProposerAddress)
	}
	if log.Event.ChunkRoot != [32]byte{chunkRoot} {
		return fmt.Errorf("chunk root missmatch in addHeader log. Want: %v Got: %v", [32]byte{chunkRoot}, log.Event.ChunkRoot)
	}
	return nil
}

// submitVote is a helper function for notary to submit vote on a given header.
func (s *smcTestHelper) submitVote(a *testAccount, shard *big.Int, period *big.Int, index *big.Int, chunkRoot uint8) error {
	_, err := s.smc.SubmitVote(a.txOpts, shard, period, index, [32]byte{chunkRoot})
	if err != nil {
		return fmt.Errorf("Notary submit vote failed: %v", err)
	}
	s.backend.Commit()

	v, err := s.smc.HasVoted(&bind.CallOpts{}, shard, index)
	if err != nil {
		return fmt.Errorf("Check notary's vote failed: %v", err)
	}
	if !v {
		return fmt.Errorf("Notary's indexd bit did not cast to 1 in index %v", index)
	}
	// Filter SMC logs by submitVote.
	shardIndex := []*big.Int{shard}
	logPeriod := uint64(period.Int64() * sharding.PeriodLength)
	log, err := s.smc.FilterVoteSubmitted(&bind.FilterOpts{Start: logPeriod}, shardIndex)
	if err != nil {
		return err
	}
	log.Next()
	if log.Event.NotaryAddress != a.addr {
		return fmt.Errorf("incorrect notary address in submitVote log. Want: %v Got: %v", s.testAccounts[0].addr, a.addr)
	}
	if log.Event.ChunkRoot != [32]byte{chunkRoot} {
		return fmt.Errorf("chunk root missmatch in submitVote log. Want: %v Got: %v", common.BytesToHash([]byte{chunkRoot}), common.BytesToHash(log.Event.ChunkRoot[:]))
	}
	return nil
}

// checkNotaryPoolLength is a helper function to verify current notary pool
// length is equal to n.
func checkNotaryPoolLength(smc *SMC, n *big.Int) error {
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("Failed to get notary pool length: %v", err)
	}
	if numNotaries.Cmp(n) != 0 {
		return fmt.Errorf("Incorrect count from notary pool. Want: %v, Got: %v", n, numNotaries)
	}
	return nil
}

// TestContractCreation tests SMC smart contract can successfully be deployed.
func TestContractCreation(t *testing.T) {
	_, err := newSMCTestHelper(1)
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// TestNotaryRegister tests notary registers in a normal condition.
func TestNotaryRegister(t *testing.T) {
	// Initializes 3 accounts to register as notaries.
	const notaryCount = 3
	s, _ := newSMCTestHelper(notaryCount)

	// Verify notary 0 has not registered.
	notary, err := s.smc.NotaryRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get notary registry info: %v", err)
	}
	if notary.Deposited {
		t.Errorf("Notary has not registered. Got deposited flag: %v", notary.Deposited)
	}

	// Test notary 0 has registered.
	err = s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Register notary failed: %v", err)
	}
	// Test notary 1 and 2 have registered.
	err = s.registerNotaries(notaryDeposit, 1, 3)
	if err != nil {
		t.Errorf("Register notary failed: %v", err)
	}
	// Check total numbers of notaries in pool, should be 3
	err = checkNotaryPoolLength(s.smc, big.NewInt(notaryCount))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
}

// TestNotaryRegisterInsufficientEther tests notary registers with insufficient deposit.
func TestNotaryRegisterInsufficientEther(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	if err := s.registerNotaries(notaryDepositInsufficient, 0, 1); err == nil {
		t.Errorf("Notary register should have failed with insufficient deposit")
	}
}

// TestNotaryDoubleRegisters tests notary registers twice.
func TestNotaryDoubleRegisters(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Register notary failed: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Notary 0 registers again, This time should fail.
	if err = s.registerNotaries(big.NewInt(0), 0, 1); err == nil {
		t.Errorf("Notary register should have failed with double registers")
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
}

// TestNotaryDeregister tests notary deregisters in a normal condition.
func TestNotaryDeregister(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
	// Fast forward 20 periods to check notary's deregistered period field is set correctly.
	s.fastForward(20)

	// Notary 0 deregisters.
	s.deregisterNotaries(0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
}

// TestNotaryDeregisterThenRegister tests notary deregisters then registers before lock up ends.
func TestNotaryDeregisterThenRegister(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to register notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Notary 0 deregisters.
	s.deregisterNotaries(0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Notary 0 re-registers again.
	err = s.registerNotaries(notaryDeposit, 0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
}

// TestNotaryRelease tests notary releases in a normal condition.
func TestNotaryRelease(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to register notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Notary 0 deregisters.
	s.deregisterNotaries(0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Fast forward until lockup ends.
	s.fastForward(int(sharding.NotaryLockupLength + 1))

	// Notary 0 releases.
	_, err = s.smc.ReleaseNotary(s.testAccounts[0].txOpts)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	s.backend.Commit()
	notary, err := s.smc.NotaryRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get notary registry info: %v", err)
	}
	if notary.Deposited {
		t.Errorf("Notary deposit flag should be false after released")
	}
	balance, err := s.backend.BalanceAt(ctx, s.testAccounts[0].addr, nil)
	if err != nil {
		t.Errorf("Can't get account balance, err: %s", err)
	}
	if balance.Cmp(notaryDeposit) < 0 {
		t.Errorf("Notary did not receive deposit after lock up ends")
	}
}

// TestNotaryInstantRelease tests notary releases before lockup ends.
func TestNotaryInstantRelease(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Notary 0 deregisters.
	s.deregisterNotaries(0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Notary 0 tries to release before lockup ends.
	_, err = s.smc.ReleaseNotary(s.testAccounts[0].txOpts)
	s.backend.Commit()
	notary, err := s.smc.NotaryRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get notary registry info: %v", err)
	}
	if !notary.Deposited {
		t.Errorf("Notary deposit flag should be true before released")
	}
	balance, err := s.backend.BalanceAt(ctx, s.testAccounts[0].addr, nil)
	if balance.Cmp(notaryDeposit) > 0 {
		t.Errorf("Notary received deposit before lockup ends")
	}
}

// TestCommitteeListsAreDifferent tests different shards have different notary committee.
func TestCommitteeListsAreDifferent(t *testing.T) {
	const notaryCount = 1000
	s, _ := newSMCTestHelper(notaryCount)

	// Register 1000 notaries to s.smc.
	err := s.registerNotaries(notaryDeposit, 0, 1000)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1000))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Compare sampled first 5 notaries of shard 0 to shard 1, they should not be identical.
	for i := 0; i < 5; i++ {
		addr0, _ := s.smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		addr1, _ := s.smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(1))
		if addr0 == addr1 {
			t.Errorf("Shard 0 committee list is identical to shard 1's committee list")
		}
	}
}

// TestGetCommitteeWithNonMember tests unregistered notary tries to be in the committee.
func TestGetCommitteeWithNonMember(t *testing.T) {
	const notaryCount = 11
	s, _ := newSMCTestHelper(notaryCount)

	// Register 10 notaries to s.smc, leave 1 address free.
	err := s.registerNotaries(notaryDeposit, 0, 10)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(10))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Verify the unregistered account is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := s.smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if s.testAccounts[10].addr == addr {
			t.Errorf("Account %s is not a notary", s.testAccounts[10].addr.String())
		}
	}
}

// TestGetCommitteeWithinSamePeriod tests notary registers and samples within the same period.
func TestGetCommitteeWithinSamePeriod(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Notary 0 samples for itself within the same period after registration.
	sampledAddr, _ := s.smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
	if s.testAccounts[0].addr != sampledAddr {
		t.Errorf("Unable to sample notary address within same period of registration, got addr: %v", sampledAddr)
	}
}

// TestGetCommitteeAfterDeregisters tests notary tries to be in committee after deregistered.
func TestGetCommitteeAfterDeregisters(t *testing.T) {
	const notaryCount = 10
	s, _ := newSMCTestHelper(notaryCount)

	// Register 10 notaries to s.smc.
	err := s.registerNotaries(notaryDeposit, 0, notaryCount)
	if err != nil {
		t.Errorf("Failed to release notary: %v", err)
	}
	err = checkNotaryPoolLength(s.smc, big.NewInt(10))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Deregister notary 0 from s.smc.
	s.deregisterNotaries(0, 1)
	err = checkNotaryPoolLength(s.smc, big.NewInt(9))
	if err != nil {
		t.Errorf("Notary pool length mismatched: %v", err)
	}

	// Verify degistered notary 0 is not in the notary pool list.
	for i := 0; i < 10; i++ {
		addr, _ := s.smc.GetNotaryInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if s.testAccounts[0].addr == addr {
			t.Errorf("Account %s is not a notary", s.testAccounts[0].addr.String())
		}
	}
}

// TestNormalAddHeader tests proposer add header in normal condition.
func TestNormalAddHeader(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	err := s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 2 and chunkroot 0xB.
	err = s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(2), 'B')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}
	// Proposer adds header consists shard 1, period 2 and chunkroot 0xC.
	err = s.addHeader(&s.testAccounts[0], big.NewInt(1), big.NewInt(2), 'C')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}
}

// TestAddTwoHeadersAtSamePeriod tests we can't add two headers within the same period.
func TestAddTwoHeadersAtSamePeriod(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	err := s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Proposer attempts to add another header chunkroot 0xB on the same period for the same shard.
	err = s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(1), 'B')
	if err == nil {
		t.Errorf("Proposer is not allowed to add 2 headers within same period")
	}
}

// TestAddHeadersAtWrongPeriod tests proposer adds header in the wrong period.
func TestAddHeadersAtWrongPeriod(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	s.fastForward(1)

	// Proposer adds header at wrong period, shard 0, period 0 and chunkroot 0xA.
	err := s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(0), 'A')
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}
	// Proposer adds header at wrong period, shard 0, period 2 and chunkroot 0xA.
	err = s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(2), 'A')
	if err == nil {
		t.Errorf("Proposer adds header at wrong period should have failed")
	}
	// Proposer adds header at correct period, shard 0, period 1 and chunkroot 0xA.
	err = s.addHeader(&s.testAccounts[0], big.NewInt(0), big.NewInt(1), 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}
}

// TestSubmitVote tests notary submit votes in normal condition.
func TestSubmitVote(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	err = s.addHeader(&s.testAccounts[0], shard0, period1, 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Notary 0 votes on header.
	c, err := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Errorf("Get notary vote count failed: %v", err)
	}
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}

	// Notary votes on the header that was submitted.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err != nil {
		t.Fatalf("Notary submits vote failed: %v", err)
	}
	c, err = s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}

	// Check header's approved with the current period, should be period 0.
	p, err := s.smc.LastApprovedCollation(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Fatalf("Get period of last approved header failed: %v", err)
	}
	if p.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect period submitted, want: 0, got: %v", p)
	}

}

// TestSubmitVoteTwice tests notary tries to submit same vote twice.
func TestSubmitVoteTwice(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	err = s.addHeader(&s.testAccounts[0], shard0, period1, 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Notary 0 votes on header.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err != nil {
		t.Errorf("Notary submits vote failed: %v", err)
	}

	// Notary 0 votes on header again, it should fail.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("notary voting twice should have failed")
	}

	// Check notary's vote count is correct in shard.
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteByNonEligibleNotary tests a non-eligible notary tries to submit vote.
func TestSubmitVoteByNonEligibleNotary(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	err := s.addHeader(&s.testAccounts[0], shard0, period1, 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Unregistered Notary 0 votes on header, it should fail.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("Non registered notary submits vote should have failed")
	}

	// Check notary's vote count is correct in shard.
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 0, got: %v", c)
	}
}

// TestSubmitVoteWithOutAHeader tests a notary tries to submit vote before header gets added.
func TestSubmitVoteWithOutAHeader(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)

	// Notary 0 votes on header, it should fail because no header has added.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("Notary votes should have failed due to missing header")
	}

	// Check notary's vote count is correct in shard
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect notary vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVoteWithInvalidArgs tests notary submits vote using wrong chunkroot and period.
func TestSubmitVoteWithInvalidArgs(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Notary 0 registers.
	err := s.registerNotaries(notaryDeposit, 0, 1)
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	err = s.addHeader(&s.testAccounts[0], shard0, period1, 'A')
	if err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Notary voting with incorrect period.
	period2 := big.NewInt(2)
	err = s.submitVote(&s.testAccounts[0], shard0, period2, index0, 'A')
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect period")
	}

	// Notary voting with incorrect chunk root.
	err = s.submitVote(&s.testAccounts[0], shard0, period2, index0, 'B')
	if err == nil {
		t.Errorf("Notary votes should have failed due to incorrect chunk root")
	}
}
