package spectest

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestationMinimal(t *testing.T) {
	file, err := ioutil.ReadFile("attestation_minimal_formatted.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AttestationMinimalTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			pre := &pb.BeaconState{}
			err := convertToPb(tt.Pre, pre)
			if err != nil {
				t.Fatal(err)
			}

			_ = pre
			fmt.Printf("%v", pre)
			t.Fail() // TODO
		})
	}
}
