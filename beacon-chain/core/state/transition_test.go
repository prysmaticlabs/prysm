package state_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func init() {
	state.SkipSlotCache.Disable()
}

func TestExecuteStateTransition_IncorrectSlot(t *testing.T) {
	base := &pb.BeaconState{
		Slot: 5,
	}
	beaconState, err := beaconstate.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 4,
		},
	}
	want := "expected state.slot"
	_, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestExecuteStateTransition_FullProcess(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  []byte{2},
	}
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch - 1); err != nil {
		t.Fatal(err)
	}
	e := beaconState.Eth1Data()
	e.DepositCount = 100
	if err := beaconState.SetEth1Data(e); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{Slot: beaconState.Slot()}); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetEth1DataVotes([]*ethpb.Eth1Data{eth1Data}); err != nil {
		t.Fatal(err)
	}

	oldMix, err := beaconState.RandaoMixAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	parentRoot, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	if err := beaconState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Fatal(err)
	}

	nextSlotState := beaconState.Copy()
	if err := nextSlotState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	if err != nil {
		t.Error(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          beaconState.Slot() + 1,
			ParentRoot:    parentRoot[:],
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: randaoReveal,
				Eth1Data:     eth1Data,
			},
		},
	}

	stateRoot, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	block.Block.StateRoot = stateRoot[:]

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	block.Signature = sig.Marshal()

	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
	if err != nil {
		t.Error(err)
	}

	if beaconState.Slot() != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Unexpected Slot number, expected: 64, received: %d", beaconState.Slot())
	}

	mix, err := beaconState.RandaoMixAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(mix, oldMix) {
		t.Errorf("Did not expect new and old randao mix to equal, %#x == %#x", mix, oldMix)
	}
}

func TestExecuteStateTransitionNoVerify_FullProcess(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	eth1Data := &ethpb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  bytesutil.PadTo([]byte{2}, 32),
		BlockHash:    make([]byte, 32),
	}
	if err := beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch - 1); err != nil {
		t.Fatal(err)
	}
	e := beaconState.Eth1Data()
	e.DepositCount = 100
	if err := beaconState.SetEth1Data(e); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{Slot: beaconState.Slot()}); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetEth1DataVotes([]*ethpb.Eth1Data{eth1Data}); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	if err := beaconState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Fatal(err)
	}

	nextSlotState := beaconState.Copy()
	if err := nextSlotState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	if err != nil {
		t.Error(err)
	}
	block := testutil.NewBeaconBlock()
	block.Block.ProposerIndex = proposerIdx
	block.Block.Slot = beaconState.Slot() + 1
	block.Block.ParentRoot = parentRoot[:]
	block.Block.Body.RandaoReveal = randaoReveal
	block.Block.Body.Eth1Data = eth1Data

	stateRoot, err := state.CalculateStateRoot(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}

	block.Block.StateRoot = stateRoot[:]

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	block.Signature = sig.Marshal()

	set, beaconState, err := state.ExecuteStateTransitionNoVerifyAnySig(context.Background(), beaconState, block)
	if err != nil {
		t.Error(err)
	}
	verified, err := set.Verify()
	if err != nil {
		t.Error(err)
	}
	if !verified {
		t.Error("Could not verify signature set")
	}

}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	slashing := &ethpb.ProposerSlashing{
		Header_1: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:       params.BeaconConfig().SlotsPerEpoch,
				ParentRoot: make([]byte, 32),
				StateRoot:  make([]byte, 32),
				BodyRoot:   make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
		Header_2: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:       params.BeaconConfig().SlotsPerEpoch * 2,
				ParentRoot: make([]byte, 32),
				StateRoot:  make([]byte, 32),
				BodyRoot:   make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
	}
	block.Block.Body.ProposerSlashings = []*ethpb.ProposerSlashing{slashing}

	if err := beaconState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Fatal(err)
	}
	block.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), block.Block, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	beaconState, err = state.ProcessSlots(context.Background(), beaconState, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "could not process block proposer slashing"
	_, err = state.ProcessBlock(context.Background(), beaconState, block)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AggregationBits: bitfield.NewBitlist(3),
	}

	block, err := testutil.GenerateFullBlock(beaconState, privKeys, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	block.Block.Body.Attestations = []*ethpb.Attestation{att}
	if err := beaconState.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Fatal(err)
	}
	block.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), block.Block, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	beaconState, err = state.ProcessSlots(context.Background(), beaconState, 1)
	if err != nil {
		t.Fatal(err)
	}

	want := "could not process block attestations"
	_, err = state.ProcessBlock(context.Background(), beaconState, block)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
					Slot:          1,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte("A"), 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
					Slot:          1,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: bytesutil.PadTo([]byte("B"), 96),
			},
		},
	}
	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				},
				AttestingIndices: []uint64{0, 1},
				Signature:        make([]byte, 96),
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
					Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				},
				AttestingIndices: []uint64{0, 1},
				Signature:        make([]byte, 96),
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}
	blockAtt := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
	}
	attestations := []*ethpb.Attestation{blockAtt}
	var exits []*ethpb.SignedVoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &ethpb.SignedVoluntaryExit{})
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		BodyRoot:   bodyRoot[:],
		StateRoot:  make([]byte, 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	parentRoot, err := beaconState.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: parentRoot[:],
			Slot:       1,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal:      make([]byte, 96),
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      attestations,
				VoluntaryExits:    exits,
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: bytesutil.PadTo([]byte{2}, 32),
					BlockHash:   bytesutil.PadTo([]byte{3}, 32),
				},
			},
		},
	}
	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}
	cp := beaconState.CurrentJustifiedCheckpoint()
	cp.Root = []byte("hello-world")
	if err := beaconState.SetCurrentJustifiedCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}
	if _, err := state.ProcessBlock(context.Background(), beaconState, block); err == nil {
		t.Error("Expected err, received nil")
	}
}

