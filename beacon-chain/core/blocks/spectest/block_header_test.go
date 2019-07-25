package spectest

import (
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

const blkHeaderPrefix = "tests/operations/block_header/"

// Block header test is actually a full block processing test. Not sure why it
// was named "block_header". The note in the test format readme says "Note that
// block_header is not strictly an operation (and is a full Block), but
// processed in the same manner, and hence included here."
func runBlockHeaderTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &BlockOperationTest{}
	if err := testutil.UnmarshalYaml(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if err := spectest.SetConfig(test.Config); err != nil {
		t.Fatal(err)
	}

	if len(test.TestCases) == 0 {
		t.Fatal("No tests!")
	}

	for _, tt := range test.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()

			post, err := blocks.ProcessBlockHeader(tt.Pre, tt.Block, true)

			if tt.Post == nil {
				// Note: This doesn't test anything worthwhile. It essentially tests
				// that *any* error has occurred, not any specific error.
				if err == nil {
					t.Fatal("did not fail when expected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if !proto.Equal(post, tt.Post) {
				diff, _ := messagediff.PrettyDiff(post, tt.Post)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
