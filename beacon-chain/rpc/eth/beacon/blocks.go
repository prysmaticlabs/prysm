package beacon

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errNilBlock = errors.New("nil block")
)

// GetWeakSubjectivity computes the starting epoch of the current weak subjectivity period, and then also
// determines the best block root and state root to use for a Checkpoint Sync starting from that point.
// DEPRECATED: GetWeakSubjectivity endpoint will no longer be supported
func (bs *Server) GetWeakSubjectivity(ctx context.Context, _ *empty.Empty) (*ethpbv1.WeakSubjectivityResponse, error) {
	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.GenesisTimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// This is already a grpc error, so we can't wrap it any further
		return nil, err
	}

	hs, err := bs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not get head state")
	}
	wsEpoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, hs, params.BeaconConfig())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get weak subjectivity epoch: %v", err)
	}
	wsSlot, err := slots.EpochStart(wsEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get weak subjectivity slot: %v", err)
	}
	cbr, err := bs.CanonicalHistory.BlockRootForSlot(ctx, wsSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("could not find highest block below slot %d", wsSlot))
	}
	cb, err := bs.BeaconDB.Block(ctx, cbr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("block with root %#x from slot index %d not found in db", cbr, wsSlot))
	}
	stateRoot := cb.Block().StateRoot()
	log.Printf("weak subjectivity checkpoint reported as epoch=%d, block root=%#x, state root=%#x", wsEpoch, cbr, stateRoot)
	return &ethpbv1.WeakSubjectivityResponse{
		Data: &ethpbv1.WeakSubjectivityData{
			WsCheckpoint: &ethpbv1.Checkpoint{
				Epoch: wsEpoch,
				Root:  cbr[:],
			},
			StateRoot: stateRoot[:],
		},
	}, nil
}
