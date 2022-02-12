package debug

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func protoArrayNode(node *protoarray.Node) *pbrpc.ProtoArrayNode {
	var pbChildren []*pbrpc.ProtoArrayNode
	for _, child := range node.Children() {
		pbChildren = append(pbChildren, protoArrayNode(child))
	}
	root := node.Root()
	return &pbrpc.ProtoArrayNode{
		Slot:           node.Slot(),
		Root:           root[:],
		Children:       pbChildren,
		JustifiedEpoch: node.JustifiedEpoch(),
		FinalizedEpoch: node.FinalizedEpoch(),
		Balance:        node.Balance(),
		Optimistic:     node.Optimistic(),
	}
}

// GetProtoArrayForkChoice returns proto array fork choice store.
func (ds *Server) GetProtoArrayForkChoice(_ context.Context, _ *empty.Empty) (*pbrpc.ProtoArrayForkChoiceResponse, error) {
	store := ds.HeadFetcher.ProtoArrayStore()
	treeRoot := store.TreeRoot()
	pbTreeRoot := protoArrayNode(treeRoot)

	return &pbrpc.ProtoArrayForkChoiceResponse{
		PruneThreshold: store.PruneThreshold(),
		JustifiedEpoch: store.JustifiedEpoch(),
		FinalizedEpoch: store.FinalizedEpoch(),
		TreeRoot:       pbTreeRoot,
	}, nil
}
