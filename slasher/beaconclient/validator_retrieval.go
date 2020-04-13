package beaconclient

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// FindOrGetPublicKeys gets public keys from cache or request validators public
// keys from a beacon node via gRPC.
func (bs *Service) FindOrGetPublicKeys(
	ctx context.Context,
	validatorIndices []uint64,
) (map[uint64][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.FindOrGetPublicKeys")
	defer span.End()

	validators := make(map[uint64][]byte, len(validatorIndices))
	notFound := 0
	for _, validatorIdx := range validatorIndices {
		pub, exists := bs.publicKeyCache.Get(validatorIdx)
		if exists {
			validators[validatorIdx] = pub
			continue
		}
		// inline removal of cached elements from slice
		validatorIndices[notFound] = validatorIdx
		notFound++
	}
	// trim the slice to its new size
	validatorIndices = validatorIndices[:notFound]

	if len(validators) > 0 {
		log.Tracef(
			"Retrieved validators public keys from cache: %v",
			validators,
		)
	}

	if notFound == 0 {
		return validators, nil
	}
	vc, err := bs.beaconClient.ListValidators(ctx, &ethpb.ListValidatorsRequest{
		Indices: validatorIndices,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not request validators public key: %d", validatorIndices)
	}
	for _, v := range vc.ValidatorList {
		validators[v.Index] = v.Validator.PublicKey
		bs.publicKeyCache.Set(v.Index, v.Validator.PublicKey)
	}
	log.Tracef(
		"Retrieved validators id public key map: %v",
		validators,
	)
	return validators, nil
}
