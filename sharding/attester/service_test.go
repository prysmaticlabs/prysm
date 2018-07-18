package attester

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
<<<<<<< HEAD:sharding/attester/service_test.go
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prysmaticlabs/geth-sharding/sharding/internal"
	shardparams "github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
=======
	"github.com/prysmaticlabs/geth-sharding/sharding/internal"
	shardparams "github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
)

// Verifies that Attester implements the Actor interface.
var _ = types.Actor(&Attester{})

func TestHasAccountBeenDeregistered(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}

	client.SetDepositFlag(true)
<<<<<<< HEAD:sharding/attester/service_test.go
	err := joinAttesterPool(client, client, nil)
	if err != nil {
		t.Error(err)
	}

	err = leaveAttesterPool(client, client)

	if err != nil {
		t.Error(err)
	}

	dreg, err := hasAccountBeenDeregistered(client, client.Account())

	if err != nil {
		t.Error(err)
	}

	if !dreg {
		t.Error("account unable to be deregistered from attester pool")
	}
}

func TestIsLockupOver(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	client.SetDepositFlag(true)
	err := joinAttesterPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	err = leaveAttesterPool(client, client)

	if err != nil {
		t.Error(err)
	}

	client.FastForward(int(shardparams.DefaultConfig.AttesterLockupLength + 100))

	err = releaseAttester(client, client, client)
	if err != nil {
		t.Error(err)
	}

	lockup, err := isLockUpOver(client, client, client.Account())

	if err != nil {
		t.Error(err)
	}

	if !lockup {
		t.Error("lockup period is not over despite account being relased from registry")
	}
=======
	err := joinNotaryPool(client, client)
	if err != nil {
		t.Error(err)
	}
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
}

func TestIsAccountInAttesterPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	// address should not be in pool initially.
	b, err := isAccountInAttesterPool(client, client.Account())
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("account unexpectedly in attester pool")
	}

	txOpts, _ := client.CreateTXOpts(shardparams.DefaultConfig.AttesterDeposit)
	if _, err := smc.RegisterAttester(txOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	client.CommitWithBlock()
	b, err = isAccountInAttesterPool(client, client.Account())
	if err != nil {
		t.Error(err)
	}
	if !b {
		t.Error("account not in attester pool when expected to be")
	}
}

func TestJoinAttesterPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	// There should be no attester initially.
	numAttesters, err := smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numAttesters) != 0 {
		t.Errorf("unexpected number of attesters. Got %d, wanted 0.", numAttesters)
	}

	client.SetDepositFlag(false)
<<<<<<< HEAD:sharding/attester/service_test.go
	err = joinAttesterPool(client, client, shardparams.DefaultConfig)
=======
	err = joinNotaryPool(client, client)
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
	if err == nil {
		t.Error("joined attester pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
<<<<<<< HEAD:sharding/attester/service_test.go
	err = joinAttesterPool(client, client, shardparams.DefaultConfig)
=======
	err = joinNotaryPool(client, client)
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
	if err != nil {
		t.Error(err)
	}

	// Now there should be one attester.
	numAttesters, err = smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numAttesters) != 0 {
		t.Errorf("unexpected number of attesters. Got %d, wanted 1", numAttesters)
	}

	// Join while deposited should do nothing.
<<<<<<< HEAD:sharding/attester/service_test.go
	err = joinAttesterPool(client, client, shardparams.DefaultConfig)
=======
	err = joinNotaryPool(client, client)
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
	if err != nil {
		t.Error(err)
	}

	numAttesters, err = smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numAttesters) != 0 {
		t.Errorf("unexpected number of attesters. Got %d, wanted 1", numAttesters)
	}
}
<<<<<<< HEAD:sharding/attester/service_test.go

func TestLeaveAttesterPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	client.SetDepositFlag(true)

	// Test leaving attester pool before joining it.
	err := leaveAttesterPool(client, client)
	if err == nil {
		t.Error("able to leave attester pool despite having not joined it")
	}

	// Roundtrip test, join and leave pool.
	err = joinAttesterPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}
	client.CommitWithBlock()

	// Now there should be one attester.
	numAttesters, err := smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numAttesters) != 0 {
		t.Errorf("unexpected number of attesters. Got %d, wanted 1", numAttesters)
	}

	err = leaveAttesterPool(client, client)
	if err != nil {
		t.Error(err)
	}
	client.CommitWithBlock()

	numAttesters, err = smc.AttesterPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numAttesters) != 0 {
		t.Errorf("unexpected number of attesters. Got %d, wanted 0", numAttesters)
	}
}

func TestReleaseAttester(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	client.SetDepositFlag(true)

	// Test release attester before joining it.
	err := releaseAttester(client, client, client)
	if err == nil {
		t.Error("released From attester despite never joining pool")
	}

	// Roundtrip test, join and leave pool.
	err = joinAttesterPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	err = leaveAttesterPool(client, client)
	if err != nil {
		t.Error(err)
	}

	balance, err := backend.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		t.Error("unable to retrieve balance")
	}
	client.FastForward(int(shardparams.DefaultConfig.AttesterLockupLength + 10))

	err = releaseAttester(client, client, client)
	if err != nil {
		t.Fatal(err)
	}

	nreg, err := smc.AttesterRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatal(err)
	}
	if nreg.Deposited {
		t.Error("Unable to release Attester and deposit money back")
	}

	newbalance, err := client.Backend.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		t.Error("unable to retrieve balance")
	}

	if balance.Cmp(newbalance) != -1 {
		t.Errorf("Deposit was not returned, balance is currently: %v", newbalance)
	}
}

func TestSubmitVote(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	client.SetDepositFlag(true)

	err := joinAttesterPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}
}
=======
>>>>>>> f2f8850cccf5ff3498aebbce71baa05267bc07cc:sharding/notary/service_test.go
