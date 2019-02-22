package smc

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
	accountBalance2000Eth, _       = new(big.Int).SetString("2000000000000000000000", 10)
	attesterDepositInsufficient, _ = new(big.Int).SetString("999000000000000000000", 10)
	attesterDeposit, _             = new(big.Int).SetString("1000000000000000000000", 10)
	ctx                            = context.Background()
	periodLength                   = 5
	attesterLockupLength           = 16128
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
	backend := backends.NewSimulatedBackend(genesis, 2100000)
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
	for i := 0; i < p*int(periodLength); i++ {
		s.backend.Commit()
	}
}

// registerAttesters is a helper function register attesters in batch.
func (s *smcTestHelper) registerAttesters(deposit *big.Int, params ...int) error {
	for i := params[0]; i < params[1]; i++ {
		s.testAccounts[i].txOpts.Value = deposit
		_, err := s.smc.RegisterAttester(s.testAccounts[i].txOpts)
		if err != nil {
			return err
		}
		s.backend.Commit()

		attester, _ := s.smc.AttesterRegistry(&bind.CallOpts{}, s.testAccounts[i].addr)
		if !attester.Deposited ||
			attester.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 ||
			attester.DeregisteredPeriod.Cmp(big.NewInt(0)) != 0 {
			return fmt.Errorf("Incorrect attester registry. Want - deposited:true, index:%v, period:0"+
				"Got - deposited:%v, index:%v, period:%v ", i, attester.Deposited, attester.PoolIndex, attester.DeregisteredPeriod)
		}
	}
	// Filter SMC logs by attesterRegistered.
	log, err := s.smc.FilterAttesterRegistered(&bind.FilterOpts{})
	if err != nil {
		return err
	}
	// Iterate attesterRegistered logs, compare each address and poolIndex.
	for i := 0; i < params[1]; i++ {
		log.Next()
		if log.Event.Attester != s.testAccounts[i].addr {
			return fmt.Errorf("incorrect address in attesterRegistered log. Want: %v Got: %v", s.testAccounts[i].addr, log.Event.Attester)
		}
		// Verify attesterPoolIndex is incremental starting from 1st registered Attester.
		if log.Event.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 {
			return fmt.Errorf("incorrect index in attesterRegistered log. Want: %v Got: %v", i, log.Event.Attester)
		}
	}
	return nil
}

// deregisterAttesters is a helper function that deregister attesters in batch.
func (s *smcTestHelper) deregisterAttesters(params ...int) error {
	for i := params[0]; i < params[1]; i++ {
		s.testAccounts[i].txOpts.Value = big.NewInt(0)
		_, err := s.smc.DeregisterAttester(s.testAccounts[i].txOpts)
		if err != nil {
			return fmt.Errorf("failed to deregister attester: %v", err)
		}
		s.backend.Commit()
		attester, _ := s.smc.AttesterRegistry(&bind.CallOpts{}, s.testAccounts[i].addr)
		if attester.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {
			return fmt.Errorf("degistered period can not be 0 right after deregistration")
		}
	}
	// Filter SMC logs by attesterDeregistered.
	log, err := s.smc.FilterAttesterDeregistered(&bind.FilterOpts{})
	if err != nil {
		return err
	}
	// Iterate attesterDeregistered logs, compare each address, poolIndex and verify period is set.
	for i := 0; i < params[1]; i++ {
		log.Next()
		if log.Event.Attester != s.testAccounts[i].addr {
			return fmt.Errorf("incorrect address in attesterDeregistered log. Want: %v Got: %v", s.testAccounts[i].addr, log.Event.Attester)
		}
		if log.Event.PoolIndex.Cmp(big.NewInt(int64(i))) != 0 {
			return fmt.Errorf("incorrect index in attesterDeregistered log. Want: %v Got: %v", i, log.Event.Attester)
		}
		if log.Event.DeregisteredPeriod.Cmp(big.NewInt(0)) == 0 {
			return fmt.Errorf("incorrect period in attesterDeregistered log. Got: %v", log.Event.DeregisteredPeriod)
		}
	}
	return nil
}

