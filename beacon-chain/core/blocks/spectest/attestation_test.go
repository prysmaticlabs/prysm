package spectest

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func runAttestationTest(t *testing.T, filename string) {
	file, err := ioutil.ReadFile(filename)
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
			pre := &pb.BeaconState{}
			err := convertToPb(tt.Pre, pre)
			if err != nil {
				t.Fatal(err)
			}

			att := &pb.Attestation{}
			if err := convertToPb(tt.Attestation, att); err != nil {
				t.Fatal(err)
			}

			block := &pb.BeaconBlock{
				Body: &pb.BeaconBlockBody{
					Attestations: []*pb.Attestation{
						att,
					},
				},
			}

			post, err := blocks.ProcessBlockAttestations(pre, block, true /*verify sig*/)

			if !reflect.ValueOf(tt.Post).IsValid() {
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

			expectedPost := &pb.BeaconState{}
			if err := convertToPb(tt.Post, expectedPost); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(post, expectedPost) {
				t.Fatal("Post state does not match expected")
			}
		})
	}
}

func TestAttestationMinimal(t *testing.T) {
	runAttestationTest(t,"attestation_minimal_formatted.yaml")
}

func TestAttestationMainnet(t *testing.T) {
	runAttestationTest(t,"attestation_mainnet_formatted.yaml")
}
