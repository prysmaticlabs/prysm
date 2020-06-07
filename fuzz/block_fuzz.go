package fuzz

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/runutil"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zssz"
	//ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/fuzz/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const timeout = 60*time.Second

// BeaconFuzzBlock using the corpora from sigp/beacon-fuzz.
func BeaconFuzzBlock(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputBlockHeader{}
	if err := input.UnmarshalSSZ(b); err != nil {
		return nil, false
	}
	sb, err := prylabs_testing.GetBeaconFuzzStateBytes(input.StateID)
	if err != nil || len(sb) == 0 {
		return fail(err)
	}
	prysmResult, prysmOK := beaconFuzzBlockPrysm(input, sb)

	bb, err := input.Block.MarshalSSZ()
	if err != nil {
		return fail(err)
	}
	zrntResult, zrntOK := beaconFuzzBlockZrnt(bb, sb)

	if prysmOK != zrntOK {
		panic(fmt.Sprintf("Prysm=%t, ZRNT=%t", prysmOK, zrntOK))
	}
	if !prysmOK {
		return nil, false
	}
	if !bytes.Equal(prysmResult, zrntResult) {
		panic("Prysm's result state does not match ZRNT's result state.")
	}
	return prysmResult, prysmOK
}

func beaconFuzzBlockPrysm(input *InputBlockHeader, sb []byte) ([]byte, bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runutil.RunAfter(ctx, timeout, func() {
		panic("Deadline exceeded")
	})

	s := &pb.BeaconState{}
	if err := s.UnmarshalSSZ(sb); err != nil {
		return nil, false
	}
	st, err := stateTrie.InitializeFromProto(s)
	if err != nil {
		return fail(err)
	}
	ctx, cancel2 := context.WithTimeout(ctx, timeout/2)
	defer cancel2()
	post, err := state.ExecuteStateTransition(ctx, st, input.Block)
	if err != nil {
		return fail(err)
	}
	return success(post)
}

func beaconFuzzBlockZrnt(bb []byte, sb []byte) ([]byte, bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runutil.RunAfter(ctx, timeout, func() {
		panic("Deadline exceeded")
	})

	st := &beacon.BeaconState{}
	if err := zssz.Decode(bytes.NewReader(sb), uint64(len(sb)), st, beacon.BeaconStateSSZ); err != nil {
		return fail(err)
	}
	blk := &beacon.SignedBeaconBlock{}
	if err := zssz.Decode(bytes.NewReader(bb), uint64(len(bb)), blk, beacon.SignedBeaconBlockSSZ); err != nil {
		return fail(err)
	}
	state, err := beacon.AsBeaconStateView(beacon.BeaconStateType.Deserialize(bytes.NewReader(bb), uint64(len(bb))))
	if err != nil {
		return fail(err)
	}
	if err := state.StateTransition(ctx, nil, blk, false /*TODO:something*/); err != nil {
		return fail(err)
	}
	var ret bytes.Buffer
	writer := bufio.NewWriter(&ret)
	if err := state.Serialize(writer); err != nil {
		return fail(err)
	}
	if err := writer.Flush(); err != nil {
		return fail(err)
	}

	return ret.Bytes(), true
}
