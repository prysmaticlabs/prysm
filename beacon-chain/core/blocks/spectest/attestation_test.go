package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runAttestationTest(t *testing.T, filename string) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/operations/attestation/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AttestationTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if err := spectest.SetConfig(test.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range test.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			helpers.ClearAllCaches()
			pre := &pb.BeaconState{}
			err := testutil.ConvertToPb(tt.Pre, pre)
			if err != nil {
				t.Fatal(err)
			}

			att := &pb.Attestation{}
			if err := testutil.ConvertToPb(tt.Attestation, att); err != nil {
				t.Fatal(err)
			}

			block := &pb.BeaconBlock{
				Body: &pb.BeaconBlockBody{
					Attestations: []*pb.Attestation{
						att,
					},
				},
			}

			// TODO: TURN ON VERIFY SIGNATURE! Getting a panic...
			post, err := blocks.ProcessBlockAttestations(pre, block,
				true /*verify sig*/)

			if !reflect.ValueOf(tt.Post).IsValid() {
				// Note: This doesn't test anything worthwhile. It essentially tests
				// that *any* error has occurred, not any specific error.
				if err == nil {
					t.Fatal("did not fail when expected")
				}
				return
			}

			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if len(tt.Post.ValidatorRegistry) == 0 {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}

			expectedPost := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(post, expectedPost) {
				t.Fatal("Post state does not match expected")
			}
		})
	}
}

func TestAttestationMinimal(t *testing.T) {
	runAttestationTest(t, "attestation_minimal.yaml")
}

func TestAttestationMainnet(t *testing.T) {
	runAttestationTest(t, "attestation_mainnet.yaml")
}
