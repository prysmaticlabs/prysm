package fuzz

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/protolambda/zrnt/eth2/phase0"
	"github.com/protolambda/zssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	prylabs_testing "github.com/prysmaticlabs/prysm/fuzz/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconFuzzBlock using the corpora from sigp/beacon-fuzz.
func BeaconFuzzBlock(b []byte) ([]byte, bool) {
	params.UseMainnetConfig()
	input := &InputBlockHeader{}
	if err := ssz.Unmarshal(b, input); err != nil {
		return fail(err)
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
	s := &pb.BeaconState{}
	if err := s.UnmarshalSSZ(sb); err != nil {
		return fail(err)
	}
	st, err := stateTrie.InitializeFromProto(s)
	if err != nil {
		return fail(err)
	}
	post, err := state.ProcessBlock(context.Background(), st, input.Block)
	if err != nil {
		return fail(err)
	}
	return success(post)
}

func beaconFuzzBlockZrnt(bb []byte, sb []byte) ([]byte, bool) {
	st := &phase0.BeaconState{}
	if err := zssz.Decode(bytes.NewReader(sb), uint64(len(sb)), st, phase0.BeaconStateSSZ); err != nil {
		return fail(err)
	}
	blk := &phase0.SignedBeaconBlock{}
	if err := zssz.Decode(bytes.NewReader(bb), uint64(len(bb)), blk, phase0.BeaconBlockSSZ); err != nil {
		return fail(err)
	}
	ffstate := phase0.NewFullFeaturedState(st)
	ffstate.LoadPrecomputedData()
	blockProc := new(phase0.BlockProcessFeature)
	blockProc.Meta = ffstate
	blockProc.Block = blk
	if err := ffstate.StateTransition(blockProc, true /*validate state root*/); err != nil {
		return fail(err)
	}
	var ret bytes.Buffer
	writer := bufio.NewWriter(&ret)
	if _, err := zssz.Encode(writer, ffstate.BeaconState, phase0.BeaconStateSSZ); err != nil {
		return fail(err)
	}
	if err := writer.Flush(); err != nil {
		return fail(err)
	}

	return ret.Bytes(), true
}
