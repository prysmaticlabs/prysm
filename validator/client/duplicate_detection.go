// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for N(two) epochs. That is better than starting a duplicate validator and getting slashed.
package client

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	//"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	//"github.com/prysmaticlabs/prysm/shared/params"
)

type DuplicateDetection struct {
	Slot         types.Slot
	DuplicateKey []byte
}

// Starts the Doppelganger detection
func (v *validator) DoppelgangerService(ctx context.Context) ([]byte, error) {
	pKTargets, _, err := v.retrieveValidatingPublicKeysTargets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Doppelganger detection - failed to retrieve validator keys and indices")
	}

	/*NoEpochsToCheck = params.BeaconConfig().DuplicateValidatorEpochsCheck */
	// rpc call to retrieve balances for this validator indices
	req := &ethpb.DetectDoppelgangerRequest{PubKeysTargets: pKTargets}

	res, err := v.validatorClient.DetectDoppelganger(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Doppelganger detection - failed to beacon-rpc list balances")
	}
	return res.PublicKey, nil

}

// Load the PublicKeys and the corresponding Indices of the Validator. Do it once.
func (v *validator) retrieveValidatingPublicKeysTargets(ctx context.Context) ([]*ethpb.PubKeyTarget, [][48]byte, error) {
	validatingPublicKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	pKsTargets := make([]*ethpb.PubKeyTarget, len(validatingPublicKeys))

	// Convert the ValidatingKeys to an array of Indices to be used by Committee retrieval.
	for _, key := range validatingPublicKeys {
		lowestTargetEpoch, exists, err := v.db.LowestSignedTargetEpoch(ctx, key)
		if err != nil {
			return nil, nil, err
		}
		// if validator db has no target, key is not bothered with
		if exists {
			pKT := &ethpb.PubKeyTarget{PubKey: key[:], TargetEpoch: lowestTargetEpoch}
			pKsTargets = append(pKsTargets, pKT)
		}
	}
	return pKsTargets, validatingPublicKeys, nil

}
