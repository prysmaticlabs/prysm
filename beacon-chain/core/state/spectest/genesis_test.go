package spectest

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	fmt.Println(params.BeaconConfig().HistoricalRootsLimit)

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()

			deposits := tt.Deposits
			dataLeaves := make([]*pb.DepositData, len(deposits))
			for i := range deposits {
				dataLeaves[i] = deposits[i].Data
			}
			depositRoot, err := ssz.HashTreeRootWithCapacity(dataLeaves, 1<<params.BeaconConfig().DepositContractTreeDepth)
			if err != nil {
				t.Fatal(err)
			}
			eth1Data := &pb.Eth1Data{
				DepositRoot:  depositRoot[:],
				DepositCount: uint64(len(deposits)),
				BlockHash:    tt.Eth1BlockHash,
			}

			expectedGenesisState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.State, expectedGenesisState); err != nil {
				t.Fatal(err)
			}

			genesisState, err := state.GenesisBeaconState(deposits, tt.Eth1Timestamp, eth1Data)
			if err != nil {
				t.Fatal(err)
			}

			rval := reflect.ValueOf(genesisState).Elem()
			otherVal := reflect.ValueOf(expectedGenesisState).Elem()
			for i := 0; i < rval.Type().NumField(); i++ {
				fmt.Println(rval.Type().Field(i).Name)
				if !reflect.DeepEqual(rval.Field(i).Interface(), otherVal.Field(i).Interface()) {
					fmt.Println("Did not get expected genesis state")
				}
			}

			if !reflect.DeepEqual(genesisState, expectedGenesisState) {
				t.Error("Did not get expected genesis state")
				// diff, _ := messagediff.PrettyDiff(expectedGenesisState, genesisState)
				// t.Log(diff)
				// _ = diff
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
