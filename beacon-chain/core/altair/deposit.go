package altair

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// ProcessPreGenesisDeposits processes a deposit for the beacon state Altair before chain start.
func ProcessPreGenesisDeposits(
	ctx context.Context,
	beaconState iface.BeaconStateAltair,
	deposits []*ethpb.Deposit,
) (iface.BeaconStateAltair, error) {
	var err error
	beaconState, err = ProcessDeposits(ctx, beaconState, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process deposit")
	}
	beaconState, err = blocks.ActivateValidatorWithEffectiveBalance(beaconState, deposits)
	if err != nil {
		return nil, err
	}
	return beaconState, nil
}

// ProcessDeposits processes validator deposits for beacon state Altair.
func ProcessDeposits(
	ctx context.Context,
	beaconState iface.BeaconStateAltair,
	deposits []*ethpb.Deposit,
) (iface.BeaconStateAltair, error) {
	batchVerified, err := blocks.BatchVerifyDepositsSignatures(ctx, deposits)
	if err != nil {
		return nil, err
	}

	for _, deposit := range deposits {
		if deposit == nil || deposit.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDeposit(ctx, beaconState, deposit, batchVerified)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
		}
	}
	return beaconState, nil
}

// ProcessDeposit processes validator deposit for beacon state Altair.
func ProcessDeposit(ctx context.Context, beaconState iface.BeaconStateAltair, deposit *ethpb.Deposit, verifySignature bool) (iface.BeaconStateAltair, error) {
	beaconState, err := blocks.ProcessDeposit(beaconState, deposit, verifySignature)
	if err != nil {
		return nil, err
	}

	pubKey := deposit.Data.PublicKey
	_, ok := beaconState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok {
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
