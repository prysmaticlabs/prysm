package beaconclient

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// FindOrGetPublicKeys gets public keys from cache or request validators public
// keys from a beacon node via gRPC.
func (s *Service) FindOrGetPublicKeys(
	ctx context.Context,
	validatorIndices []types.ValidatorIndex,
) (map[types.ValidatorIndex][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconclient.FindOrGetPublicKeys")
	defer span.End()

	validators := make(map[types.ValidatorIndex][]byte, len(validatorIndices))
	notFound := 0
	for _, validatorIndex := range validatorIndices {
		pub, exists := s.publicKeyCache.Get(validatorIndex)
		if exists {
			validators[validatorIndex] = pub
			continue
		}
		// inline removal of cached elements from slice
		validatorIndices[notFound] = validatorIndex
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
	vc, err := s.cfg.BeaconClient.ListValidators(ctx, &ethpb.ListValidatorsRequest{
		Indices: validatorIndices,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "could not request validators public key: %d", validatorIndices)
	}
	for _, v := range vc.ValidatorList {
		validators[v.Index] = v.Validator.PublicKey
		s.publicKeyCache.Set(v.Index, v.Validator.PublicKey)
	}
	log.Tracef(
		"Retrieved validators id public key map: %v",
		validators,
	)
	return validators, nil
}
