package spectest

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/trieutil"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runGenesisInitializationTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &GenesisInitializationTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()

			deposits := tt.Deposits
			leaves := [][]byte{}
			fmt.Println(deposits)
			for _, deposit := range deposits {
				hash, err := hashutil.DepositHash(deposit.Data)
				if err != nil {
					t.Fatal(err)
				}
				leaves = append(leaves, hash[:])
			}
			depositTrie, err := trieutil.GenerateTrieFromItems(leaves, int(params.BeaconConfig().DepositContractTreeDepth))
			if err != nil {
				t.Fatal(err)
			}
			depositRoot := depositTrie.Root()
			eth1Data := &pb.Eth1Data{
				DepositRoot:  depositRoot[:],
				DepositCount: uint64(len(deposits)),
				BlockHash:    tt.Eth1BlockHash,
			}

			genesisState, err := state.GenesisBeaconState(deposits, tt.Eth1Timestamp, eth1Data)
			if err != nil {
				t.Fatal(err)
			}

			expectedGenesisState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.State, expectedGenesisState); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(genesisState, expectedGenesisState) {
				t.Error("Did not get expected genesis state")
				diff, _ := messagediff.PrettyDiff(expectedGenesisState, genesisState)
				t.Log(diff)
			}
		})
	}
}

const genesisInitializationPrefix = "eth2_spec_tests/tests/genesis/initialization/"

func TestGenesisInitializationMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(genesisInitializationPrefix + "genesis_initialization_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runGenesisInitializationTest(t, filepath)
}
