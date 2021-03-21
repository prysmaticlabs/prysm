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
func (s *Service) ChainHead(
	ctx context.Context,
) (*ethpb.ChainHead, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.ChainHead")
	defer span.End()
	res, err := s.cfg.BeaconClient.GetChainHead(ctx, &ptypes.Empty{})
	if err != nil || res == nil {
		return nil, errors.Wrap(err, "Could not retrieve chain head or got nil chain head")
	}
	return res, nil
}

// GenesisValidatorsRoot requests or fetch from memory the beacon chain genesis
// validators root via gRPC.
func (s *Service) GenesisValidatorsRoot(
	ctx context.Context,
) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.GenesisValidatorsRoot")
	defer span.End()

	if s.genesisValidatorRoot == nil {
		res, err := s.cfg.NodeClient.GetGenesis(ctx, &ptypes.Empty{})
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve genesis data")
		}
		if res == nil {
			return nil, errors.Wrap(err, "nil genesis data")
		}
		s.genesisValidatorRoot = res.GenesisValidatorsRoot
	}
	return s.genesisValidatorRoot, nil
}

// Poll the beacon node every syncStatusPollingInterval until the node
// is no longer syncing.
func (s *Service) querySyncStatus(ctx context.Context) {
	status, err := s.cfg.NodeClient.GetSyncStatus(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Could not fetch sync status")
	}
	if status != nil && !status.Syncing {
		log.Info("Beacon node is fully synced, starting slashing detection")
		return
	}
	ticker := time.NewTicker(syncStatusPollingInterval)
	defer ticker.Stop()
	log.Info("Waiting for beacon node to be fully synced...")
	for {
		select {
		case <-ticker.C:
			status, err := s.cfg.NodeClient.GetSyncStatus(ctx, &ptypes.Empty{})
			if err != nil {
				log.WithError(err).Error("Could not fetch sync status")
			}
			if status != nil && !status.Syncing {
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
