package state

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ob "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"gopkg.in/d4l3k/messagediff.v1"
)

func loadSszOrDie(t *testing.T, filepath string, dst interface{}) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	if err := ssz.Unmarshal(b, dst); err != nil {
		t.Fatal(err)
	}
}

// This scenario produced a consensus issue when running Prysm with Trinity.
// The test expects that running a state transition with 0_block 0_prestate would
// output 0_poststate.
//
// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
//
// More context/data: https://github.com/djrtwo/interop-test-cases/tree/master/tests/prysm_16_duplicate_attestation_rewards
func TestConsensusIssueDuplicateRewards(t *testing.T) {
	pre := &ob.BeaconState{}
	block := &eth.BeaconBlock{}
	post := &ob.BeaconState{}

	params.UseMinimalConfig()

	loadSszOrDie(t, "testdata/minimal/duplicate_rewards/block.ssz", block)
	loadSszOrDie(t, "testdata/minimal/duplicate_rewards/pre.ssz", pre)
	loadSszOrDie(t, "testdata/minimal/duplicate_rewards/post.ssz", post)

	result, err := ExecuteStateTransition(context.Background(), pre, block)
	if err != nil {
		t.Fatalf("Could not process state transition %v", err)
	}
	if !proto.Equal(result, post) {
		diff, _ := messagediff.PrettyDiff(result, post)
		t.Log(diff)
		t.Fail()
	}
}

// This scenario produced a consensus issue between Trinity, ZCLI, and Lighthouse.
// The test expects that running a state transition with 0_block 0_prestate would
// output 0_poststate.
//
// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
//
// More context/data: https://github.com/djrtwo/interop-test-cases/tree/master/tests/night_one_16_crosslinks
func TestConsensusIssueCrosslinkMismatch(t *testing.T) {
	pre := &ob.BeaconState{}
	block := &eth.BeaconBlock{}
	post := &ob.BeaconState{}

	params.UseMinimalConfig()

	loadSszOrDie(t, "testdata/minimal/crosslink_mismatch/block.ssz", block)
	loadSszOrDie(t, "testdata/minimal/crosslink_mismatch/pre.ssz", pre)
	loadSszOrDie(t, "testdata/minimal/crosslink_mismatch/post.ssz", post)

	result, err := ExecuteStateTransition(context.Background(), pre, block)
	if err != nil {
		if !ssz.DeepEqual(result, post) {
			diff, _ := messagediff.PrettyDiff(result, post)
			t.Log(diff)
		}
		t.Fatalf("Could not process state transition %v", err)
	}
	if !proto.Equal(result, post) {
		diff, _ := messagediff.PrettyDiff(result, post)
		t.Log(diff)
		t.Fail()
	}
}

// This scenario produced a consensus issue between ZCLI, and Artemis.
// The test expects that running a state transition with 0_block 0_prestate would
// output 0_poststate.
//
// Assert ExecuteStateTransition(ctx, 0_block, 0_prestate) == 0_poststate
//
// More context/data: https://github.com/djrtwo/interop-test-cases/tree/master/tests/artemis_16_crosslinks_and_balances
func TestConsensusIssueArtemisCrosslink(t *testing.T) {
	pre := &ob.BeaconState{}
	block := &eth.BeaconBlock{}
	post := &ob.BeaconState{}

	params.UseMinimalConfig()

	loadSszOrDie(t, "testdata/minimal/artemis_crosslink/block.ssz", block)
	loadSszOrDie(t, "testdata/minimal/artemis_crosslink/pre.ssz", pre)
	loadSszOrDie(t, "testdata/minimal/artemis_crosslink/post.ssz", post)

	result, err := ExecuteStateTransition(context.Background(), pre, block)
	if err != nil {
		diff, _ := messagediff.PrettyDiff(result, post)
		t.Log(diff)
		t.Fatalf("Could not process state transition %v", err)
	}
	if !proto.Equal(result, post) {
		diff, _ := messagediff.PrettyDiff(result, post)
		t.Log(diff)
		t.Fail()
	}
}