// addHeader is a helper function to add header to smc.
func (s *smcTestHelper) addHeader(a *testAccount, shard *big.Int, period *big.Int, chunkRoot uint8) error {
	sig := [32]byte{'S', 'I', 'G', 'N', 'A', 'T', 'U', 'R', 'E'}
	_, err := s.smc.AddHeader(a.txOpts, shard, period, [32]byte{chunkRoot}, sig)
	if err != nil {
		return err
	}
	s.backend.Commit()

	p, err := s.smc.LastSubmittedCollation(&bind.CallOpts{}, shard)
	if err != nil {
		return fmt.Errorf("can't get last submitted collation's period number: %v", err)
	}
	if p.Cmp(period) != 0 {
		return fmt.Errorf("incorrect last period, when header was added. Got: %v", p)
	}

	cr, err := s.smc.CollationRecords(&bind.CallOpts{}, shard, period)
	if err != nil {
		return err
	}
	if cr.ChunkRoot != [32]byte{chunkRoot} {
		return fmt.Errorf("chunkroot mismatched. Want: %v, Got: %v", chunkRoot, cr)
	}

	// Filter SMC logs by headerAdded.
	shardIndex := []*big.Int{shard}
	logPeriod := uint64(period.Int64() * int64(periodLength))
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

// submitVote is a helper function for attester to submit vote on a given header.
func (s *smcTestHelper) submitVote(a *testAccount, shard *big.Int, period *big.Int, index *big.Int, chunkRoot uint8) error {
	_, err := s.smc.SubmitVote(a.txOpts, shard, period, index, [32]byte{chunkRoot})
	if err != nil {
		return fmt.Errorf("attester submit vote failed: %v", err)
	}
	s.backend.Commit()

	v, err := s.smc.HasVoted(&bind.CallOpts{}, shard, index)
	if err != nil {
		return fmt.Errorf("check attester's vote failed: %v", err)
	}
	if !v {
		return fmt.Errorf("attester's indexd bit did not cast to 1 in index %v", index)
	}
	// Filter SMC logs by submitVote.
	shardIndex := []*big.Int{shard}
	logPeriod := uint64(period.Int64() * int64(periodLength))
	log, err := s.smc.FilterVoteSubmitted(&bind.FilterOpts{Start: logPeriod}, shardIndex)
	if err != nil {
		return err
	}
	log.Next()
	if log.Event.AttesterAddress != a.addr {
		return fmt.Errorf("incorrect attester address in submitVote log. Want: %v Got: %v", s.testAccounts[0].addr, a.addr)
	}
	if log.Event.ChunkRoot != [32]byte{chunkRoot} {
		return fmt.Errorf("chunk root missmatch in submitVote log. Want: %v Got: %v", common.BytesToHash([]byte{chunkRoot}), common.BytesToHash(log.Event.ChunkRoot[:]))
	}
	return nil
}

// checkAttesterPoolLength is a helper function to verify current attester pool
// length is equal to n.
func checkAttesterPoolLength(smc *SMC, n *big.Int) error {
	numAttesters, err := smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		return fmt.Errorf("failed to get attester pool length: %v", err)
	}
	if numAttesters.Cmp(n) != 0 {
		return fmt.Errorf("incorrect count from attester pool. Want: %v, Got: %v", n, numAttesters)
	}
	return nil
}

// TestContractCreation_OK tests SMC smart contract can successfully be deployed.
func TestContractCreation_OK(t *testing.T) {
	_, err := newSMCTestHelper(1)
	if err != nil {
		t.Fatalf("can't deploy SMC: %v", err)
	}
}

