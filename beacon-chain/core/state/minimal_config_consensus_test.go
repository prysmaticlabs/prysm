package state

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestConsensusBugs(t *testing.T) {
	tests := []struct {
		name          string
		blockPath     string
		preStatePath  string
		postStatePath string
	}{
		{
			// This scenario produced a consensus issue between ZCLI, and Artemis.
			// The test expects that running a state transition with 0_block 0_prestate would
			// output 0_poststate.
			//
			// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
			//
			// https://github.com/djrtwo/interop-test-cases/tree/master/tests/artemis_16_crosslinks_and_balances
			name:          "ZcliArtemisCrosslinks",
			blockPath:     "testdata/minimal/artemis_crosslink/block.ssz",
			preStatePath:  "testdata/minimal/artemis_crosslink/pre.ssz",
			postStatePath: "testdata/minimal/artemis_crosslink/post.ssz",
		},
		{
			// This scenario produced a consensus issue when running Prysm with Trinity.
			// The test expects that running a state transition with 0_block 0_prestate would
			// output 0_poststate.
			//
			// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
			//
			// https://github.com/djrtwo/interop-test-cases/tree/master/tests/prysm_16_duplicate_attestation_rewards
			name:          "TrinityPrysmDuplicateRewards",
			blockPath:     "testdata/minimal/duplicate_rewards/block.ssz",
			preStatePath:  "testdata/minimal/duplicate_rewards/pre.ssz",
			postStatePath: "testdata/minimal/duplicate_rewards/post.ssz",
		},
		{
			// This scenario produced a consensus issue between Trinity, ZCLI, and Lighthouse.
			// The test expects that running a state transition with 0_block 0_prestate would
			// output 0_poststate.
			//
			// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
			//
			// https://github.com/djrtwo/interop-test-cases/tree/master/tests/night_one_16_crosslinks
			name:          "ZcliTrinityLighthouseCrosslinks",
			blockPath:     "testdata/minimal/crosslink_mismatch/block.ssz",
			preStatePath:  "testdata/minimal/crosslink_mismatch/pre.ssz",
			postStatePath: "testdata/minimal/crosslink_mismatch/post.ssz",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			block := &ethpb.BeaconBlock{}
			pre := &pb.BeaconState{}
			post := &pb.BeaconState{}

			params.UseMinimalConfig()

			loadSszOrDie(t, test.blockPath, block)
			loadSszOrDie(t, test.preStatePath, pre)
			loadSszOrDie(t, test.postStatePath, post)

			result, err := ExecuteStateTransition(context.Background(), pre, block)
			if err != nil {
				t.Logf("Could not process state transition %v", err)
			}
			if !ssz.DeepEqual(result, post) {
				diff, _ := messagediff.PrettyDiff(result, post)
				t.Log(diff)
				t.Fatal("Resulting state is not equal to expected post state")
			}
		})
	}
}

func loadSszOrDie(t *testing.T, filepath string, dst interface{}) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ssz.Unmarshal(b, dst); err != nil {
		t.Fatal(err)
	}
}
