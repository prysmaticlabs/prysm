package debug

import (
	"context"
	"encoding/hex"

	ptypes "github.com/gogo/protobuf/types"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

// GetProtoArrayForkChoice returns proto array fork choice store.
func (ds *Server) GetProtoArrayForkChoice(ctx context.Context, _ *ptypes.Empty) (*pbrpc.ProtoArrayForkChoiceResponse, error) {
	store := ds.HeadFetcher.ProtoArrayStore()

	nodes := store.Nodes
	returnedNodes := make([]*pbrpc.ProtoArrayNode, len(nodes))

	for i := 0; i < len(returnedNodes); i++ {
		returnedNodes[i] = &pbrpc.ProtoArrayNode{
			Slot:           nodes[i].Slot,
			Root:           nodes[i].Root[:],
			Parent:         nodes[i].Parent,
			JustifiedEpoch: nodes[i].JustifiedEpoch,
			FinalizedEpoch: nodes[i].FinalizedEpoch,
			Weight:         nodes[i].Weight,
			BestChild:      nodes[i].BestChild,
			BestDescendant: nodes[i].BestDescendant,
		}
	}

	indices := make(map[string]uint64, len(store.NodeIndices))
	for k, v := range store.NodeIndices {
		indices[hex.EncodeToString(k[:])] = v
	}

	return &pbrpc.ProtoArrayForkChoiceResponse{
		PruneThreshold:  store.PruneThreshold,
		JustifiedEpoch:  store.JustifiedEpoch,
		FinalizedEpoch:  store.FinalizedEpoch,
		ProtoArrayNodes: returnedNodes,
		Indices:         indices,
	}, nil
}
