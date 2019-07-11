package spectest

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"gopkg.in/d4l3k/messagediff.v1"
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
			eth1Data := &pb.Eth1Data{
				BlockHash: tt.Eth1BlockHash,
			}
			genesisState, err := state.GenesisBeaconState(tt.Deposits, tt.Eth1Timestamp, eth1Data)
			if err != nil {
				t.Fatal(err)
			}

			if !proto.Equal(genesisState, tt.State) {
				diff, _ := messagediff.PrettyDiff(genesisState, tt.State)
				t.Log(diff)
				t.Fatal("Genesis state does not match expected")
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
				t.Fatal("Genesis state does not have expected validity.")
			}
		})
	}
}
