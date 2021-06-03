// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for N(two) epochs. That is better than starting a duplicate validator and getting slashed.
package client

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	//ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	//"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	//"github.com/prysmaticlabs/prysm/shared/params"
)

type DuplicateDetection struct {
	Slot         types.Slot
	DuplicateKey []byte
}

type PubKeyTarget struct {
	TargetEpoch types.Epoch
	pubKey      [48]byte
}

// The Public Keys and Indices of this Validator. Retrieve once.
//var validatingPublicKeys [][48]byte
var valIndices []types.ValidatorIndex

//var  pKTargets    []PubKeyTarget

// N epochs to check
var NoEpochsToCheck uint64

// Starts the Doppelganger detection
func (v *validator) StartDoppelgangerService(ctx context.Context) ([48]byte, bool, error) {
	pKTargets, _, err := v.retrieveValidatingPublicKeysTargets(ctx)
	if err != nil {
		return pKTargets[1].pubKey, false, errors.Wrap(err, "Doppelganger detection - failed to retrieve validator keys and indices")
	}

	/*NoEpochsToCheck = params.BeaconConfig().DuplicateValidatorEpochsCheck
	// rpc call to retrieve balances for this validator indices
	req := &ethpb..DetectDoppelgangerRequest{PubKeyTarget:pKTargets,
		NEpochDuplicateCheck: NoEpochsToCheck}

	//res, err := v.beaconClient.ListValidatorBalances(ctx, req)
	if err != nil {
		return nil, false, errors.Wrap(err, "Doppelganger detection - failed to beacon-rpc list balances")
	}
	return nil, false, nil
	*/
	//return res.duplicateFound, res.pubKey, nil
	return pKTargets[1].pubKey, false, nil
}

// Load the PublicKeys and the corresponding Indices of the Validator. Do it once.
func (v *validator) retrieveValidatingPublicKeysTargets(ctx context.Context) ([]PubKeyTarget, [][48]byte, error) {
	validatingPublicKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	pKTargets := make([]PubKeyTarget, len(validatingPublicKeys))

	// Convert the ValidatingKeys to an array of Indices to be used by Committee retrieval.
	for _, key := range validatingPublicKeys {
		lowestTargetEpoch, exists, err := v.db.LowestSignedTargetEpoch(ctx, key)
		if err != nil {
			return nil, nil, err
		}
		pKT := PubKeyTarget{pubKey: key, TargetEpoch: lowestTargetEpoch}
		if exists {
			pKTargets = append(pKTargets, pKT)
		}
	}
	return pKTargets, validatingPublicKeys, nil

}
