package exit

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// ValidateVoluntaryExit validates the voluntary exit.
// If it is invalid for some reason an error, if valid it will return no error.
func ValidateVoluntaryExit(state *stateTrie.BeaconState, genesisTime time.Time, signed *ethpb.SignedVoluntaryExit) error {
	if signed == nil || signed.Exit == nil {
		return errors.New("nil signed voluntary exit")
	}
	ve := signed.Exit
	validator, err := state.ValidatorAtIndex(ve.ValidatorIndex)
	if err != nil {
		return err
	}

	if !helpers.IsActiveValidator(validator, ve.Epoch) {
		return fmt.Errorf("validator %d not active at epoch %d", ve.ValidatorIndex, ve.Epoch)
	}
	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("validator %d already exiting or exited", ve.ValidatorIndex)
	}

	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	currentEpoch := uint64(roughtime.Now().Unix()-genesisTime.Unix()) / secondsPerEpoch
	earliestRequestedExitEpoch := mathutil.Max(ve.Epoch, currentEpoch)
	earliestExitEpoch := validator.ActivationEpoch + params.BeaconConfig().PersistentCommitteePeriod
	if earliestRequestedExitEpoch < earliestExitEpoch {
		return fmt.Errorf("validator %d cannot exit before epoch %d", ve.ValidatorIndex, earliestExitEpoch)
	}

	// Confirm signature is valid
	root, err := ssz.HashTreeRoot(ve)
	if err != nil {
		return errors.Wrap(err, "cannot confirm signature")
	}
	sig, err := bls.SignatureFromBytes(signed.Signature)
	if err != nil {
		return errors.Wrap(err, "malformed signature")
	}
	validatorPubKey, err := bls.PublicKeyFromBytes(validator.PublicKey)
	if err != nil {
		return errors.Wrap(err, "invalid validator public key")
	}
	domain := bls.ComputeDomain(params.BeaconConfig().DomainVoluntaryExit)
	verified := sig.Verify(root[:], validatorPubKey, domain)
	if !verified {
		return errors.New("incorrect signature")
	}

	// Parameters are valid.
	return nil
}
