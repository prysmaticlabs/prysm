package debug

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// GetForkChoice returns a fork choice store.
func (ds *Server) GetForkChoice(_ context.Context, _ *empty.Empty) (*pbrpc.ForkChoiceResponse, error) {
	store := ds.HeadFetcher.ForkChoicer()

	return &pbrpc.ForkChoiceResponse{
		JustifiedEpoch:  store.JustifiedEpoch(),
		FinalizedEpoch:  store.FinalizedEpoch(),
		ForkchoiceNodes: store.ForkChoiceNodes(),
	}, nil
}
