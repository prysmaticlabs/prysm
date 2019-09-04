package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestGenerateGenesisState(t *testing.T) {
	privKeys, pubKeys := deterministicallyGenerateKeys(10)
	depositDataItems, depositDataRoots, err := depositDataFromKeys(privKeys, pubKeys)
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
	deposits, err := generateDepositsFromData(depositDataItems, trie)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	genesisState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    mockEth1BlockHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := 10
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