func createFullBlockWithOperations(t *testing.T) (*beaconstate.BeaconState,
	*ethpb.SignedBeaconBlock, []*ethpb.Attestation, []*ethpb.ProposerSlashing, []*ethpb.SignedVoluntaryExit) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	if err != nil {
		t.Fatal(err)
	}
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	proposerSlashIdx := uint64(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	err = beaconState.SetSlot((params.BeaconConfig().ShardCommitteePeriod * slotsPerEpoch) + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	currentEpoch := helpers.CurrentEpoch(beaconState)
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("A"), 32),
			ParentRoot:    make([]byte, 32),
			BodyRoot:      make([]byte, 32),
		},
	}
	header1.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, header1.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("B"), 32),
			ParentRoot:    make([]byte, 32),
			BodyRoot:      make([]byte, 32),
		},
	}
	header2.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), header2.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}
	validators := beaconState.Validators()
	validators[proposerSlashIdx].PublicKey = privKeys[proposerSlashIdx].PublicKey().Marshal()[:]
	if err := beaconState.SetValidators(validators); err != nil {
		t.Fatal(err)
	}

	mockRoot2 := [32]byte{'A'}
	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: mockRoot2[:]},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			BeaconBlockRoot: make([]byte, 32),
		},
		AttestingIndices: []uint64{0, 1},
		Signature:        make([]byte, 96),
	}
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	mockRoot3 := [32]byte{'B'}
	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: mockRoot3[:]},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			BeaconBlockRoot: make([]byte, 32),
		},
		AttestingIndices: []uint64{0, 1},
		Signature:        make([]byte, 96),
	}

	hashTreeRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	if err := beaconState.SetBlockRoots(blockRoots); err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(0, true)
	blockAtt := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            beaconState.Slot(),
			BeaconBlockRoot: make([]byte, 32),
			Target:          &ethpb.Checkpoint{Epoch: helpers.CurrentEpoch(beaconState), Root: make([]byte, 32)},
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  mockRoot[:],
			}},
		AggregationBits: aggBits,
		Signature:       make([]byte, 96),
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, blockAtt.Data.Slot, blockAtt.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices := attestationutil.AttestingIndices(blockAtt.AggregationBits, committee)
	if err != nil {
		t.Error(err)
	}
	hashTreeRoot, err = helpers.ComputeSigningRoot(blockAtt.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	blockAtt.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 10,
			Epoch:          0,
		},
	}
	exit.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, exit.Exit, params.BeaconConfig().DomainVoluntaryExit, privKeys[exit.Exit.ValidatorIndex])
	require.NoError(t, err)

	header := beaconState.LatestBlockHeader()
	prevStateRoot, err := beaconState.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	header.StateRoot = prevStateRoot[:]
	if err := beaconState.SetLatestBlockHeader(header); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	if err != nil {
		t.Fatal(err)
	}
	copied := beaconState.Copy()
	if err := copied.SetSlot(beaconState.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	randaoReveal, err := testutil.RandaoReveal(copied, currentEpoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	proposerIndex, err := helpers.BeaconProposerIndex(copied)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot:    parentRoot[:],
			Slot:          beaconState.Slot() + 1,
			ProposerIndex: proposerIndex,
			Body: &ethpb.BeaconBlockBody{
				Graffiti:          make([]byte, 32),
				RandaoReveal:      randaoReveal,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      []*ethpb.Attestation{blockAtt},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{exit},
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: bytesutil.PadTo([]byte{2}, 32),
					BlockHash:   bytesutil.PadTo([]byte{3}, 32),
				},
			},
		},
	}

	sig, err := testutil.BlockSignature(beaconState, block.Block, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	block.Signature = sig.Marshal()

	if beaconState.SetSlot(block.Block.Slot) != nil {
		t.Fatal(err)
	}
	return beaconState, block, []*ethpb.Attestation{blockAtt}, proposerSlashings, []*ethpb.SignedVoluntaryExit{exit}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	beaconState, block, _, proposerSlashings, exits := createFullBlockWithOperations(t)
	exit := exits[0]

	beaconState, err := state.ProcessBlock(context.Background(), beaconState, block)
	if err != nil {
		t.Fatalf("Expected block to pass processing conditions: %v", err)
	}

	v, err := beaconState.ValidatorAtIndex(proposerSlashings[0].Header_1.Header.ProposerIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Slashed {
		t.Errorf("Expected validator at index %d to be slashed, received false", proposerSlashings[0].Header_1.Header.ProposerIndex)
	}
	v, err = beaconState.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Slashed {
		t.Error("Expected validator at index 1 to be slashed, received false")
	}

	v, err = beaconState.ValidatorAtIndex(exit.Exit.ValidatorIndex)
	if err != nil {
		t.Fatal(err)
	}
	received := v.ExitEpoch
	wanted := params.BeaconConfig().FarFutureEpoch
	if received == wanted {
		t.Errorf("Expected validator at index %d to be exiting, did not expect: %d", exit.Exit.ValidatorIndex, wanted)
	}
}

func TestProcessBlockNoVerify_PassesProcessingConditions(t *testing.T) {
	beaconState, block, _, _, _ := createFullBlockWithOperations(t)
	set, _, err := state.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, block)
	if err != nil {
		t.Fatal(err)
	}
	// Test Signature set verifies.
	verified, err := set.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("Could not verify signature set.")
	}

}

