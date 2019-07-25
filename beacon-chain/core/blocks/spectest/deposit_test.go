package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

const depositPrefix = "tests/operations/deposit/"

func runDepositTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	test := &BlockOperationTest{}
	if err := testutil.UnmarshalYaml(file, test); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(test.Config); err != nil {
		t.Fatal(err)
	}

	if len(test.TestCases) == 0 {
		t.Fatal("No tests!")
	}

	for _, tt := range test.TestCases {
		helpers.ClearAllCaches()
		t.Run(tt.Description, func(t *testing.T) {
			valMap := stateutils.ValidatorIndexMap(tt.Pre)
			post, err := blocks.ProcessDeposit(tt.Pre, tt.Deposit, valMap)
			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if tt.Post == nil {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(post, tt.Post) {
				t.Error("Post state does not match expected")
			}
		})
	}
}
