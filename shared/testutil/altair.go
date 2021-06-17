package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DeterministicGenesisStateAltair returns a genesis state in hard fork 1 format made using the deterministic deposits.
func DeterministicGenesisStateAltair(t testing.TB, numValidators uint64) (iface.BeaconStateAltair, []bls.SecretKey) {
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := altair.GenesisBeaconState(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	ResetCache()
	return beaconState, privKeys
}

// NewBeaconBlockAltair creates a beacon block with minimum marshalable fields.
func NewBeaconBlockAltair() *ethpb.SignedBeaconBlockAltair {
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
					SyncCommitteeBits:      make([]byte, len(bitfield.NewBitvector512())),
					SyncCommitteeSignature: make([]byte, 96),
				},
			},
		},
		Signature: make([]byte, 96),
	}
}

// BlockSignatureAltair calculates the post-state root of the block and returns the signature.
func BlockSignatureAltair(
	bState iface.BeaconStateAltair,
	block *ethpb.BeaconBlockAltair,
	privKeys []bls.SecretKey,
) (bls.Signature, error) {
	var err error

	bState, err = state.ProcessSlots(context.Background(), bState, block.Slot)
	if err != nil {
		return nil, err
	}
	bState, err = state.ProcessBlockForStateRoot(context.Background(), bState, interfaces.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: block}))
	if err != nil {
		return nil, err
	}
	r, err := bState.HashTreeRoot(context.Background())
	if err != nil {
		return nil, err
	}

	block.StateRoot = r[:]
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

// GenerateFullBlockAltair generates a fully valid block with the requested parameters.
// Use BlockGenConfig to declare the conditions you would like the block generated under.
func GenerateFullBlockAltair(
	bState iface.BeaconState,
	privs []bls.SecretKey,
	conf *BlockGenConfig,
	slot types.Slot,
) (*ethpb.SignedBeaconBlockAltair, error) {
	ctx := context.Background()
	currentSlot := bState.Slot()
	if currentSlot > slot {
		return nil, fmt.Errorf("current slot in state is larger than given slot. %d > %d", currentSlot, slot)
	}
	bState = bState.Copy()

	if conf == nil {
		conf = &BlockGenConfig{}
	}

	var err error
	var pSlashings []*ethpb.ProposerSlashing
	numToGen := conf.NumProposerSlashings
	if numToGen > 0 {
		pSlashings, err = generateProposerSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d proposer slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttesterSlashings
	var aSlashings []*ethpb.AttesterSlashing
	if numToGen > 0 {
		aSlashings, err = generateAttesterSlashings(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	numToGen = conf.NumAttestations
	var atts []*ethpb.Attestation
	if numToGen > 0 {
		atts, err = GenerateAttestations(bState, privs, numToGen, slot, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attestations:", numToGen)
		}
	}

	numToGen = conf.NumDeposits
	var newDeposits []*ethpb.Deposit
	eth1Data := bState.Eth1Data()
	if numToGen > 0 {
		newDeposits, eth1Data, err = generateDepositsAndEth1Data(bState, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d deposits:", numToGen)
		}
	}

	numToGen = conf.NumVoluntaryExits
	var exits []*ethpb.SignedVoluntaryExit
	if numToGen > 0 {
		exits, err = generateVoluntaryExits(bState, privs, numToGen)
		if err != nil {
			return nil, errors.Wrapf(err, "failed generating %d attester slashings:", numToGen)
		}
	}

	newHeader := bState.LatestBlockHeader()
	prevStateRoot, err := bState.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	newHeader.StateRoot = prevStateRoot[:]
	parentRoot, err := newHeader.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	if slot == currentSlot {
		slot = currentSlot + 1
	}

	syncAgg, err := generateSyncCommittees(bState, privs, parentRoot)
	if err != nil {
		return nil, err
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	if err := bState.SetSlot(slot); err != nil {
		return nil, err
	}
	reveal, err := RandaoReveal(bState, helpers.CurrentEpoch(bState), privs)
	if err != nil {
		return nil, err
	}

	idx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}

	block := &ethpb.BeaconBlockAltair{
		Slot:          slot,
		ParentRoot:    parentRoot[:],
		ProposerIndex: idx,
		Body: &ethpb.BeaconBlockBodyAltair{
			Eth1Data:          eth1Data,
			RandaoReveal:      reveal,
			ProposerSlashings: pSlashings,
			AttesterSlashings: aSlashings,
			Attestations:      atts,
			VoluntaryExits:    exits,
			Deposits:          newDeposits,
			Graffiti:          make([]byte, 32),
			SyncAggregate:     syncAgg,
		},
	}
	if err := bState.SetSlot(currentSlot); err != nil {
		return nil, err
	}

	signature, err := BlockSignatureAltair(bState, block, privs)
	if err != nil {
		return nil, err
	}

	return &ethpb.SignedBeaconBlockAltair{Block: block, Signature: signature.Marshal()}, nil
}