// TestRegisterAttesters_OK tests attester registers in a normal condition.
func TestRegisterAttesters_OK(t *testing.T) {
	// Initializes 3 accounts to register as attesters.
	const attesterCount = 3
	s, _ := newSMCTestHelper(attesterCount)

	// Verify attester 0 has not registered.
	attester, err := s.smc.AttesterRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get attester registry info: %v", err)
	}
	if attester.Deposited {
		t.Errorf("Attester has not registered. Got deposited flag: %v", attester.Deposited)
	}

	// Test attester 0 has registered.
	err = s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Register attester failed: %v", err)
	}
	// Test attester 1 and 2 have registered.
	err = s.registerAttesters(attesterDeposit, 1, 3)
	if err != nil {
		t.Errorf("Register attester failed: %v", err)
	}
	// Check total numbers of attesters in pool, should be 3
	err = checkAttesterPoolLength(s.smc, big.NewInt(attesterCount))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
}

// TestRegisterAttesters_InsufficientEther tests attester registration with insufficient deposit.
func TestRegisterAttesters_InsufficientEther(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	if err := s.registerAttesters(attesterDepositInsufficient, 0, 1); err == nil {
		t.Errorf("Attester register should have failed with insufficient deposit")
	}
}

// TestRegisterAttesters_DoubleRegister tests the same attester registering twice.
func TestRegisterAttesters_DoubleRegister(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	err := s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Register attester failed: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Attester 0 registers again, This time should fail.
	if err = s.registerAttesters(big.NewInt(0), 0, 1); err == nil {
		t.Errorf("Attester register should have failed with double registers")
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
}

// TestDeregisterAttesters_OK tests attester deregisters in a normal condition.
func TestDeregisterAttesters_OK(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	err := s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
	// Fast forward 20 periods to check attester's deregistered period field is set correctly.
	s.fastForward(20)

	// Attester 0 deregisters.
	s.deregisterAttesters(0, 1)
	err = checkAttesterPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
}

// TestDeregisterAttesters_RegisterAgain tests attester deregisteration and then registers before lock up ends.
func TestDeregisterAttesters_RegisterAgain(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	err := s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to register attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Attester 0 deregisters.
	s.deregisterAttesters(0, 1)
	err = checkAttesterPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Attester 0 re-registers again.
	err = s.registerAttesters(attesterDeposit, 0, 1)
	if err == nil {
		t.Error("Expected re-registration to fail")
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
}

// TestAttesterRelease tests attester releases in a normal condition.
func TestReleaseAttester_OK(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	err := s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to register attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Attester 0 deregisters.
	s.deregisterAttesters(0, 1)
	err = checkAttesterPoolLength(s.smc, big.NewInt(0))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Fast forward until lockup ends.
	s.fastForward(int(attesterLockupLength + 1))

	// Attester 0 releases.
	_, err = s.smc.ReleaseAttester(s.testAccounts[0].txOpts)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	s.backend.Commit()
	attester, err := s.smc.AttesterRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get attester registry info: %v", err)
	}
	if attester.Deposited {
		t.Errorf("Attester deposit flag should be false after released")
	}
	balance, err := s.backend.BalanceAt(ctx, s.testAccounts[0].addr, nil)
	if err != nil {
		t.Errorf("Can't get account balance, err: %s", err)
	}
	if balance.Cmp(attesterDeposit) < 0 {
		t.Errorf("Attester did not receive deposit after lock up ends")
	}
}

// TestAttesterInstantRelease tests attester releases before lockup ends.
func TestReleaseAttester_BeforeLockup(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	if err := s.registerAttesters(attesterDeposit, 0, 1); err != nil {
		t.Error(err)
	}
	if err := checkAttesterPoolLength(s.smc, big.NewInt(1)); err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}
	s.fastForward(1)

	// Attester 0 deregisters.
	s.deregisterAttesters(0, 1)
	if err := checkAttesterPoolLength(s.smc, big.NewInt(0)); err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Attester 0 tries to release before lockup ends.
	if _, err := s.smc.ReleaseAttester(s.testAccounts[0].txOpts); err == nil {
		t.Error("Expected release attester to fail")
	}
	s.backend.Commit()
	attester, err := s.smc.AttesterRegistry(&bind.CallOpts{}, s.testAccounts[0].addr)
	if err != nil {
		t.Errorf("Can't get attester registry info: %v", err)
	}
	if !attester.Deposited {
		t.Errorf("Attester deposit flag should be true before released")
	}
	balance, err := s.backend.BalanceAt(ctx, s.testAccounts[0].addr, nil)
	if err != nil {
		t.Error(err)
	}
	if balance.Cmp(attesterDeposit) > 0 {
		t.Errorf("Attester received deposit before lockup ends")
	}
}

