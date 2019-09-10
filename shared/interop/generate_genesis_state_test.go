package interop

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestGenerateGenesisState(t *testing.T) {
	numValidators := uint64(64)
	privKeys, pubKeys, err := DeterministicallyGenerateKeys(0 /*startIndex*/, numValidators)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := DepositDataFromKeys(privKeys, pubKeys)
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
	deposits, err := GenerateDepositsFromData(depositDataItems, trie)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	genesisState, err := state.GenesisBeaconState(deposits, 0, &eth.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    mockEth1BlockHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := int(numValidators)
	if len(genesisState.Validators) != want {
		t.Errorf("Wanted %d validators, received %d", want, len(genesisState.Validators))
	}
	if len(genesisState.Validators) != want {
		t.Errorf("Wanted %d validators, received %v", want, len(genesisState.Validators))
	}
	if genesisState.GenesisTime != 0 {
		t.Errorf("Wanted genesis time 0, received %d", genesisState.GenesisTime)
	}
}
