package spectest

import (
	"io/ioutil"
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
	t.Skip("Tests will fail with mainnet config - awaiting mainnet tests from the researchers")
	filepath, err := bazel.Runfile("tests/genesis/initialization/genesis_initialization_minimal.yaml")
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
				t.Error("States are not equal")
			}
		})
	}
}

func TestGenesisValidityMinimal(t *testing.T) {
	filepath, err := bazel.Runfile("tests/genesis/validity/genesis_validity_minimal.yaml")
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
			helpers.ClearAllCaches()
			genesisState := tt.Genesis
			validatorCount, err := helpers.ActiveValidatorCount(genesisState, 0)
			if err != nil {
				t.Fatalf("Could not get active validator count: %v", err)
			}
			isValid := state.IsValidGenesisState(validatorCount, genesisState.GenesisTime)
			if isValid != tt.IsValid {
				t.Fatalf(
					"Genesis state does not have expected validity. Expected to be valid: %d, %d. %t %t",
					tt.Genesis.GenesisTime,
					validatorCount,
					isValid,
					tt.IsValid,
				)
			}
		})
	}
}
