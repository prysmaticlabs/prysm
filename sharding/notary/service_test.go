package notary

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/internal"
	shardparams "github.com/ethereum/go-ethereum/sharding/params"
)

var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

// Verifies that Notary implements the Actor interface.
var _ = sharding.Actor(&Notary{})

func TestHasAccountBeenDeregistered(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}

	client.SetDepositFlag(true)
	err := joinNotaryPool(client, client, nil)
	if err != nil {
		t.Error(err)
	}

	err = leaveNotaryPool(client, client)

	if err != nil {
		t.Error(err)
	}

	dreg, err := hasAccountBeenDeregistered(client, client.Account())

	if err != nil {
		t.Error(err)
	}

	if !dreg {
		t.Error("account unable to be deregistered from notary pool")
	}
}

func TestIsLockupOver(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	client.SetDepositFlag(true)
	err := joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	err = leaveNotaryPool(client, client)

	if err != nil {
		t.Error(err)
	}

	client.FastForward(int(shardparams.DefaultConfig.NotaryLockupLength + 100))

	err = releaseNotary(client, client, client)
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
}

func TestIsAccountInNotaryPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	// address should not be in pool initially.
	b, err := isAccountInNotaryPool(client, client.Account())
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Fatal("account unexpectedly in notary pool")
	}

	txOpts, _ := client.CreateTXOpts(shardparams.DefaultConfig.NotaryDeposit)
	if _, err := smc.RegisterNotary(txOpts); err != nil {
		t.Fatalf("Failed to deposit: %v", err)
	}
	client.CommitWithBlock()
	b, err = isAccountInNotaryPool(client, client.Account())
	if err != nil {
		t.Error(err)
	}
	if !b {
		t.Error("account not in notary pool when expected to be")
	}
}

func TestJoinNotaryPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}

	// There should be no notary initially.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 0.", numNotaries)
	}

	client.SetDepositFlag(false)
	err = joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err == nil {
		t.Error("joined notary pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
	err = joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	// Now there should be one notary.
	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}

	// Join while deposited should do nothing.
	err = joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}
}

func TestLeaveNotaryPool(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	client.SetDepositFlag(true)

	// Test leaving notary pool before joining it.
	err := leaveNotaryPool(client, client)
	if err == nil {
		t.Error("able to leave notary pool despite having not joined it")
	}

	// Roundtrip test, join and leave pool.
	err = joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}
	client.CommitWithBlock()

	// Now there should be one notary.
	numNotaries, err := smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(1).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 1", numNotaries)
	}

	err = leaveNotaryPool(client, client)
	if err != nil {
		t.Error(err)
	}
	client.CommitWithBlock()

	numNotaries, err = smc.NotaryPoolLength(&bind.CallOpts{})
	if err != nil {
		t.Error(err)
	}
	if big.NewInt(0).Cmp(numNotaries) != 0 {
		t.Errorf("unexpected number of notaries. Got %d, wanted 0", numNotaries)
	}
}

func TestReleaseNotary(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend}
	client.SetDepositFlag(true)

	// Test release notary before joining it.
	err := releaseNotary(client, client, client)
	if err == nil {
		t.Error("released From notary despite never joining pool")
	}

	// Roundtrip test, join and leave pool.
	err = joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}

	err = leaveNotaryPool(client, client)
	if err != nil {
		t.Error(err)
	}

	balance, err := backend.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		t.Error("unable to retrieve balance")
	}
	client.FastForward(int(shardparams.DefaultConfig.NotaryLockupLength + 10))

	err = releaseNotary(client, client, client)
	if err != nil {
		t.Fatal(err)
	}

	nreg, err := smc.NotaryRegistry(&bind.CallOpts{}, addr)
	if err != nil {
		t.Fatal(err)
	}
	if nreg.Deposited {
		t.Error("Unable to release Notary and deposit money back")
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

	err := joinNotaryPool(client, client, shardparams.DefaultConfig)
	if err != nil {
		t.Error(err)
	}
}
