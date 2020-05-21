package state

import (
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestTopazRewardsRegression_StateRootValid(t *testing.T) {
	regressionPath := "beacon-chain/core/state/regression_files"
	blockFile := "topaz_lighthouse_block.ssz"
	preStateFile := "topaz_lighthouse_pre_state.ssz"
	postStateFile := "topaz_lighthouse_post_state.ssz"

	// fetch data and unmarshal to provided data structure from file
	dataFetcher := func(fPath string, data interface{}) error {
		bPath, err := bazel.Runfile(path.Join(regressionPath, fPath))
		if err != nil {
			return err
		}
		rawFile, err := ioutil.ReadFile(bPath)
		if err != nil {
			return err
		}
		return ssz.Unmarshal(rawFile, data)
	}
	blk := &ethpb.SignedBeaconBlock{}
	if err := dataFetcher(blockFile, blk); err != nil {
		t.Fatal(err)
	}
	preState := &pb.BeaconState{}
	if err := dataFetcher(preStateFile, preState); err != nil {
		t.Fatal(err)
	}
	postState := &pb.BeaconState{}
	if err := dataFetcher(postStateFile, postState); err != nil {
		t.Fatal(err)
	}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(preState)
	if err != nil {
		t.Fatal(err)
	}
	expectedStateObj, err := stateTrie.InitializeFromProtoUnsafe(postState)
	if err != nil {
		t.Fatal(err)
	}
	// expect state transition to fail, as the block had an incorrect state root
	postStateObj, err := ExecuteStateTransition(context.Background(), stateObj, blk)
	if err == nil {
		t.Error("State transition did not fail")
	}
	expectedRt, err := expectedStateObj.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := postStateObj.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if expectedRt != rt {
		t.Errorf("Wanted a state root of %#x but got %#x", expectedRt, rt)
	}
	if !ssz.DeepEqual(expectedStateObj.InnerStateUnsafe(), postStateObj.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(expectedStateObj.InnerStateUnsafe(), postStateObj.InnerStateUnsafe())
		t.Errorf("Difference between states: %s", diff)
	}
}

// Regression test for a consensus split that was found in
// schlesi due to differences in how we handled slashed validators
// in epoch transitions from the pyspec. We use lighthouse's block
// as that was the block which triggered the split. The post state
// is from the pyspec which helps us validate that we are indeed
// performing the state transition correctly.
// Related issues:
// https://github.com/prysmaticlabs/prysm/issues/5901
// https://github.com/prysmaticlabs/prysm/issues/5897
func TestSchlesiRewardsRegression_PyspecState(t *testing.T) {
	t.Skip("Skipping until 5935 is resolved. New v0.12 config invalidates this test")

	regressionPath := "beacon-chain/core/state/regression_files"
	blockFile := "schlesi_lighthouse_block.ssz"
	preStateFile := "schlesi_lighthouse_pre_state.ssz"
	postStateFile := "schlesi_pyspec_post_state.ssz"

	// fetch data and unmarshal to provided data structure from file
	dataFetcher := func(fPath string, data interface{}) error {
		bPath, err := bazel.Runfile(path.Join(regressionPath, fPath))
		if err != nil {
			return err
		}
		rawFile, err := ioutil.ReadFile(bPath)
		if err != nil {
			return err
		}
		return ssz.Unmarshal(rawFile, data)
	}
	blk := &ethpb.SignedBeaconBlock{}
	if err := dataFetcher(blockFile, blk); err != nil {
		t.Fatal(err)
	}
	preState := &pb.BeaconState{}
	if err := dataFetcher(preStateFile, preState); err != nil {
		t.Fatal(err)
	}
	postState := &pb.BeaconState{}
	if err := dataFetcher(postStateFile, postState); err != nil {
		t.Fatal(err)
	}
	stateObj, err := stateTrie.InitializeFromProtoUnsafe(preState)
	if err != nil {
		t.Fatal(err)
	}
	expectedStateObj, err := stateTrie.InitializeFromProtoUnsafe(postState)
	if err != nil {
		t.Fatal(err)
	}
	// The state transition is expected to fail as lighthouse produced a block
	// with the wrong state root.
	postStateObj, err := ExecuteStateTransition(context.Background(), stateObj, blk)
	if err == nil {
		t.Error("Expected state root failure")
	}
	expectedRt, err := expectedStateObj.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := postStateObj.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if expectedRt != rt {
		t.Errorf("Wanted a state root of %#x but got %#x", expectedRt, rt)
	}
	if !ssz.DeepEqual(expectedStateObj.InnerStateUnsafe(), postStateObj.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(expectedStateObj.InnerStateUnsafe(), postStateObj.InnerStateUnsafe())
		t.Errorf("Difference between states: %s", diff)
	}
}
