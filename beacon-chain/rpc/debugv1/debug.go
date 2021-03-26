package debugv1

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"go.opencensus.io/trace"
)

// GetBeaconState returns the full beacon state for a given state id.
func (bs *Server) GetBeaconState(ctx context.Context, req *ethpb.StateRequest) (*ethpb.BeaconStateResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListForkChoiceHeads retrieves the fork choice leaves for the current head.
func (bs *Server) ListForkChoiceHeads(ctx context.Context, _ *ptypes.Empty) (*ethpb.ForkChoiceHeadsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debugv1.ListForkChoiceHeads")
	defer span.End()

	heads := bs.ForkChoiceStore.ViableHeads()
	resp := &ethpb.ForkChoiceHeadsResponse{
		Data: make([]*ethpb.ForkChoiceHead, len(heads)),
	}
	for i, h := range heads {
		root := h.Root()
		resp.Data[i] = &ethpb.ForkChoiceHead{
			Root: root[:],
			Slot: h.Slot(),
		}
	}

	return resp, nil
}