// TestCommitteeListsAreDifferent tests that different shards should have different attester committee.
func TestCommitteeListsAreDifferent_OK(t *testing.T) {
	const attesterCount = 1000
	s, _ := newSMCTestHelper(attesterCount)

	// Register 1000 attesters to s.smc.
	err := s.registerAttesters(attesterDeposit, 0, 1000)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1000))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Compare sampled first 5 attesters of shard 0 to shard 1, they should not be identical.
	for i := 0; i < 5; i++ {
		addr0, _ := s.smc.GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(0))
		addr1, _ := s.smc.GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(1))
		if addr0 == addr1 {
			t.Errorf("Shard 0 committee list is identical to shard 1's committee list")
		}
	}
}

// TestGetAttesterInCommittee_NonMember tests unregistered attester trying to be in the committee.
func TestGetAttesterInCommittee_NonMember(t *testing.T) {
	const attesterCount = 11
	s, _ := newSMCTestHelper(attesterCount)

	// Register 10 attesters to s.smc, leave 1 address free.
	err := s.registerAttesters(attesterDeposit, 0, 10)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(10))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Verify the unregistered account is not in the attester pool list.
	for i := 0; i < 10; i++ {
		addr, _ := s.smc.GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if s.testAccounts[10].addr == addr {
			t.Errorf("Account %s is not a attester", s.testAccounts[10].addr.String())
		}
	}
}

// TestGetAttesterInCommittee_SamePeriod tests attester registeration and samples within the same period.
func TestGetAttesterInCommittee_SamePeriod(t *testing.T) {
	s, _ := newSMCTestHelper(1)

	// Attester 0 registers.
	err := s.registerAttesters(attesterDeposit, 0, 1)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(1))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Attester 0 samples for itself within the same period after registration.
	sampledAddr, _ := s.smc.GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(0))
	if s.testAccounts[0].addr != sampledAddr {
		t.Errorf("Unable to sample attester address within same period of registration, got addr: %v", sampledAddr)
	}
}

// TestGetCommitteeAfterDeregisters tests attester tries to be in committee after deregistered.
func TestGetAttesterInCommittee_AfterDeregister(t *testing.T) {
	const attesterCount = 10
	s, _ := newSMCTestHelper(attesterCount)

	// Register 10 attesters to s.smc.
	err := s.registerAttesters(attesterDeposit, 0, attesterCount)
	if err != nil {
		t.Errorf("Failed to release attester: %v", err)
	}
	err = checkAttesterPoolLength(s.smc, big.NewInt(10))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Deregister attester 0 from s.smc.
	s.deregisterAttesters(0, 1)
	err = checkAttesterPoolLength(s.smc, big.NewInt(9))
	if err != nil {
		t.Errorf("Attester pool length mismatched: %v", err)
	}

	// Verify degistered attester 0 is not in the attester pool list.
	for i := 0; i < 10; i++ {
		addr, _ := s.smc.GetAttesterInCommittee(&bind.CallOpts{}, big.NewInt(0))
		if s.testAccounts[0].addr == addr {
			t.Errorf("Account %s is not a attester", s.testAccounts[0].addr.String())
		}
	}
}

