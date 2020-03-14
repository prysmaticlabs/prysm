// +build libfuzzer

package blocks

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

type InputBlockHeader struct {
	Pre   *pb.BeaconState          `json:"pre"`
	Block *ethpb.SignedBeaconBlock `json:"state"`
}

func Fuzz(b []byte) []byte {
	input := &InputBlockHeader{}
	if err := ssz.Unmarshal(b, input); err != nil {
		return fail()
	}
	st, err := stateTrie.InitializeFromProto(input.Pre)
	if err != nil {
		return fail()
	}
	post, err := ProcessBlockHeader(st, input.Block)
	if err != nil {
		return fail()
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		return fail()
	}
	return result
}

func fail() []byte {
	// TODO: Enable panic if desired.
	return nil
}
