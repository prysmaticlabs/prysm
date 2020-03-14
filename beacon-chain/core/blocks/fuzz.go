// +build libfuzzer

package blocks

import (
	"encoding/json"

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
	//ssz.Unmarshal(b, input)
	if err := json.Unmarshal(b, input); err != nil {
		panic(err)
	}
	st, err := stateTrie.InitializeFromProto(input.Pre)
	if err != nil {
		// panic(err)
		return nil
	}
	post, err := ProcessBlockHeader(st, input.Block)
	if err != nil {
		// panic(err)
		return nil
	}

	result, err := ssz.Marshal(post.InnerStateUnsafe())
	if err != nil {
		// panic(err)
		return nil
	}
	return result
}
