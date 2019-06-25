package spectest

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSlotProcessingMainnet(t *testing.T) {
	filepath, err := bazel.Runfile("/eth2_spec_tests/tests/sanity/slots/sanity_slots_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Could not load file %v", err)
	}

	s := &SanitySlotsTest{}
	if err := yaml.Unmarshal(file, s); err != nil {
		t.Fatalf("Failed to Unmarshal: %v", err)
	}

	if err := spectest.SetConfig(s.Config); err != nil {
		t.Fatal(err)
	}

	for _, tt := range s.TestCases {
		t.Run(tt.Description, func(t *testing.T) {
			preState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Pre, preState); err != nil {
				t.Fatal(err)
			}

			var postState *pb.BeaconState

			postState, err = state.ProcessSlots(context.Background(), preState, preState.Slot+tt.Slots)
			if err != nil {
				t.Fatal(err)
			}

			expectedPostState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Post, expectedPostState); err != nil {
				t.Fatal(err)
			}

			expected, err := ssz.HashTreeRoot(expectedPostState)
			if err != nil {
				t.Fatal(err)
			}
			received, err := ssz.HashTreeRoot(postState)
			if err != nil {
				t.Fatal(err)
			}
			if expected != received {
				t.Fatal("post state does not match")
			}
		})
	}
}
