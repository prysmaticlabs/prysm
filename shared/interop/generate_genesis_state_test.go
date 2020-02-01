package interop_test

import (
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestGenerateGenesisState(t *testing.T) {
	numValidators := uint64(64)
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, numValidators)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}
	trie, err := trieutil.GenerateTrieFromItems(
		depositDataRoots,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		t.Fatal(err)
	}
	deposits, err := interop.GenerateDepositsFromData(depositDataItems, trie)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	genesisState, err := state.GenesisBeaconState(deposits, 0, &eth.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := int(numValidators)
	if genesisState.NumValidators() != want {
		t.Errorf("Wanted %d validators, received %d", want, genesisState.NumValidators())
	}
	if genesisState.NumValidators() != want {
		t.Errorf("Wanted %d validators, received %v", want, genesisState.NumValidators())
	}
	if genesisState.GenesisTime() != 0 {
		t.Errorf("Wanted genesis time 0, received %d", genesisState.GenesisTime())
	}
}
