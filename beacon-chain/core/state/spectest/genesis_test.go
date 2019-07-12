package spectest

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestGenesisInitializationMinimal(t *testing.T) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/genesis/initialization/genesis_initialization_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(filepath)
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

			genesisState, err := state.GenesisBeaconState(deposits, tt.Eth1Timestamp, eth1Data)
			if err != nil {
				t.Fatal(err)
			}

			if !proto.Equal(genesisState, tt.State) {
				rval := reflect.ValueOf(genesisState).Elem()
				otherVal := reflect.ValueOf(tt.State).Elem()
				for i := 0; i < rval.Type().NumField(); i++ {
					fmt.Println(rval.Type().Field(i).Name)
					if !reflect.DeepEqual(rval.Field(i).Interface(), otherVal.Field(i).Interface()) {
						fmt.Printf("--Did not get expected %v\n--Received: %v\n", rval.Type().Field(i).Name, otherVal.Field(i).Interface())
					} else {
						fmt.Println("--Matched")
					}
				}
				t.Error("States are not equal")
			}

		})
	}
}

func TestGenesisValidityMinimal(t *testing.T) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/genesis/validity/genesis_validity_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &GensisValidityTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			genesisState := tt.Genesis
			validatorCount, err := helpers.ActiveValidatorCount(genesisState, 0)
			if err != nil {
				t.Fatalf("Could not get active validator count: %v", err)
			}
			isValid := state.IsValidGenesisState(validatorCount, genesisState.GenesisTime)
			if isValid != tt.IsValid {
				t.Fatal("Genesis state does not have expected validity")
			}
		})
	}
}
