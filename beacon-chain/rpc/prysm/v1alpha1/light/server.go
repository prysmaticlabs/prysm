package light

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/light"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	Fetcher light.UpdatesFetcher
}

// BestUpdates GET /eth/v1alpha1/lightclient/best_update/:periods.
func (s *Server) BestUpdates(ctx context.Context, req *ethpb.BestUpdatesRequest) (*ethpb.BestUpdatesResponse, error) {
	updates := make([]*ethpb.LightClientUpdate, 0)
	for _, period := range req.SyncCommitteePeriods {
		update, err := s.Fetcher.BestUpdateForPeriod(ctx, period)
		if err != nil {
			log.Error(err)
			continue
		}
		updates = append(updates, update)
	}
	return &ethpb.BestUpdatesResponse{Updates: updates}, nil
}

// LatestUpdateFinalized GET /eth/v1alpha1/lightclient/latest_update_finalized/
func (s *Server) LatestUpdateFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
	return s.Fetcher.LatestFinalizedUpdate(ctx), nil
}

// LatestUpdateNonFinalized /eth/v1alpha1/lightclient/latest_update_nonfinalized/
func (s *Server) LatestUpdateNonFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
	return s.Fetcher.LatestNonFinalizedUpdate(ctx), nil
}
