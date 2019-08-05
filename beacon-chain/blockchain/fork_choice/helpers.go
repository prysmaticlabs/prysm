package forkchoice

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// logs epoch related data in each epoch transition
func logEpochData(beaconState *pb.BeaconState) {
	log.WithField(
		"previousJustifiedEpoch", beaconState.PreviousJustifiedCheckpoint.Epoch,
	).Info("Previous justified epoch")
	log.WithField(
		"justifiedEpoch", beaconState.CurrentJustifiedCheckpoint.Epoch,
	).Info("Justified epoch")
	log.WithField(
		"finalizedEpoch", beaconState.FinalizedCheckpoint.Epoch,
	).Info("Finalized epoch")
	log.WithField(
		"Deposit Index", beaconState.Eth1DepositIndex,
	).Info("ETH1 Deposit Index")
	log.WithField(
		"numValidators", len(beaconState.Validators),
	).Info("Validator registry length")
}

// saveValidatorIdx saves the current epoch activated validators public key to index mapping in DB,
// current epoch key is then deleted from ActivatedValidators mapping.
func saveValidatorIdx(state *pb.BeaconState, db *db.BeaconDB) error {
	nextEpoch := helpers.CurrentEpoch(state) + 1
	activatedValidators := validators.ActivatedValFromEpoch(nextEpoch)
	var idxNotInState []uint64
	for _, idx := range activatedValidators {
		// If for some reason the activated validator indices is not in state,
		// we skip them and save them to process for next epoch.
		if int(idx) >= len(state.Validators) {
			idxNotInState = append(idxNotInState, idx)
			continue
		}
		pubKey := state.Validators[idx].PublicKey
		if err := db.SaveValidatorIndex(pubKey, int(idx)); err != nil {
			return errors.Wrap(err, "could not save validator index")
		}
	}
	// Since we are processing next epoch, save the can't processed validator indices
	// to the epoch after that.
	validators.InsertActivatedIndices(nextEpoch+1, idxNotInState)
	validators.DeleteActivatedVal(helpers.CurrentEpoch(state))
	return nil
}

// deleteValidatorIdx deletes the current epoch exited validators public key to index mapping in DB,
// current epoch key is then deleted from ExitedValidators mapping.
func deleteValidatorIdx(state *pb.BeaconState, db *db.BeaconDB) error {
	exitedValidators := validators.ExitedValFromEpoch(helpers.CurrentEpoch(state) + 1)
	for _, idx := range exitedValidators {
		pubKey := state.Validators[idx].PublicKey
		if err := db.DeleteValidatorIndex(pubKey); err != nil {
			return errors.Wrap(err, "could not delete validator index")
		}
	}
	validators.DeleteExitedVal(helpers.CurrentEpoch(state))
	return nil
}
