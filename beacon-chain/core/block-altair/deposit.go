package block_altair

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

// ProcessPreGenesisDepositsV1 processes a deposit for the beacon state hard fork 1 before chainstart.
func ProcessPreGenesisDepositsV1(
	ctx context.Context,
	beaconState iface.BeaconStateAltair,
	deposits []*ethpb.Deposit,
) (iface.BeaconStateAltair, error) {

}

func ProcessDeposits(
	ctx context.Context,
	beaconState iface.BeaconStateAltair,
	b *ethpb.SignedBeaconBlock,
) (iface.BeaconStateAltair, error) {
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}

	deposits := b.Block.Body.Deposits
	var err error
	domain, err := helpers.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	if err != nil {
		return nil, err
	}

	// Attempt to verify all deposit signatures at once, if this fails then fall back to processing
	// individual deposits with signature verification enabled.
	var verifySignature bool
	if err := verifyDepositDataWithDomain(ctx, deposits, domain); err != nil {
		log.WithError(err).Debug("Failed to verify deposit data, verifying signatures individually")
		verifySignature = true
	}

	for _, deposit := range deposits {
		if deposit == nil || deposit.Data == nil {
			return nil, errors.New("got a nil deposit in block")
		}
		beaconState, err = ProcessDepositV1(beaconState, deposit, verifySignature)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process deposit from %#x", bytesutil.Trunc(deposit.Data.PublicKey))
		}
	}
	return beaconState, nil
}

func ProcessDeposit(beaconState iface.BeaconStateAltair, deposit *ethpb.Deposit, verifySignature bool) (iface.BeaconStateAltair, error) {
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
