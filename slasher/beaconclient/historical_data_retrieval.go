package beaconclient

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// RequestHistoricalAttestations requests all indexed attestations for a
// given epoch from a beacon node via gRPC.
func (bs *Service) RequestHistoricalAttestations(
	ctx context.Context,
	epoch uint64,
) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.RequestHistoricalAttestations")
	defer span.End()
	indexedAtts := make([]*ethpb.IndexedAttestation, 0)
	res := &ethpb.ListIndexedAttestationsResponse{}
	var err error
	for {
		res, err = bs.beaconClient.ListIndexedAttestations(ctx, &ethpb.ListIndexedAttestationsRequest{
			QueryFilter: &ethpb.ListIndexedAttestationsRequest_TargetEpoch{
				TargetEpoch: epoch,
			},
			PageSize:  int32(params.BeaconConfig().DefaultPageSize),
			PageToken: res.NextPageToken,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not request indexed attestations for epoch: %d", epoch)
		}
		indexedAtts = append(indexedAtts, res.IndexedAttestations...)
		log.Infof(
			"Retrieved %d/%d indexed attestations for epoch %d",
			len(indexedAtts),
			res.TotalSize,
			epoch,
		)
		if res.NextPageToken == "" || res.TotalSize == 0 || len(indexedAtts) == int(res.TotalSize) {
			break
		}
	}
	if err := bs.slasherDB.SaveIncomingIndexedAttestations(ctx, indexedAtts); err != nil {
		return nil, errors.Wrap(err, "could not save indexed attestations")
	}
	return indexedAtts, nil
}
