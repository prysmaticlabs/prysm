package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runDepositTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	test := &BlockOperationTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(test.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range test.TestCases {
		helpers.ClearAllCaches()
		t.Run(tt.Description, func(t *testing.T) {
			if tt.Description == "invalid_sig_new_deposit" {
				// TOOD(2857): uncompressed signature format is not supported
				t.Skip()
			}
			preState := &pb.BeaconState{}
			if err = testutil.ConvertToPb(tt.Pre, preState); err != nil {
				t.Fatal(err)
			}

			deposit := &pb.Deposit{}
			if err = testutil.ConvertToPb(tt.Deposit, deposit); err != nil {
				t.Fatal(err)
			}

			expectedPost := &pb.BeaconState{}
			if err = testutil.ConvertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}

			valMap := stateutils.ValidatorIndexMap(preState)
			post, err := blocks.ProcessDeposit(preState, deposit, valMap, true, true)
			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if len(expectedPost.ValidatorRegistry) == 0 {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(post, expectedPost) {
				t.Error("Post state does not match expected")
			}
		})
	}
}

var depositPrefix = "eth2_spec_tests/tests/operations/deposit/"

func TestDepositMinimalYaml(t *testing.T) {
	filepath, err := bazel.Runfile(depositPrefix + "deposit_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runDepositTest(t, filepath)
}

func TestDepositMainnetYaml(t *testing.T) {
	filepath, err := bazel.Runfile(depositPrefix + "deposit_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runDepositTest(t, filepath)
}
