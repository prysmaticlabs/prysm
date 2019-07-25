package spectest

import (
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

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
	if err := testutil.UnmarshalYaml(file, s); err != nil {
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
