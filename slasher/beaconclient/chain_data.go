package beaconclient

import (
	"context"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var syncStatusPollingInterval = time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second

// ChainHead requests the latest beacon chain head
// from a beacon node via gRPC.
func (bs *Service) ChainHead(
	ctx context.Context,
) (*ethpb.ChainHead, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.ChainHead")
	defer span.End()
	res, err := bs.beaconClient.GetChainHead(ctx, &ptypes.Empty{})
	if err != nil || res == nil {
		return nil, errors.Wrap(err, "Could not retrieve chain head or got nil chain head")
	}
	return res, nil
}

// GenesisValidatorsRoot requests the beacon chain genesis validators
// root via gRPC.
func (bs *Service) GenesisValidatorsRoot(
	ctx context.Context,
) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.GenesisValidatorsRoot")
	defer span.End()
	res, err := bs.nodeClient.GetGenesis(ctx, &ptypes.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve genesis data")
	}
	if res == nil {
		return nil, errors.Wrap(err, " genesis data")
	}
	return res.GenesisValidatorsRoot, nil
}

// Poll the beacon node every syncStatusPollingInterval until the node
// is no longer syncing.
func (bs *Service) querySyncStatus(ctx context.Context) {
	status, err := bs.nodeClient.GetSyncStatus(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Could not fetch sync status")
	}
	if status != nil && !status.Syncing {
		log.Info("Beacon node is fully synced, starting slashing detection")
		return
	}
	ticker := time.NewTicker(syncStatusPollingInterval)
	log.Info("Waiting for beacon node to be fully synced...")
	for {
		select {
		case <-ticker.C:
			status, err := bs.nodeClient.GetSyncStatus(ctx, &ptypes.Empty{})
			if err != nil {
				log.WithError(err).Error("Could not fetch sync status")
			}
			if !status.Syncing {
				log.Info("Beacon node is fully synced, starting slashing detection")
				return
			}
			log.Info("Waiting for beacon node to be fully synced...")
		case <-ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}
