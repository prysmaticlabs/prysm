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
	"gopkg.in/d4l3k/messagediff.v1"
)

func runAttestationTest(t *testing.T, filename string) {
	filepath, err := bazel.Runfile("tests/operations/attestation/" + filename)
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
			body := &pb.BeaconBlockBody{
				Attestations: []*pb.Attestation{
					tt.Attestation,
				},
			}

			post, err := blocks.ProcessAttestations(tt.Pre, body, true /*verify sig*/)
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
			if tt.Post == nil {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				t.Logf("Expected failure; failure reason = %v", err)
				return
			} else if err != nil {
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

func TestAttestationMinimal(t *testing.T) {
	runAttestationTest(t, "attestation_minimal.yaml")
}

func TestAttestationMainnet(t *testing.T) {
	runAttestationTest(t, "attestation_mainnet.yaml")
}