// TestAddHeader_OK tests proposer adding header in normal condition.
func TestAddHeader_OK(t *testing.T) {
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
func TestAddHeader_TwoSamePeriod(t *testing.T) {
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

// TestAddHeader_WrongPeriod tests proposer adding the header in the wrong period.
func TestAddHeader_WrongPeriod(t *testing.T) {
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

// TestSubmitVote_OK tests attester submitting votes in normal condition.
func TestSubmitVote_OK(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Attester 0 registers.
	if err := s.registerAttesters(attesterDeposit, 0, 1); err != nil {
		t.Error(err)
	}
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	if err := s.addHeader(&s.testAccounts[0], shard0, period1, 'A'); err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Attester 0 votes on header.
	c, err := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Errorf("Get attester vote count failed: %v", err)
	}
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect attester vote count, want: 0, got: %v", c)
	}

	// Attester votes on the header that was submitted.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err != nil {
		t.Fatalf("Attester submits vote failed: %v", err)
	}
	c, err = s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if err != nil {
		t.Error(err)
	}
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect attester vote count, want: 1, got: %v", c)
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

// TestSubmitVote_Twice tests attester trying to submit same vote twice.
func TestSubmitVote_Twice(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Attester 0 registers.
	if err := s.registerAttesters(attesterDeposit, 0, 1); err != nil {
		t.Error(err)
	}
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	if err := s.addHeader(&s.testAccounts[0], shard0, period1, 'A'); err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Attester 0 votes on header.
	if err := s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A'); err != nil {
		t.Errorf("Attester submits vote failed: %v", err)
	}

	// Attester 0 votes on header again, it should fail.
	if err := s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A'); err == nil {
		t.Errorf("attester voting twice should have failed")
	}

	// Check attester's vote count is correct in shard.
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Incorrect attester vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVote_IneligibleAttester tests a ineligible attester tries to submit vote.
func TestSubmitVote_IneligibleAttester(t *testing.T) {
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

	// Unregistered Attester 0 votes on header, it should fail.
	err = s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A')
	if err == nil {
		t.Errorf("Non registered attester submits vote should have failed")
	}

	// Check attester's vote count is correct in shard.
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect attester vote count, want: 0, got: %v", c)
	}
}

// TestSubmitVote_NoHeader tests a attester tries to submit vote before header gets added.
func TestSubmitVote_NoHeader(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Attester 0 registers.
	if err := s.registerAttesters(attesterDeposit, 0, 1); err != nil {
		t.Error(err)
	}
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)

	// Attester 0 votes on header, it should fail because no header has added.
	if err := s.submitVote(&s.testAccounts[0], shard0, period1, index0, 'A'); err == nil {
		t.Errorf("Attester votes should have failed due to missing header")
	}

	// Check attester's vote count is correct in shard
	c, _ := s.smc.GetVoteCount(&bind.CallOpts{}, shard0)
	if c.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Incorrect attester vote count, want: 1, got: %v", c)
	}
}

// TestSubmitVote_InvalidArgs tests attester submits vote using wrong chunkroot and period.
func TestSubmitVote_InvalidArgs(t *testing.T) {
	s, _ := newSMCTestHelper(1)
	// Attester 0 registers.
	if err := s.registerAttesters(attesterDeposit, 0, 1); err != nil {
		t.Error(err)
	}
	s.fastForward(1)

	// Proposer adds header consists shard 0, period 1 and chunkroot 0xA.
	period1 := big.NewInt(1)
	shard0 := big.NewInt(0)
	index0 := big.NewInt(0)
	s.testAccounts[0].txOpts.Value = big.NewInt(0)
	if err := s.addHeader(&s.testAccounts[0], shard0, period1, 'A'); err != nil {
		t.Errorf("Proposer adds header failed: %v", err)
	}

	// Attester voting with incorrect period.
	period2 := big.NewInt(2)
	if err := s.submitVote(&s.testAccounts[0], shard0, period2, index0, 'A'); err == nil {
		t.Errorf("Attester votes should have failed due to incorrect period")
	}

	// Attester voting with incorrect chunk root.
	if err := s.submitVote(&s.testAccounts[0], shard0, period2, index0, 'B'); err == nil {
		t.Errorf("Attester votes should have failed due to incorrect chunk root")
	}
}
