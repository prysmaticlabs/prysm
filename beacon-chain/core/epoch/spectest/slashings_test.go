package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runSlashingsTests(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &EpochProcessingTest{}
	if err := testutil.UnmarshalYaml(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	if len(s.TestCases) == 0 {
		t.Fatal("No tests!")
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()
			postState, err := epoch.ProcessSlashings(tt.Pre)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(postState, tt.Post) {
				t.Error("Did not get expected state")
			}
		})
	}
}

const slashingsPrefix = "tests/epoch_processing/slashings/"

func TestSlashingsMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(slashingsPrefix + "slashings_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runSlashingsTests(t, filepath)
}

func TestSlashingsMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(slashingsPrefix + "slashings_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runSlashingsTests(t, filepath)
}
