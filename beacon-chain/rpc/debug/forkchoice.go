package debug

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

// GetProtoArrayForkChoice returns proto array fork choice store.
func (ds *Server) GetProtoArrayForkChoice(_ context.Context, _ *ptypes.Empty) (*pbrpc.ProtoArrayForkChoiceResponse, error) {
	store := ds.HeadFetcher.ProtoArrayStore()

	nodes := store.Nodes()
	returnedNodes := make([]*pbrpc.ProtoArrayNode, len(nodes))

	for i := 0; i < len(returnedNodes); i++ {
		r := nodes[i].Root()
		returnedNodes[i] = &pbrpc.ProtoArrayNode{
			Slot:           nodes[i].Slot(),
			Root:           r[:],
			Parent:         nodes[i].Parent(),
			JustifiedEpoch: nodes[i].JustifiedEpoch(),
			FinalizedEpoch: nodes[i].FinalizedEpoch(),
			Weight:         nodes[i].Weight(),
			BestChild:      nodes[i].BestChild(),
			BestDescendant: nodes[i].BestDescendant(),
		}
	}

	return &pbrpc.ProtoArrayForkChoiceResponse{
		PruneThreshold:  store.PruneThreshold(),
		JustifiedEpoch:  store.JustifiedEpoch(),
		FinalizedEpoch:  store.FinalizedEpoch(),
		ProtoArrayNodes: returnedNodes,
		Indices:         store.NodesIndices(),
	}, nil
}