func TestProcessEpochPrecompute_CanProcess(t *testing.T) {
	epoch := uint64(1)

	atts := []*pb.PendingAttestation{{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}, InclusionDelay: 1}}
	slashing := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	base := &pb.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  slashing,
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
		Validators:                 []*ethpb.Validator{},
	}
	s, err := beaconstate.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := state.ProcessEpochPrecompute(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	wanted := uint64(0)
	if newState.Slashings()[2] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Slashings()[2])
	}
}
func BenchmarkProcessBlk_65536Validators_FullBlock(b *testing.B) {
	logrus.SetLevel(logrus.PanicLevel)

	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	committeeCount := validatorCount / params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	base := &pb.BeaconState{
		Slot:              20,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{},
		BlockRoots:        make([][]byte, 254),
		RandaoMixes:       randaoMixes,
		Validators:        validators,
		Balances:          validatorBalances,
		Slashings:         make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Root: []byte("hello-world"),
		},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
	}
	s, err := beaconstate.InitializeFromProto(base)
	if err != nil {
		b.Fatal(err)
	}

	// Set up proposer slashing object for block
	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
				Signature: bytesutil.PadTo([]byte("A"), 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
				Signature: bytesutil.PadTo([]byte("B"), 96),
			},
		},
	}

	// Set up attester slashing object for block
	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: make([]byte, 32),
					Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				},
				AttestingIndices: []uint64{2, 3},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: make([]byte, 32),
					Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				},
				AttestingIndices: []uint64{2, 3},
			},
		},
	}

	// Set up deposit object for block
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: []byte{1, 2, 3},
			Amount:    params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	leaf, err := deposit.Data.HashTreeRoot()
	if err != nil {
		b.Fatal(err)
	}
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		b.Fatalf("Could not generate proof: %v", err)
	}
	deposit.Proof = proof
	root := depositTrie.Root()

	// Set up randao reveal object for block
	proposerIdx, err := helpers.BeaconProposerIndex(s)
	if err != nil {
		b.Fatal(err)
	}
	priv := bls.RandKey()
	v := s.Validators()
	v[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, 0)
	domain, err := helpers.Domain(s.Fork(), 0, params.BeaconConfig().DomainRandao, s.GenesisValidatorRoot())
	if err != nil {
		b.Fatal(err)
	}
	ctr := &pb.SigningData{
		ObjectRoot: buf,
		Domain:     domain,
	}
	root, err = ctr.HashTreeRoot()
	if err != nil {
		b.Fatal(err)
	}
	epochSignature := priv.Sign(root[:])

	buf = []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("A")
	hashed := hashutil.Hash(pubKey)
	buf = append(buf, hashed[:]...)
	v[3].WithdrawalCredentials = buf

	if err := s.SetValidators(v); err != nil {
		b.Fatal(err)
	}

	attestations := make([]*ethpb.Attestation, 128)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Root: []byte("hello-world")}},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0,
				0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		}
	}

	blk := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: s.Slot(),
			Body: &ethpb.BeaconBlockBody{
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot: root[:],
					BlockHash:   root[:],
				},
				RandaoReveal:      epochSignature.Marshal(),
				Attestations:      attestations,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
			},
		},
	}

	// Precache the shuffled indices
	for i := uint64(0); i < committeeCount; i++ {
		if _, err := helpers.BeaconCommitteeFromState(s, 0, i); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := state.ProcessBlock(context.Background(), s, blk)
		if err != nil {
			b.Fatal(err)
		}
		// Reset state fields to process block again
		v := s.Validators()
		v[1].Slashed = false
		v[2].Slashed = false
		if err := s.SetValidators(v); err != nil {
			b.Fatal(err)
		}
		balances := s.Balances()
		balances[3] += 2 * params.BeaconConfig().MinDepositAmount
		if err := s.SetBalances(balances); err != nil {
			b.Fatal(err)
		}
	}
}

