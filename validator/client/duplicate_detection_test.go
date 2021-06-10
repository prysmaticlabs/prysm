// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for N(two) epochs. That is better than starting a duplicate validator and getting slashed.
package client

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// The Doppelganger detection calls beacon to calculate if the balance of a duplicate validator has been
// strictly increasing for the previous N epochs straight.
func (v *validator) DoppelgangerService(ctx context.Context) ([]byte, error) {
	pKTargets, _, err := v.retrieveValidatingPublicKeysTargets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Doppelganger detection - failed to retrieve validator keys and indices")
	}

	// rpc call to retrieve balances for this validator indices
	req := &ethpb.DetectDoppelgangerRequest{PubKeysTargets: pKTargets}
	res, err := v.validatorClient.DetectDoppelganger(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Doppelganger detection - failed in beacon-rpc call")
	}
	return res.PublicKey, nil

}

// Load the PublicKeys and the corresponding Targets.
func (v *validator) retrieveValidatingPublicKeysTargets(ctx context.Context) ([]*ethpb.PubKeyTarget, [][48]byte, error) {
	validatingPublicKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	pKsTargets := make([]*ethpb.PubKeyTarget, len(validatingPublicKeys))

	// Find the lowest signed Target for each Key and append to return struct
	for _, key := range validatingPublicKeys {
		lowestTargetEpoch, exists, err := v.db.LowestSignedTargetEpoch(ctx, key)
		if err != nil {
			return nil, nil, err
		}
		// if validator db has no target, key is not appended
		if exists {
			pKT := &ethpb.PubKeyTarget{PubKey: key[:], TargetEpoch: lowestTargetEpoch}
			pKsTargets = append(pKsTargets, pKT)
		}
	}
	return pKsTargets, validatingPublicKeys, nil

}
