package blocks_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	if err != nil {
		t.Fatal(err)
	}

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Error(err)
	}
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
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, dt)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := priv.Sign(signingRoot[:])
	block.Signature = blockSig.Marshal()[:]

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Fatal(err)
	}
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	if err != nil {
		t.Fatal(err)
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	if err == nil || err.Error() != "block.Slot 10 must be greater than state.LatestBlockHeader.Slot 10" {
		t.Fatalf("did not get expected error, got %v", err)
	}
}

func TestProcessBlockHeader_WrongProposerSig(t *testing.T) {
	testutil.ResetCache()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{Slot: 9}); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetSlot(10); err != nil {
		t.Error(err)
	}

	lbhdr, err := stateutil.BlockHeaderRoot(beaconState.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}

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
	dt, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, dt)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := privKeys[proposerIdx+1].Sign(signingRoot[:])
	block.Signature = blockSig.Marshal()[:]

	_, err = blocks.ProcessBlockHeader(beaconState, block)
	want := "signature did not verify"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	lbhsr, err := state.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	root, err := helpers.ComputeSigningRoot([]byte("hello"), dt)
	if err != nil {
		t.Error(err)
	}
	blockSig := priv.Sign(root[:])
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 1,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: lbhsr[:],
		},
		Signature: blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "is different than block slot"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	root, err := helpers.ComputeSigningRoot([]byte("hello"), dt)
	if err != nil {
		t.Error(err)
	}
	blockSig := priv.Sign(root[:])
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	pID, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Error(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: pID,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: []byte{'A'},
		},
		Signature: blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "does not match"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

	parentRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	root, err := helpers.ComputeSigningRoot([]byte("hello"), dt)
	if err != nil {
		t.Error(err)
	}
	blockSig := priv.Sign(root[:])

	validators[12683].PublicKey = priv.PublicKey().Marshal()
	pID, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Error(err)
	}
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: pID,
			Slot:          10,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: parentRoot[:],
		},
		Signature: blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "was previously slashed"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_OK(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Error(err)
	}
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
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, dt)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := priv.Sign(signingRoot[:])
	block.Signature = blockSig.Marshal()[:]
	bodyRoot, err := stateutil.BlockBodyRoot(block.Block.Body)
	if err != nil {
		t.Fatalf("Failed to hash block bytes got: %v", err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Fatal(err)
	}
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	if err != nil {
		t.Fatal(err)
	}

	newState, err := blocks.ProcessBlockHeader(state, block)
	if err != nil {
		t.Fatalf("Failed to process block header got: %v", err)
	}
	var zeroHash [32]byte
	nsh := newState.LatestBlockHeader()
	expected := &ethpb.BeaconBlockHeader{
		ProposerIndex: pID,
		Slot:          block.Block.Slot,
		ParentRoot:    latestBlockSignedRoot[:],
		BodyRoot:      bodyRoot[:],
		StateRoot:     zeroHash[:],
	}
	if !proto.Equal(nsh, expected) {
		t.Errorf("Expected %v, received %v", expected, nsh)
	}
}

func TestBlockSignatureSet_OK(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}

	latestBlockSignedRoot, err := stateutil.BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt, err := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatalf("Failed to get domain form state: %v", err)
	}
	priv := bls.RandKey()
	pID, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Error(err)
	}
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
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, dt)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := priv.Sign(signingRoot[:])
	block.Signature = blockSig.Marshal()[:]

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Fatal(err)
	}
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()
	err = state.UpdateValidatorAtIndex(proposerIdx, validators[proposerIdx])
	if err != nil {
		t.Fatal(err)
	}
	set, err := blocks.BlockSignatureSet(state, block)
	if err != nil {
		t.Fatal(err)
	}

	verified, err := set.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("Block signature set returned a set which was unable to be verified")
	}
}
