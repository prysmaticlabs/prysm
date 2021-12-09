package util

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
)

// DeterministicGenesisStateAltair returns a genesis state in hard fork 1 format made using the deterministic deposits.
func DeterministicGenesisStateAltair(t testing.TB, numValidators uint64) (state.BeaconStateAltair, []bls.SecretKey) {
	deposits, privKeys, err := DeterministicDepositsAndKeys(numValidators)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get %d deposits", numValidators))
	}
	eth1Data, err := DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get eth1data for %d deposits", numValidators))
	}
	beaconState, err := GenesisBeaconState(context.Background(), deposits, uint64(0), eth1Data)
	if err != nil {
		t.Fatal(errors.Wrapf(err, "failed to get genesis beacon state of %d validators", numValidators))
	}
	resetCache()
	return beaconState, privKeys
}

// GenesisBeaconState returns the genesis beacon state.
func GenesisBeaconState(ctx context.Context, deposits []*ethpb.Deposit, genesisTime uint64, eth1Data *ethpb.Eth1Data) (state.BeaconStateAltair, error) {
	st, err := emptyGenesisState()
	if err != nil {
		return nil, err
	}

	// Process initial deposits.
	st, err = helpers.UpdateGenesisEth1Data(st, deposits, eth1Data)
	if err != nil {
		return nil, err
	}

	st, err = processPreGenesisDeposits(ctx, st, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator deposits")
	}

	return buildGenesisBeaconState(genesisTime, st, st.Eth1Data())
}

// processPreGenesisDeposits processes a deposit for the beacon state Altair before chain start.
func processPreGenesisDeposits(
	ctx context.Context,
	beaconState state.BeaconStateAltair,
	deposits []*ethpb.Deposit,
) (state.BeaconStateAltair, error) {
	var err error
	beaconState, err = altair.ProcessDeposits(ctx, beaconState, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process deposit")
	}
	beaconState, err = blocks.ActivateValidatorWithEffectiveBalance(beaconState, deposits)
	if err != nil {
		return nil, err
	}
	return beaconState, nil
}

func buildGenesisBeaconState(genesisTime uint64, preState state.BeaconStateAltair, eth1Data *ethpb.Eth1Data) (state.BeaconStateAltair, error) {
	if eth1Data == nil {
		return nil, errors.New("no eth1data provided for genesis state")
	}

	var randaoMixes [customtypes.RandaoMixesSize][32]byte
	for i := 0; i < len(randaoMixes); i++ {
		var h [32]byte
		copy(h[:], eth1Data.BlockHash)
		randaoMixes[i] = h
	}

	zeroHash32 := params.BeaconConfig().ZeroHash
	zeroHash := zeroHash32[:]

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		activeIndexRoots[i] = zeroHash
	}

	var blockRoots [customtypes.BlockRootsSize][32]byte
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = zeroHash32
	}

	var stateRoots [customtypes.StateRootsSize][32]byte
	for i := 0; i < len(stateRoots); i++ {
		stateRoots[i] = zeroHash32
	}

	slashings := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)

	genesisValidatorsRoot, err := stateutil.ValidatorRegistryRoot(preState.Validators())
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash tree root genesis validators %v", err)
	}

	prevEpochParticipation, err := preState.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	currEpochParticipation, err := preState.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	scores, err := preState.InactivityScores()
	if err != nil {
		return nil, err
	}

	bodyRoot, err := (&ethpb.BeaconBlockBodyAltair{
		RandaoReveal: make([]byte, 96),
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		},
		Graffiti: make([]byte, 32),
		SyncAggregate: &ethpb.SyncAggregate{
			SyncCommitteeBits:      make([]byte, len(bitfield.NewBitvector512())),
			SyncCommitteeSignature: make([]byte, 96),
		},
	}).HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not hash tree root empty block body")
	}

	var pubKeys [][]byte
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		pubKeys = append(pubKeys, bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength))
	}

	s, err := stateAltair.Initialize()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize state from proto state")
	}

	if err = s.SetGenesisTime(genesisTime); err != nil {
		return nil, errors.Wrap(err, "could not set genesis time")
	}
	if err = s.SetGenesisValidatorRoot(genesisValidatorsRoot); err != nil {
		return nil, errors.Wrap(err, "could not set genesis validators root")
	}
	if err = s.SetSlot(0); err != nil {
		return nil, errors.Wrap(err, "could not set slot")
	}
	if err = s.SetFork(&ethpb.Fork{
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		Epoch:           0,
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = s.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		ParentRoot: zeroHash,
		StateRoot:  zeroHash,
		BodyRoot:   bodyRoot[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set latest block header")
	}
	if err = s.SetBlockRoots(&blockRoots); err != nil {
		return nil, errors.Wrap(err, "could not set block roots")
	}
	if err = s.SetStateRoots(&stateRoots); err != nil {
		return nil, errors.Wrap(err, "could not set state roots")
	}
	if err = s.SetHistoricalRoots([][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = s.SetEth1Data(eth1Data); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = s.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = s.SetEth1Data(eth1Data); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = s.SetEth1DepositIndex(preState.Eth1DepositIndex()); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 deposit index")
	}
	if err = s.SetValidators(preState.Validators()); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = s.SetBalances(preState.Balances()); err != nil {
		return nil, errors.Wrap(err, "could not set balances")
	}
	if err = s.SetRandaoMixes(&randaoMixes); err != nil {
		return nil, errors.Wrap(err, "could not set randao mixes")
	}
	if err = s.SetPreviousParticipationBits(prevEpochParticipation); err != nil {
		return nil, errors.Wrap(err, "could not set previous epoch participation")
	}
	if err = s.SetCurrentParticipationBits(currEpochParticipation); err != nil {
		return nil, errors.Wrap(err, "could not set current epoch participation")
	}
	if err = s.SetInactivityScores(scores); err != nil {
		return nil, errors.Wrap(err, "could not set inactivity scores")
	}
	if err = s.SetSlashings(slashings); err != nil {
		return nil, errors.Wrap(err, "could not set slashings")
	}
	if err = s.SetJustificationBits([]byte{0}); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}
	if err = s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set previous justified checkpoint")
	}
	if err = s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set current justified checkpoint")
	}
	if err = s.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set finalized checkpoint")
	}
	if err = s.SetCurrentSyncCommittee(&ethpb.SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}); err != nil {
		return nil, errors.Wrap(err, "could not set current sync committee")
	}
	if err = s.SetNextSyncCommittee(&ethpb.SyncCommittee{
		Pubkeys:         bytesutil.SafeCopy2dBytes(pubKeys),
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}); err != nil {
		return nil, errors.Wrap(err, "could not set next sync committee")
	}

	return s, nil
}

