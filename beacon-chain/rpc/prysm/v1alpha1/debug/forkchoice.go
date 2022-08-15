package debug

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pbrpc "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// GetForkChoice returns a fork choice store.
func (ds *Server) GetForkChoice(_ context.Context, _ *empty.Empty) (*pbrpc.ForkChoiceResponse, error) {
	store := ds.ForkFetcher.ForkChoicer()

	return &pbrpc.ForkChoiceResponse{
		JustifiedEpoch:  store.JustifiedCheckpoint().Epoch,
		FinalizedEpoch:  store.FinalizedCheckpoint().Epoch,
		ForkchoiceNodes: store.ForkChoiceNodes(),
	}, nil
}
