package blocks_test

import (
	"context"
	"io/ioutil"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// https://github.com/sigp/beacon-fuzz/issues/78
func TestProcessAttestationNoVerifySignature_BeaconFuzzIssue78(t *testing.T) {
	attData, err := ioutil.ReadFile("testdata/beaconfuzz_78_attestation.ssz")
	if err != nil {
		t.Fatal(err)
	}
	att := &ethpb.Attestation{}
	if err := att.UnmarshalSSZ(attData); err != nil {
		t.Fatal(err)
	}
	stateData, err := ioutil.ReadFile("testdata/beaconfuzz_78_beacon.ssz")
	if err != nil {
		t.Fatal(err)
	}
	spb := &pb.BeaconState{}
	if err := spb.UnmarshalSSZ(stateData); err != nil {
		t.Fatal(err)
	}
	st, err := state.InitializeFromProtoUnsafe(spb)
	if err != nil {
		t.Fatal(err)
	}

	_, err = blocks.ProcessAttestationNoVerifySignature(context.Background(), st, att)
	t.Log(err)
	if err == nil {
		t.Error("Expected an error")
	}
}
