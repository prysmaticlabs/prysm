package spectest

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

// This is a subset of state.ProcessEpoch. The spec test defines input data for
// `justification_and_finalization` only.
func processJustificationAndFinalizationWrapper(state *pb.BeaconState) (*pb.BeaconState, error) {
	helpers.ClearAllCaches()

	// This process mutates the state, so we'll make a copy in order to print debug before/after.
	state = proto.Clone(state).(*pb.BeaconState)

	prevEpochAtts, err := epoch.MatchAttestations(state, helpers.PrevEpoch(state))
	if err != nil {
		return nil, fmt.Errorf("could not get target atts prev epoch %d: %v",
			helpers.PrevEpoch(state), err)
	}
	currentEpochAtts, err := epoch.MatchAttestations(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, fmt.Errorf("could not get target atts current epoch %d: %v",
			helpers.CurrentEpoch(state), err)
	}
	prevEpochAttestedBalance, err := epoch.AttestingBalance(state, prevEpochAtts.Target)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting balance prev epoch: %v", err)
	}
	currentEpochAttestedBalance, err := epoch.AttestingBalance(state, currentEpochAtts.Target)
	if err != nil {
		return nil, fmt.Errorf("could not get attesting balance current epoch: %v", err)
	}

	state, err = epoch.ProcessJustificationAndFinalization(state, prevEpochAttestedBalance, currentEpochAttestedBalance)
	if err != nil {
		return nil, fmt.Errorf("could not process justification: %v", err)
	}

	return state, nil
}

func runJustificationAndFinalizationTests(t *testing.T, filename string) {
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
			preState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Pre, preState); err != nil {
				t.Fatal(err)
			}

			postState, err := processJustificationAndFinalizationWrapper(preState)
			if err != nil {
				t.Fatal(err)
			}

			expectedPostState := &pb.BeaconState{}
			if err := testutil.ConvertToPb(tt.Post, expectedPostState); err != nil {
				t.Fatal(err)
			}

			if postState.JustificationBits[0] != expectedPostState.JustificationBits[0] {
				t.Errorf("Justification bits mismatch. PreState.JustificationBits=%v. PostState.JustificationBits=%v. Expected=%v", preState.JustificationBits, postState.JustificationBits, expectedPostState.JustificationBits)
			}

			if !reflect.DeepEqual(postState, expectedPostState) {
				diff, _ := messagediff.PrettyDiff(postState, expectedPostState)
				t.Log(diff)
				t.Error("Did not get expected state")
			}
		})
	}
}

const justificationAndFinalizationPrefix = "tests/epoch_processing/justification_and_finalization/"

func TestJustificationAndFinalizationMinimal(t *testing.T) {
	// TODO(#2891): Verify with ETH2 spec test.
	t.Skip("The input data fails preconditions for matching attestations in the state for the current epoch.")
	filepath, err := bazel.Runfile(justificationAndFinalizationPrefix + "justification_and_finalization_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runJustificationAndFinalizationTests(t, filepath)
}

func TestJustificationAndFinalizationMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(justificationAndFinalizationPrefix + "justification_and_finalization_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runJustificationAndFinalizationTests(t, filepath)
}
