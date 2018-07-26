package attester

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/prysmaticlabs/prysm/client/internal"
	shardparams "github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/types"
)

// Verifies that Attester implements the Actor interface.
var _ = types.Actor(&Attester{})

func TestHasAccountBeenDeregistered(t *testing.T) {
	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{SMC: smc, T: t, Backend: backend, BlockNumber: 1}

	client.SetDepositFlag(true)
	err := joinAttesterPool(client, client)
	if err != nil {
		t.Error(err)
	}
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
	err = joinAttesterPool(client, client)
	if err == nil {
		t.Error("joined attester pool while --deposit was not present")
	}

	client.SetDepositFlag(true)
	err = joinAttesterPool(client, client)
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
	err = joinAttesterPool(client, client)
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
