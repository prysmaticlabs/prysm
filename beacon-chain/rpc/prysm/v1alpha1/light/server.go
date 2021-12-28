// Package light defines a gRPC beacon service implementation, providing
// useful endpoints for light clients.
package light

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum beacon chain.
type Server struct {
	BeaconDB    db.ReadOnlyDatabase
	TimeFetcher blockchain.TimeFetcher
	Ctx         context.Context
}

func (s Server) GetBestUpdates(ctx context.Context, request *ethpb.GetBestUpdatesRequest) (*ethpb.LightClientUpdates, error) {
	endPeriod := request.GetToPeriod()
	startPeriod := request.GetFromPeriod()
	if startPeriod > endPeriod {
		return nil, status.Error(codes.InvalidArgument, "start period must be less than or equal to end period")
	}
	p := uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	currentPeriod := uint64(slots.ToEpoch(s.TimeFetcher.CurrentSlot())) / p
	if endPeriod > currentPeriod {
		return nil, status.Error(codes.InvalidArgument, "end period must be less than or equal to current period")
	}

	if startPeriod == endPeriod && endPeriod == currentPeriod {
		startEpoch := types.Epoch(currentPeriod * p)
		endEpoch := types.Epoch(currentPeriod*p + (p - 1))
		updates, err := s.BeaconDB.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(startEpoch).SetEndEpoch(endEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get light client updates: %v", err)
		}
		return &ethpb.LightClientUpdates{
			Updates: []*ethpb.LightClientUpdate{filterUpdates(updates)},
		}, nil
	}

	if startPeriod == endPeriod {
		startEpoch := types.Epoch(startPeriod * p)
		endEpoch := types.Epoch(startPeriod*p + (p - 1))
		updates, err := s.BeaconDB.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(startEpoch).SetEndEpoch(endEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get light client updates: %v", err)
		}
		return &ethpb.LightClientUpdates{
			Updates: updates,
		}, nil
	}

	if endPeriod >= currentPeriod {
		endPeriod = currentPeriod - 1
		startEpoch := types.Epoch(startPeriod * p)
		endEpoch := types.Epoch(endPeriod*p + (p - 1))
		updates, err := s.BeaconDB.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(startEpoch).SetEndEpoch(endEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get light client updates: %v", err)
		}
		startEpoch = types.Epoch(currentPeriod * p)
		endEpoch = types.Epoch(currentPeriod*p + (p - 1))
		moreUpdates, err := s.BeaconDB.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(startEpoch).SetEndEpoch(endEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get light client updates: %v", err)
		}
		updates = append(updates, filterUpdates(moreUpdates))
		return &ethpb.LightClientUpdates{
			Updates: updates,
		}, nil
	}

	startEpoch := types.Epoch(startPeriod * p)
	endEpoch := types.Epoch(endPeriod*p + (p - 1))
	updates, err := s.BeaconDB.LightClientUpdates(ctx, filters.NewFilter().SetStartEpoch(startEpoch).SetEndEpoch(endEpoch))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get light client updates: %v", err)
	}
	return &ethpb.LightClientUpdates{
		Updates: updates,
	}, nil
}

func (s Server) GetLatestUpdateFinalized(ctx context.Context, empty *empty.Empty) (*ethpb.LightClientUpdate, error) {
	update, err := s.BeaconDB.LatestFinalizedLightClientUpdate(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return update, nil
}

func (s Server) GetLatestUpdateUnFinalized(ctx context.Context, empty *empty.Empty) (*ethpb.LightClientUpdate, error) {
	update, err := s.BeaconDB.LatestLightClientUpdate(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return update, nil
}

func filterUpdates(updates []*ethpb.LightClientUpdate) *ethpb.LightClientUpdate {
	lastIndex := len(updates) - 1
	bestUpdate := updates[lastIndex]
	for i := lastIndex - 1; i >= 0; i-- {
		if updates[i].SyncAggregate.SyncCommitteeBits.Count() > bestUpdate.SyncAggregate.SyncCommitteeBits.Count() {
			bestUpdate = updates[i]
		}
		if updates[i].SyncAggregate.SyncCommitteeBits.Count() == params.BeaconConfig().SyncCommitteeSize {
			break
		}
	}
	return bestUpdate
}