func emptyGenesisState() (state.BeaconStateAltair, error) {
	s, err := stateAltair.Initialize()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize state from proto state")
	}

	if err = s.SetSlot(0); err != nil {
		return nil, errors.Wrap(err, "could not set slot")
	}
	if err = s.SetFork(&ethpb.Fork{
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion:  params.BeaconConfig().AltairForkVersion,
		Epoch:           0,
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = s.SetHistoricalRoots([][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = s.SetEth1Data(&ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = s.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = s.SetEth1DepositIndex(0); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 deposit index")
	}
	if err = s.SetValidators([]*ethpb.Validator{}); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = s.SetBalances([]uint64{}); err != nil {
		return nil, errors.Wrap(err, "could not set balances")
	}
	if err = s.SetInactivityScores([]uint64{}); err != nil {
		return nil, errors.Wrap(err, "could not set inactivity scores")
	}
	if err = s.SetPreviousParticipationBits([]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set previous participation bits")
	}
	if err = s.SetCurrentParticipationBits([]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set current participation bits")
	}
	if err = s.SetJustificationBits([]byte{0}); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}

	return s, nil
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
	bState state.BeaconStateAltair,
	block *ethpb.BeaconBlockAltair,
	privKeys []bls.SecretKey,
) (bls.Signature, error) {
	var err error
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: block})
	if err != nil {
		return nil, err
	}
	s, err := transition.CalculateStateRoot(context.Background(), bState, wsb)
	if err != nil {
		return nil, err
	}
	block.StateRoot = s[:]
	gvRoot := bState.GenesisValidatorRoot()
	domain, err := signing.Domain(bState.Fork(), time.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer, gvRoot[:])
	if err != nil {
		return nil, err
	}
	blockRoot, err := signing.ComputeSigningRoot(block, domain)
	if err != nil {
		return nil, err
	}
	// Temporarily increasing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	currentSlot := bState.Slot()
	if err := bState.SetSlot(block.Slot); err != nil {
		return nil, err
	}
	proposerIdx, err := helpers.BeaconProposerIndex(context.Background(), bState)
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
	bState state.BeaconState,
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

	syncAgg, err := generateSyncAggregate(bState, privs, parentRoot)
	if err != nil {
		return nil, err
	}

	// Temporarily incrementing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	if err := bState.SetSlot(slot); err != nil {
		return nil, err
	}
	reveal, err := RandaoReveal(bState, time.CurrentEpoch(bState), privs)
	if err != nil {
		return nil, err
	}

	idx, err := helpers.BeaconProposerIndex(ctx, bState)
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
