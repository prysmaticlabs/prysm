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
// More context/data: https://gist.github.com/lithp/382d4051fc91ef494c6bf772f0fd21c5
func TestConsensusIssue0(t *testing.T) {
	pre := &ob.BeaconState{}
	block := &eth.BeaconBlock{}
	post := &ob.BeaconState{}

	params.UseMinimalConfig()

	loadSszOrDie(t, "testdata/minimal/0_block.ssz", block)
	loadSszOrDie(t, "testdata/minimal/0_prestate.ssz", pre)
	loadSszOrDie(t, "testdata/minimal/0_poststate.ssz", post)

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
