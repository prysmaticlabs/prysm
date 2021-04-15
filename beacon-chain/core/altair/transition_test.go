package altair_test

import (
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	statealtair "github.com/prysmaticlabs/prysm/beacon-chain/state/state-altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessEpoch_CanProcess(t *testing.T) {
	epoch := types.Epoch(1)
	slashing := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	base := &pb.BeaconStateAltair{
		Slot:                       params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epoch)) + 1,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  slashing,
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		FinalizedCheckpoint:        &ethpb.Checkpoint{Root: make([]byte, 32)},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
		Validators:                 []*ethpb.Validator{},
	}
	s, err := statealtair.InitializeFromProto(base)
	require.NoError(t, err)
	newState, err := altair.ProcessEpoch(context.Background(), s)
	require.NoError(t, err)
	require.Equal(t, uint64(0), newState.Slashings()[2], "Unexpected slashed balance")
}

func TestFuzzProcessEpoch_1000(t *testing.T) {
	ctx := context.Background()
	state := &stateV0.BeaconState{}
	fuzzer := fuzz.NewWithSeed(0)
	fuzzer.NilChance(0.1)
	for i := 0; i < 1000; i++ {
		fuzzer.Fuzz(state)
		s, err := altair.ProcessEpoch(ctx, state)
		if err != nil && s != nil {
			t.Fatalf("state should be nil on err. found: %v on error: %v for state: %v", s, err, state)
		}
	}
}

func TestProcessBlockNoVerify_PassesProcessingConditions(t *testing.T) {
	beaconState, block, _, _, _ := createFullBlockWithOperations(t)
	set, _, err := altair.ProcessBlockNoVerifyAnySig(context.Background(), beaconState, block)
	require.NoError(t, err)
	// Test Signature set verifies.
	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Could not verify signature set.")
}

func createFullBlockWithOperations(t *testing.T) (iface.BeaconState,
	*ethpb.SignedBeaconBlockAltair, []*ethpb.Attestation, []*ethpb.ProposerSlashing, []*ethpb.SignedVoluntaryExit) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	err = beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	require.NoError(t, err)
	err = beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	require.NoError(t, err)
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cp))
	require.NoError(t, beaconState.AppendCurrentEpochAttestations(&pb.PendingAttestation{}))

	proposerSlashIdx := types.ValidatorIndex(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	err = beaconState.SetSlot(slotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod)) + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(beaconState)
	header1 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("A"), 32),
		},
	})
	header1.Signature, err = helpers.ComputeDomainAndSign(beaconState, currentEpoch, header1.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	header2 := testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerSlashIdx,
			Slot:          1,
			StateRoot:     bytesutil.PadTo([]byte("B"), 32),
		},
	})
	header2.Signature, err = helpers.ComputeDomainAndSign(beaconState, helpers.CurrentEpoch(beaconState), header2.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerSlashIdx])
	require.NoError(t, err)

	proposerSlashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}
	validators := beaconState.Validators()
	validators[proposerSlashIdx].PublicKey = privKeys[proposerSlashIdx].PublicKey().Marshal()
	require.NoError(t, beaconState.SetValidators(validators))

	mockRoot2 := [32]byte{'A'}
	att1 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot2[:]},
		},
		AttestingIndices: []uint64{0, 1},
	})
	domain, err := helpers.Domain(beaconState.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err)
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()

	mockRoot3 := [32]byte{'B'}
	att2 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot3[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AttestingIndices: []uint64{0, 1},
	})

	hashTreeRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	require.NoError(t, err)
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()

	attesterSlashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	var blockRoots [][]byte
	for i := uint64(0); i < uint64(params.BeaconConfig().SlotsPerHistoricalRoot); i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	require.NoError(t, beaconState.SetBlockRoots(blockRoots))

	aggBits := bitfield.NewBitlist(1)
	aggBits.SetBitAt(0, true)
	blockAtt := testutil.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   beaconState.Slot(),
			Target: &ethpb.Checkpoint{Epoch: helpers.CurrentEpoch(beaconState)},
			Source: &ethpb.Checkpoint{Root: mockRoot[:]}},
		AggregationBits: aggBits,
	})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, blockAtt.Data.Slot, blockAtt.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestationutil.AttestingIndices(blockAtt.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	hashTreeRoot, err = helpers.ComputeSigningRoot(blockAtt.Data, domain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	blockAtt.Signature = bls.AggregateSignatures(sigs).Marshal()

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
	require.NoError(t, err)
	header.StateRoot = prevStateRoot[:]
	require.NoError(t, beaconState.SetLatestBlockHeader(header))
	parentRoot, err := beaconState.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(beaconState.Slot()+1))
	randaoReveal, err := testutil.RandaoReveal(copied, currentEpoch, privKeys)
	require.NoError(t, err)
	proposerIndex, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	block := testutil.HydrateSignedBeaconBlockAltair(&ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			ParentRoot:    parentRoot[:],
			Slot:          beaconState.Slot() + 1,
			ProposerIndex: proposerIndex,
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal:      randaoReveal,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      []*ethpb.Attestation{blockAtt},
				VoluntaryExits:    []*ethpb.SignedVoluntaryExit{exit},
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      make([]byte, 1),
					SyncCommitteeSignature: make([]byte, 96),
				},
			},
		},
	})

	sig, err := testutil.BlockSignatureAltair(beaconState, block.Block, privKeys)
	require.NoError(t, err)
	block.Signature = sig.Marshal()

	require.NoError(t, beaconState.SetSlot(block.Block.Slot))
	return beaconState, block, []*ethpb.Attestation{blockAtt}, proposerSlashings, []*ethpb.SignedVoluntaryExit{exit}
}
