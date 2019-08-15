package forkchoice

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// saveValidatorIdx saves the current epoch activated validators public key to index mapping in DB,
// current epoch key is then deleted from ActivatedValidators mapping.
func saveValidatorIdx(ctx context.Context, state *pb.BeaconState, db db.Database) error {
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
		if err := db.SaveValidatorIndex(ctx, bytesutil.ToBytes48(pubKey), idx); err != nil {
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
func deleteValidatorIdx(ctx context.Context, state *pb.BeaconState, db db.Database) error {
	exitedValidators := validators.ExitedValFromEpoch(helpers.CurrentEpoch(state) + 1)
	for _, idx := range exitedValidators {
		pubKey := state.Validators[idx].PublicKey
		if err := db.DeleteValidatorIndex(ctx, bytesutil.ToBytes48(pubKey)); err != nil {
			return errors.Wrap(err, "could not delete validator index")
		}
	}
	validators.DeleteExitedVal(helpers.CurrentEpoch(state))
	return nil
}
