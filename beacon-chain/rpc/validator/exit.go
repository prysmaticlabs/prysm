package validator

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/exit"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestExit requests an exit for a validator.
func (vs *Server) RequestExit(ctx context.Context, req *ethpb.VoluntaryExit) (*ptypes.Empty, error) {
	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Confirm the validator is eligible to exit with the parameters provided
	err = exit.ValidateVoluntaryExit(s, vs.GenesisTimeFetcher.GenesisTime(), req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Send the voluntary exit to the state feed.
	vs.StateNotifier.StateFeed().Send(&statefeed.Event{
		Type: statefeed.VoluntaryExitReceived,
		Data: &statefeed.VoluntaryExitReceivedData{
			VoluntaryExit: req,
		},
	})

	return nil, nil
}
