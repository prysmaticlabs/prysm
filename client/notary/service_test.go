package notary

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/prysmaticlabs/prysm/client/internal"
	shardparams "github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/types"
)

// Verifies that Notary implements the Actor interface.
var _ = types.Actor(&Notary{})

func TestHasAccountBeenDeregistered(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}

	client.SetDepositFlag(true)
	err := joinNotaryPool(client, client)
	if err != nil {
		t.Error(err)
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
	err = joinNotaryPool(client, client)
	if err == nil {
		t.Error("joined notary pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
	err = joinNotaryPool(client, client)
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
	err = joinNotaryPool(client, client)
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
