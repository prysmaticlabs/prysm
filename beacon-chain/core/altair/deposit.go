package altair

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// ProcessDeposits processes validator deposits for beacon state Altair.
func ProcessDeposits(
	ctx context.Context,
	beaconState types.BeaconState,
	deposits []*ethpb.Deposit,
) (types.BeaconState, error) {
	batchVerified, err := blocks.BatchVerifyDepositsSignatures(ctx, deposits)
	if err != nil {
		return nil, err
	}

	for _, deposit := range deposits {
		if deposit == nil || deposit.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDeposit(beaconState, deposit, batchVerified)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
		}
	}
	return beaconState, nil
}

// ProcessDeposit processes validator deposit for beacon state Altair.
func ProcessDeposit(beaconState types.BeaconState, deposit *ethpb.Deposit, verifySignature bool) (types.BeaconState, error) {
	beaconState, isNewValidator, err := blocks.ProcessDeposit(beaconState, deposit, verifySignature)
	if err != nil {
		return nil, err
	}
	if isNewValidator {
		if err := beaconState.AppendInactivityScore(0); err != nil {
			return nil, err
		}
		if err := beaconState.AppendPreviousParticipationBits(0); err != nil {
			return nil, err
		}
		if err := beaconState.AppendCurrentParticipationBits(0); err != nil {
			return nil, err
		}
	}

	return beaconState, nil
}
