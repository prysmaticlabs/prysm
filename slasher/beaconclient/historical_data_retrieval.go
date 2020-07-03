package beaconclient

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
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
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if res == nil {
			res = &ethpb.ListIndexedAttestationsResponse{}
		}
		res, err = bs.beaconClient.ListIndexedAttestations(ctx, &ethpb.ListIndexedAttestationsRequest{
			QueryFilter: &ethpb.ListIndexedAttestationsRequest_Epoch{
				Epoch: epoch,
			},
			PageSize:  int32(cmd.Get().MaxRPCPageSize),
			PageToken: res.NextPageToken,
		})
		if err != nil {
			log.WithError(err).Errorf("could not request indexed attestations for epoch: %d", epoch)
			break
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
	return indexedAtts, nil
}
