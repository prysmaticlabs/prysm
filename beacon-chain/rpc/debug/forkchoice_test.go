package debug

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
)

func TestServer_GetForkChoice(t *testing.T) {
	store := &protoarray.Store{
		PruneThreshold: 1,
		JustifiedEpoch: 2,
		FinalizedEpoch: 3,
		Nodes:          []*protoarray.Node{{Slot: 4, Root: [32]byte{'a'}}},
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{ForkChoiceStore: store},
	}

	res, err := bs.GetProtoArrayForkChoice(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.PruneThreshold != store.PruneThreshold {
		t.Error("Did not get wanted prune threshold")
	}
	if res.JustifiedEpoch != store.JustifiedEpoch {
		t.Error("Did not get wanted justified epoch")
	}
	if res.FinalizedEpoch != store.FinalizedEpoch {
		t.Error("Did not get wanted finalized epoch")
	}
	if res.ProtoArrayNodes[0].Slot != store.Nodes[0].Slot {
		t.Error("Did not get wanted node slot")
	}
}