func TestProcessBlk_AttsBasedOnValidatorCount(t *testing.T) {
	logrus.SetLevel(logrus.PanicLevel)

	// Default at 256 validators, can raise this number with faster BLS.
	validatorCount := uint64(256)
	s, privKeys := testutil.DeterministicGenesisState(t, validatorCount)
	if err := s.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	bitCount := validatorCount / params.BeaconConfig().SlotsPerEpoch
	aggBits := bitfield.NewBitlist(bitCount)
	for i := uint64(1); i < bitCount; i++ {
		aggBits.SetBitAt(i, true)
	}
	atts := make([]*ethpb.Attestation, 1)

	for i := 0; i < len(atts); i++ {
		att := &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Slot:            1,
				Source:          &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
				Target:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				BeaconBlockRoot: make([]byte, 32),
			},
			AggregationBits: aggBits,
			Signature:       make([]byte, 96),
		}

		committee, err := helpers.BeaconCommitteeFromState(s, att.Data.Slot, att.Data.CommitteeIndex)
		if err != nil {
			t.Error(err)
		}
		attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
		if err != nil {
			t.Error(err)
		}
		domain, err := helpers.Domain(s.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, s.GenesisValidatorRoot())
		if err != nil {
			t.Fatal(err)
		}
		sigs := make([]bls.Signature, len(attestingIndices))
		for i, indice := range attestingIndices {
			hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
			if err != nil {
				t.Error(err)
			}
			sig := privKeys[indice].Sign(hashTreeRoot[:])
			sigs[i] = sig
		}
		att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]
		atts[i] = att
	}

	copied := s.Copy()
	if err := copied.SetSlot(s.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	epochSignature, err := testutil.RandaoReveal(copied, helpers.CurrentEpoch(copied), privKeys)
	if err != nil {
		t.Fatal(err)
	}
	header := s.LatestBlockHeader()
	prevStateRoot, err := s.HashTreeRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	header.StateRoot = prevStateRoot[:]
	if err := s.SetLatestBlockHeader(header); err != nil {
		t.Fatal(err)
	}

	parentRoot, err := stateutil.BlockHeaderRoot(s.LatestBlockHeader())
	if err != nil {
		t.Fatal(err)
	}

	nextSlotState := s.Copy()
	if err := nextSlotState.SetSlot(s.Slot() + 1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(nextSlotState)
	if err != nil {
		t.Error(err)
	}
	blk := testutil.NewBeaconBlock()
	blk.Block.ProposerIndex = proposerIdx
	blk.Block.Slot = s.Slot() + 1
	blk.Block.ParentRoot = parentRoot[:]
	blk.Block.Body.RandaoReveal = epochSignature
	blk.Block.Body.Attestations = atts
	sig, err := testutil.BlockSignature(s, blk.Block, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	blk.Signature = sig.Marshal()

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.MinAttestationInclusionDelay = 0
	params.OverrideBeaconConfig(config)

	if s.SetSlot(s.Slot()+1) != nil {
		t.Fatal(err)
	}
	if _, err := state.ProcessBlock(context.Background(), s, blk); err != nil {
		t.Fatal(err)
	}
}

func TestCanProcessEpoch_TrueOnEpochs(t *testing.T) {
	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		}, {
			slot:            63,
			canProcessEpoch: true,
		},
		{
			slot:            64,
			canProcessEpoch: false,
		}, {
			slot:            127,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: false,
		},
	}

	for _, tt := range tests {
		b := &pb.BeaconState{Slot: tt.slot}
		s, err := beaconstate.InitializeFromProto(b)
		if err != nil {
			t.Fatal(err)
		}
		if state.CanProcessEpoch(s) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				state.CanProcessEpoch(s),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestProcessOperations_OverMaxProposerSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxProposerSlashings
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: make([]*ethpb.ProposerSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of proposer slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.ProposerSlashings), params.BeaconConfig().MaxProposerSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&beaconstate.BeaconState{},
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttesterSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxAttesterSlashings
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: make([]*ethpb.AttesterSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of attester slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.AttesterSlashings), params.BeaconConfig().MaxAttesterSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&beaconstate.BeaconState{},
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttestations(t *testing.T) {
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: make([]*ethpb.Attestation, params.BeaconConfig().MaxAttestations+1),
		},
	}

	want := fmt.Sprintf("number of attestations (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.Attestations), params.BeaconConfig().MaxAttestations)
	if _, err := state.ProcessOperations(
		context.Background(),
		&beaconstate.BeaconState{},
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperation_OverMaxVoluntaryExits(t *testing.T) {
	maxExits := params.BeaconConfig().MaxVoluntaryExits
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: make([]*ethpb.SignedVoluntaryExit, maxExits+1),
		},
	}

	want := fmt.Sprintf("number of voluntary exits (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.VoluntaryExits), maxExits)
	if _, err := state.ProcessOperations(
		context.Background(),
		&beaconstate.BeaconState{},
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_IncorrectDeposits(t *testing.T) {
	base := &pb.BeaconState{
		Eth1Data:         &ethpb.Eth1Data{DepositCount: 100},
		Eth1DepositIndex: 98,
	}
	s, err := beaconstate.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{{}},
		},
	}

	want := fmt.Sprintf("incorrect outstanding deposits in block body, wanted: %d, got: %d",
		s.Eth1Data().DepositCount-s.Eth1DepositIndex(), len(block.Body.Deposits))
	if _, err := state.ProcessOperations(
		context.Background(),
		s,
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessSlots_SameSlotAsParentState(t *testing.T) {
	slot := uint64(2)
	parentState, err := beaconstate.InitializeFromProto(&pb.BeaconState{Slot: slot})
	if err != nil {
		t.Fatal(err)
	}

	wanted := "expected state.slot 2 < slot 2"
	if _, err := state.ProcessSlots(context.Background(), parentState, slot); err.Error() != wanted {
		t.Error("Did not get wanted error")
	}
}

func TestProcessSlots_LowerSlotAsParentState(t *testing.T) {
	slot := uint64(2)
	parentState, err := beaconstate.InitializeFromProto(&pb.BeaconState{Slot: slot})
	if err != nil {
		t.Fatal(err)
	}

	wanted := "expected state.slot 2 < slot 1"
	if _, err := state.ProcessSlots(context.Background(), parentState, slot-1); err.Error() != wanted {
		t.Error("Did not get wanted error")
	}
}
