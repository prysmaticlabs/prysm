package debug

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

// GetProtoArrayForkChoice returns proto array fork choice store.
func (ds *Server) GetProtoArrayForkChoice(ctx context.Context, _ *ptypes.Empty) (*pbrpc.ProtoArrayForkChoiceResponse, error) {
	store := ds.HeadFetcher.ProtoArrayStore()
	return &pbrpc.ProtoArrayForkChoiceResponse{
		PruneThreshold: store.PruneThreshold,
		JustifiedEpoch: store.JustifiedEpoch,
		FinalizedEpoch: store.FinalizedEpoch,
	}, nil
}
