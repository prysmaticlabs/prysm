package blocks_test

import (
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(ioutil.Discard) // Ignore "validator activated" logs
}

func TestProcessBlockHeader_ImproperBlockSlot(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              10,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 10}, // Must be less than block.Slot
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(state)
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: pID,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: latestBlockSignedRoot[:],
		},
	}
	block.Signature, err = helpers.ComputeDomainAndSign(state, currentEpoch, block.Block, params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	require.NoError(t, err)

	_, err = blocks.ProcessBlockHeader(state, block)
	assert.ErrorContains(t, "block.Slot 10 must be greater than state.LatestBlockHeader.Slot 10", err)
}

func TestProcessBlockHeader_WrongProposerSig(t *testing.T) {
	testutil.ResetCache()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	require.NoError(t, beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{Slot: 9}))
	require.NoError(t, beaconState.SetSlot(10))

	lbhdr, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	require.NoError(t, err)

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	require.NoError(t, err)

	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: lbhdr[:],
		},
	}
	block.Signature, err = helpers.ComputeDomainAndSign(beaconState, 0, block.Block, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx+1])
	require.NoError(t, err)

	_, err = blocks.ProcessBlockHeader(beaconState, block)
	want := "signature did not verify"
	assert.ErrorContains(t, want, err)
}

func TestProcessBlockHeader_DifferentSlots(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              10,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	lbhsr, err := state.LatestBlockHeader().HashTreeRoot()
	require.NoError(t, err)
	currentEpoch := helpers.CurrentEpoch(state)

	priv := bls.RandKey()
	blockSig, err := helpers.ComputeDomainAndSign(state, currentEpoch, []byte("hello"), params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 1,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: lbhsr[:],
		},
		Signature: blockSig,
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "is different than block slot"
	assert.ErrorContains(t, want, err)
}

func TestProcessBlockHeader_PreviousBlockRootNotSignedRoot(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              10,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(state)
	priv := bls.RandKey()
	blockSig, err := helpers.ComputeDomainAndSign(state, currentEpoch, []byte("hello"), params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	pID, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: pID,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: []byte{'A'},
		},
		Signature: blockSig,
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "does not match"
	assert.ErrorContains(t, want, err)
}

func TestProcessBlockHeader_SlashedProposer(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              10,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	parentRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	require.NoError(t, err)
	currentEpoch := helpers.CurrentEpoch(state)
	priv := bls.RandKey()
	blockSig, err := helpers.ComputeDomainAndSign(state, currentEpoch, []byte("hello"), params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)

	validators[12683].PublicKey = priv.PublicKey().Marshal()
	pID, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: pID,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: parentRoot[:],
		},
		Signature: blockSig,
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "was previously slashed"
	assert.ErrorContains(t, want, err)
}

func TestProcessBlockHeader_OK(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey: make([]byte, 32),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := testutil.NewBeaconState()
	require.NoError(t,state.SetValidators(validators))
	require.NoError(t,state.SetSlot(10))
	require.NoError(t,state.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:          9,
		ProposerIndex: 0,
		ParentRoot:    make([]byte, 32),
		StateRoot:     make([]byte, 32),
		BodyRoot:      make([]byte, 32),
	}))

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(state)
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	block.Block.ProposerIndex = pID
	block.Block.Slot = 10
	block.Block.Body.RandaoReveal = bytesutil.PadTo([]byte{'A', 'B', 'C'}, 96)
	block.Block.ParentRoot = latestBlockSignedRoot[:]
	block.Signature, err = helpers.ComputeDomainAndSign(state, currentEpoch, block.Block, params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)
	bodyRoot, err := stateutil.BlockBodyRoot(block.Block.Body)
	require.NoError(t, err, "Failed to hash block bytes got")

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	require.NoError(t, err)

	newState, err := blocks.ProcessBlockHeader(state, block)
	require.NoError(t, err, "Failed to process block header got")
	var zeroHash [32]byte
	nsh := newState.LatestBlockHeader()
	expected := &ethpb.BeaconBlockHeader{
		ProposerIndex: pID,
		Slot:          block.Block.Slot,
		ParentRoot:    latestBlockSignedRoot[:],
		BodyRoot:      bodyRoot[:],
		StateRoot:     zeroHash[:],
	}
	assert.Equal(t, true, proto.Equal(nsh, expected), "Expected %v, received %v", expected, nsh)
}

func TestBlockSignatureSet_OK(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey: make([]byte, 32),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := testutil.NewBeaconState()
	require.NoError(t, state.SetValidators(validators))
	require.NoError(t, state.SetSlot(10))
	require.NoError(t,state.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:          9,
		ProposerIndex: 0,
		ParentRoot:    make([]byte, 32),
		StateRoot:     make([]byte, 32),
		BodyRoot:      make([]byte, 32),
	}))

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	require.NoError(t, err)

	currentEpoch := helpers.CurrentEpoch(state)
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	block := testutil.NewBeaconBlock()
	block.Block.Slot = 10
	block.Block.ProposerIndex = pID
	block.Block.Body.RandaoReveal = bytesutil.PadTo([]byte{'A', 'B', 'C'}, 96)
	block.Block.ParentRoot = latestBlockSignedRoot[:]
	block.Signature, err = helpers.ComputeDomainAndSign(state, currentEpoch, block.Block, params.BeaconConfig().DomainBeaconProposer, priv)
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(state)
	require.NoError(t, err)
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	require.NoError(t, err)
	set, err := blocks.BlockSignatureSet(state, block)
	require.NoError(t, err)

	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Block signature set returned a set which was unable to be verified")
}
