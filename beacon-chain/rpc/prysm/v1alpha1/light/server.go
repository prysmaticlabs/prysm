package light

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type Service struct {
	Database     iface.LightClientDatabase
	prevHeadData map[[32]byte]*ethpb.SyncAttestedData
}

// BestUpdates GET /eth/v1alpha1/lightclient/best_update/:periods.
func (s *Service) BestUpdates(ctx context.Context, req *ethpb.BestUpdatesRequest) (*ethpb.BestUpdatesResponse, error) {
	//const updates: altair.LightClientUpdate[] = [];
	//for (const period of periods) {
	//const update = await this.db.bestUpdatePerCommitteePeriod.get(period);
	//if (update) updates.push(update);
	//}
	//return updates;
	return nil, nil
}

// LatestUpdateFinalized GET /eth/v1alpha1/lightclient/latest_update_finalized/
func (s *Service) LatestUpdateFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
	//return this.db.latestFinalizedUpdate.get();
	return nil, nil
}

// LatestUpdateNonFinalized /eth/v1alpha1/lightclient/latest_update_nonfinalized/
func (s *Service) LatestUpdateNonFinalized(ctx context.Context, _ *empty.Empty) (*ethpb.LightClientUpdate, error) {
	//return this.db.latestNonFinalizedUpdate.get();
	return nil, nil
}
