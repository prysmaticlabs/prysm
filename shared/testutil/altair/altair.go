package altair

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// DeterministicGenesisStateAltair returns a genesis state in hard fork 1 format made using the deterministic deposits.
func DeterministicGenesisStateAltair(t testing.TB, numValidators uint64) (iface.BeaconStateAltair, []bls.SecretKey) {
	deposits, privKeys, err := testutil.DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := altair.GenesisBeaconState(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	testutil.ResetCache()
	return beaconState, privKeys
}

// NewBeaconBlock creates a beacon block with minimum marshalable fields.
func NewBeaconBlock() *ethpb.SignedBeaconBlockAltair {
	return &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			ParentRoot: make([]byte, 32),
			StateRoot:  make([]byte, 32),
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti:          make([]byte, 32),
				Attestations:      []*ethpb.Attestation{},
				AttesterSlashings: []*ethpb.AttesterSlashing{},
				Deposits:          []*ethpb.Deposit{},
				ProposerSlashings: []*ethpb.ProposerSlashing{},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      make([]byte, len(bitfield.NewBitvector1024())),
					SyncCommitteeSignature: make([]byte, 96),
				},
			},
		},
		Signature: make([]byte, 96),
	}
}

// BlockSignature calculates the post-state root of the block and returns the signature.
func BlockSignature(
	bState iface.BeaconStateAltair,
	block *ethpb.BeaconBlockAltair,
	privKeys []bls.SecretKey,
) (bls.Signature, error) {
	var err error

	s, err := altair.CalculateStateRoot(context.Background(), bState, &ethpb.SignedBeaconBlockAltair{Block: block})
	if err != nil {
		return nil, err
	}
	block.StateRoot = s[:]
	domain, err := helpers.Domain(bState.Fork(), helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	blockRoot, err := helpers.ComputeSigningRoot(block, domain)
	if err != nil {
		return nil, err
	}
	// Temporarily increasing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	currentSlot := bState.Slot()
	if err := bState.SetSlot(block.Slot); err != nil {
		return nil, err
	}
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}
	if err := bState.SetSlot(currentSlot); err != nil {
		return nil, err
	}
	return privKeys[proposerIdx].Sign(blockRoot[:]), nil
}
