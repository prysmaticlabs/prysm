package beaconclient

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// RequestValidator requests validator public key from a beacon node via gRPC.
func (bs *Service) RequestValidator(
	ctx context.Context,
	validatorIdx uint64,
) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.RequestValidators")
	defer span.End()

	pub, exists := bs.validatorCache.Get(validatorIdx)
	if exists {
		log.Tracef(
			"Retrieved validator id: %d from cache, public key: %v",
			validatorIdx,
			pub,
		)
		return pub, nil
	}
	res, err := bs.beaconClient.GetValidator(ctx, &ethpb.GetValidatorRequest{
		QueryFilter: &ethpb.GetValidatorRequest_Index{
			Index: validatorIdx,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not request validator public key: %d", validatorIdx)
	}
	log.Tracef(
		"Retrieved validator id: %d public key: %v",
		validatorIdx,
		res.PublicKey,
	)
	bs.validatorCache.Set(validatorIdx, res.PublicKey)
	return res.PublicKey, nil
}
