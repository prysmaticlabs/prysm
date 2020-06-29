package blocks_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
	"gopkg.in/d4l3k/messagediff.v1"
)

func init() {
	logrus.SetOutput(ioutil.Discard) // Ignore "validator activated" logs
}

func TestProcessBlockHeader_WrongProposerSig(t *testing.T) {
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

	lbhsr, err := ssz.HashTreeRoot(state.LatestBlockHeader())
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

func TestProcessRandao_IncorrectProposerFailsVerification(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: buf, Domain: domain})
	if err != nil {
		t.Fatal(err)
	}
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(root[:])
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	want := "block randao: signature did not verify"
	if _, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	epoch := helpers.CurrentEpoch(beaconState)
	epochSignature, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	newState, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
	)
	if err != nil {
		t.Errorf("Unexpected error processing block randao: %v", err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	mix := newState.RandaoMixes()[currentEpoch%params.BeaconConfig().EpochsPerHistoricalVector]

	if bytes.Equal(mix, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf(
			"Expected empty signature to be overwritten by randao reveal, received %v",
			params.BeaconConfig().EmptySignature,
		)
	}
}

func TestProcessEth1Data_SetsCorrectly(t *testing.T) {
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
	})
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}

	period := params.BeaconConfig().EpochsPerEth1VotingPeriod * params.BeaconConfig().SlotsPerEpoch
	for i := uint64(0); i < period; i++ {
		beaconState, err = blocks.ProcessEth1DataInBlock(beaconState, block)
		if err != nil {
			t.Fatal(err)
		}
	}

	newETH1DataVotes := beaconState.Eth1DataVotes()
	if len(newETH1DataVotes) <= 1 {
		t.Error("Expected new ETH1 data votes to have length > 1")
	}
	if !proto.Equal(beaconState.Eth1Data(), stateTrie.CopyETH1Data(block.Body.Eth1Data)) {
		t.Errorf(
			"Expected latest eth1 data to have been set to %v, received %v",
			block.Body.Eth1Data,
			beaconState.Eth1Data(),
		)
	}
}
func TestProcessProposerSlashings_UnmatchedHeaderSlots(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 20)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          params.BeaconConfig().SlotsPerEpoch + 1,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}
	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header slots"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_SameHeaders(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}

	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_ValidatorNotSlashable(t *testing.T) {
	registry := []*ethpb.Validator{
		{
			PublicKey:         []byte("key"),
			Slashed:           true,
			ActivationEpoch:   0,
			WithdrawableEpoch: 0,
		},
	}
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
				},
				Signature: []byte("A"),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
				},
				Signature: []byte("B"),
			},
		},
	}

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		bytesutil.ToBytes48(beaconState.Validators()[0].PublicKey),
	)

	_, err = blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	proposerIdx := uint64(1)

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			StateRoot:     []byte("A"),
		},
	}
	signingRoot, err := helpers.ComputeSigningRoot(header1.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()[:]

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			StateRoot:     []byte("B"),
		},
	}
	signingRoot, err = helpers.ComputeSigningRoot(header2.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()[:]

	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	newStateVals := newState.Validators()
	if newStateVals[1].ExitEpoch != beaconState.Validators()[1].ExitEpoch {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			newStateVals[1].ExitEpoch, beaconState.Validators()[1].ExitEpoch)
	}
}

func TestSlashableAttestationData_CanSlash(t *testing.T) {
	att1 := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1},
		Source: &ethpb.Checkpoint{Root: []byte{'A'}},
	}
	att2 := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1},
		Source: &ethpb.Checkpoint{Root: []byte{'B'}},
	}
	if !blocks.IsSlashableAttestationData(att1, att2) {
		t.Error("atts should have been slashable")
	}
	att1.Target.Epoch = 4
	att1.Source.Epoch = 2
	att2.Source.Epoch = 3
	if !blocks.IsSlashableAttestationData(att1, att2) {
		t.Error("atts should have been slashable")
	}
}

func TestProcessAttesterSlashings_DataNotSlashable(t *testing.T) {
	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 1},
				},
			},
		},
	}
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("attestations are not slashable")

	_, err = blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_IndexedAttestationFailedToVerify(t *testing.T) {
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	want := fmt.Sprint("validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE")
	_, err = blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	for _, vv := range beaconState.Validators() {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AttestingIndices: []uint64{0, 1},
	}
	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	sig0 := privKeys[0].Sign(signingRoot[:])
	sig1 := privKeys[1].Sign(signingRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AttestingIndices: []uint64{0, 1},
	}
	signingRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	sig0 = privKeys[0].Sign(signingRoot[:])
	sig1 = privKeys[1].Sign(signingRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.Validators()

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be slashed and exited. We confirm this below.
	if newRegistry[1].ExitEpoch != beaconState.Validators()[1].ExitEpoch {
		t.Errorf(
			`
			Expected validator at index 1's exit epoch to match
			%d, received %d instead
			`,
			beaconState.Validators()[1].ExitEpoch,
			newRegistry[1].ExitEpoch,
		)
	}
}

func TestProcessAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0},
				Slot:   5,
			},
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	want := fmt.Sprintf(
		"attestation slot %d + inclusion delay %d > state slot %d",
		attestations[0].Data.Slot,
		params.BeaconConfig().MinAttestationInclusionDelay,
		beaconState.Slot(),
	)
	_, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}}}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	if err := beaconState.SetPreviousJustifiedCheckpoint(pfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(
		"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
		att.Data.Target.Epoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_CurrentEpochFFGDataMismatches(t *testing.T) {
	aggBits := bitfield.NewBitlist(3)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0},
				Source: &ethpb.Checkpoint{Epoch: 1},
			},
			AggregationBits: aggBits,
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	if err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay); err != nil {
		t.Fatal(err)
	}
	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	if err := beaconState.SetCurrentJustifiedCheckpoint(cfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.CurrentEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	_, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Source.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint().Root,
		attestations[0].Data.Source.Root,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{Epoch: 1},
				Slot:   params.BeaconConfig().SlotsPerEpoch,
			},
			AggregationBits: aggBits,
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	if err := beaconState.SetPreviousJustifiedCheckpoint(pfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Source.Epoch = helpers.PrevEpoch(beaconState)
	block.Body.Attestations[0].Data.Target.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint().Root,
		attestations[0].Data.Source.Root,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_InvalidAggregationBitsLength(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(4)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: aggBits,
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	if err := beaconState.SetCurrentJustifiedCheckpoint(cfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	expected := "failed to verify aggregation bitfield: wanted participants bitfield length 3, got: 4"
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Errorf("Did not receive wanted error")
	}
}

func TestProcessAttestations_OK(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
		},
		AggregationBits: aggBits,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		t.Error(err)
	}
	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestProcessAggregatedAttestation_OverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
	}
	aggBits1 := bitfield.NewBitlist(4)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	aggBits1.SetBitAt(2, true)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	if err := beaconState.SetCurrentJustifiedCheckpoint(cfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices1 := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggBits2 := bitfield.NewBitlist(4)
	aggBits2.SetBitAt(1, true)
	aggBits2.SetBitAt(2, true)
	aggBits2.SetBitAt(3, true)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
	}

	committee, err = helpers.BeaconCommitteeFromState(beaconState, att2.Data.Slot, att2.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices2 := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err = helpers.ComputeSigningRoot(data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs = make([]bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	if _, err = attaggregation.AggregatePair(att1, att2); err != aggregation.ErrBitsOverlap {
		t.Error("Did not receive wanted error")
	}
}

func TestProcessAggregatedAttestation_NoOverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 300)

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
	}
	aggBits1 := bitfield.NewBitlist(9)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = mockRoot[:]
	if err := beaconState.SetCurrentJustifiedCheckpoint(cfc); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices1 := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggBits2 := bitfield.NewBitlist(9)
	aggBits2.SetBitAt(2, true)
	aggBits2.SetBitAt(3, true)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
	}

	committee, err = helpers.BeaconCommitteeFromState(beaconState, att2.Data.Slot, att2.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices2 := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err = helpers.ComputeSigningRoot(data, domain)
	if err != nil {
		t.Error(err)
	}
	sigs = make([]bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggregatedAtt, err := attaggregation.AggregatePair(att1, att2)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{aggregatedAtt},
		},
	}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); err != nil {
		t.Error(err)
	}
}

func TestProcessAttestationsNoVerify_IncorrectSlotTargetEpoch(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 1)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   params.BeaconConfig().SlotsPerEpoch,
			Target: &ethpb.Checkpoint{},
		},
	}
	wanted := fmt.Sprintf("data slot is not in the same epoch as target %d != %d", helpers.SlotToEpoch(att.Data.Slot), att.Data.Target.Epoch)
	_, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att)
	if err == nil || err.Error() != wanted {
		t.Error("Did not get wanted error")
	}
}

func TestProcessAttestationsNoVerify_OK(t *testing.T) {
	// Attestation with an empty signature

	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(1, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: aggBits,
	}

	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}
	ckp := beaconState.CurrentJustifiedCheckpoint()
	copy(ckp.Root, "hello-world")
	if err := beaconState.SetCurrentJustifiedCheckpoint(ckp); err != nil {
		t.Fatal(err)
	}
	if err := beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}); err != nil {
		t.Fatal(err)
	}

	if _, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestConvertToIndexed_OK(t *testing.T) {
	helpers.ClearCache()
	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:        5,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		aggregationBitfield    bitfield.Bitlist
		wantedAttestingIndices []uint64
	}{
		{
			aggregationBitfield:    bitfield.Bitlist{0x07},
			wantedAttestingIndices: []uint64{43, 47},
		},
		{
			aggregationBitfield:    bitfield.Bitlist{0x03},
			wantedAttestingIndices: []uint64{47},
		},
		{
			aggregationBitfield:    bitfield.Bitlist{0x01},
			wantedAttestingIndices: []uint64{},
		},
	}

	var sig [96]byte
	copy(sig[:], "signed")
	attestation := &ethpb.Attestation{
		Signature: sig[:],
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
	}
	for _, tt := range tests {
		attestation.AggregationBits = tt.aggregationBitfield
		wanted := &ethpb.IndexedAttestation{
			AttestingIndices: tt.wantedAttestingIndices,
			Data:             attestation.Data,
			Signature:        attestation.Signature,
		}

		committee, err := helpers.BeaconCommitteeFromState(state, attestation.Data.Slot, attestation.Data.CommitteeIndex)
		if err != nil {
			t.Error(err)
		}
		ia := attestationutil.ConvertToIndexed(context.Background(), attestation, committee)
		if !reflect.DeepEqual(wanted, ia) {
			diff, _ := messagediff.PrettyDiff(ia, wanted)
			t.Log(diff)
			t.Error("convert attestation to indexed attestation didn't result as wanted")
		}
	}
}

func TestVerifyIndexedAttestation_OK(t *testing.T) {
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		attestation *ethpb.IndexedAttestation
	}{
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 2,
				},
			},
			AttestingIndices: []uint64{1},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 1,
				},
			},
			AttestingIndices: []uint64{47, 99, 101},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 4,
				},
			},
			AttestingIndices: []uint64{21, 72},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 7,
				},
			},
			AttestingIndices: []uint64{100, 121, 122},
		}},
	}

	for _, tt := range tests {
		domain, err := helpers.Domain(state.Fork(), tt.attestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, state.GenesisValidatorRoot())
		if err != nil {
			t.Fatal(err)
		}
		root, err := helpers.ComputeSigningRoot(tt.attestation.Data, domain)
		if err != nil {
			t.Error(err)
		}
		var sig []bls.Signature
		for _, idx := range tt.attestation.AttestingIndices {
			validatorSig := keys[idx].Sign(root[:])
			sig = append(sig, validatorSig)
		}
		aggSig := bls.AggregateSignatures(sig)
		marshalledSig := aggSig.Marshal()

		tt.attestation.Signature = marshalledSig

		err = blocks.VerifyIndexedAttestation(context.Background(), state, tt.attestation)
		if err != nil {
			t.Errorf("failed to verify indexed attestation: %v", err)
		}
	}
}

func TestValidateIndexedAttestation_AboveMaxLength(t *testing.T) {
	indexedAtt1 := &ethpb.IndexedAttestation{
		AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+5),
	}

	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee+5; i++ {
		indexedAtt1.AttestingIndices[i] = i
		indexedAtt1.Data = &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{
				Epoch: i,
			},
		}
	}

	want := "validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE"
	err := blocks.VerifyIndexedAttestation(context.Background(), &stateTrie.BeaconState{}, indexedAtt1)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected verification to fail return false, received: %v", err)
	}
}

func TestProcessDeposits_SameValidatorMultipleDepositsSameBlock(t *testing.T) {
	// Same validator created 3 valid deposits within the same block
	testutil.ResetCache()
	dep, _, err := testutil.DeterministicDepositsAndKeysSameValidator(3)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			// 3 deposits from the same validator
			Deposits: []*ethpb.Deposit{dep[0], dep[1], dep[2]},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}

	if len(newState.Validators()) != 2 {
		t.Errorf("Incorrect validator count. Wanted %d, got %d", 2, len(newState.Validators()))
	}
}

func TestProcessDeposits_MerkleBranchFailsVerification(t *testing.T) {
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: []byte{1, 2, 3},
			Signature: make([]byte, 96),
		},
	}
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit.Proof = proof
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{deposit},
		},
	}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "deposit root did not verify"
	_, err = blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessDeposits_AddsNewValidatorDeposit(t *testing.T) {
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{dep[0]},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.Balances()[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances()[1],
		)
	}
}

func TestProcessDeposits_RepeatedDeposit_IncreasesValidatorBalance(t *testing.T) {
	sk := bls.RandKey()
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: sk.PublicKey().Marshal(),
			Amount:    1000,
		},
	}
	sr, err := helpers.ComputeSigningRoot(deposit.Data, bytesutil.ToBytes(3, 8))
	if err != nil {
		t.Fatal(err)
	}
	sig := sk.Sign(sr[:])
	deposit.Data.Signature = sig.Marshal()
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit.Proof = proof
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Deposits: []*ethpb.Deposit{deposit},
		},
	}
	registry := []*ethpb.Validator{
		{
			PublicKey: []byte{1, 2, 3},
		},
		{
			PublicKey:             sk.PublicKey().Marshal(),
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{0, 50}
	root := depositTrie.Root()
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: root[:],
			BlockHash:   root[:],
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body.Deposits)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.Balances()[1] != 1000+50 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.Balances()[1])
	}
}

func TestProcessDeposit_AddsNewValidatorDeposit(t *testing.T) {
	//Similar to TestProcessDeposits_AddsNewValidatorDeposit except that this test directly calls ProcessDeposit
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(dep))
	if err != nil {
		t.Fatal(err)
	}

	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Error(err)
	}
	newState, err := blocks.ProcessDeposit(beaconState, dep[0], true, )
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.Validators()) != 2 {
		t.Errorf("Expected validator list to have length 2, received: %v", len(newState.Validators()))
	}
	if len(newState.Balances()) != 2 {
		t.Fatalf("Expected validator balances list to have length 2, received: %v", len(newState.Balances()))
	}
	if newState.Balances()[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 1 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances()[1],
		)
	}
}

func TestProcessDeposit_SkipsInvalidDeposit(t *testing.T) {
	// Same test settings as in TestProcessDeposit_AddsNewValidatorDeposit, except that we use an invalid signature
	dep, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatal(err)
	}
	dep[0].Data.Signature = make([]byte, 96)
	trie, _, err := testutil.DepositTrieFromDeposits(dep)
	if err != nil {
		t.Fatal(err)
	}
	root := trie.Root()
	eth1Data := &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: 1,
	}
	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newState, err := blocks.ProcessDeposit(beaconState, dep[0], true, )
	if err != nil {
		t.Fatalf("Expected invalid block deposit to be ignored without error, received: %v", err)
	}

	if newState.Eth1DepositIndex() != 1 {
		t.Errorf(
			"Expected Eth1DepositIndex to be increased by 1 after processing an invalid deposit, received change: %v",
			newState.Eth1DepositIndex(),
		)
	}
	if len(newState.Validators()) != 1 {
		t.Errorf("Expected validator list to have length 1, received: %v", len(newState.Validators()))
	}
	if len(newState.Balances()) != 1 {
		t.Errorf("Expected validator balances list to have length 1, received: %v", len(newState.Balances()))
	}
	if newState.Balances()[0] != 0 {
		t.Errorf("Expected validator balance at index 0 to stay 0, received: %v", newState.Balances()[0])
	}
}

func TestProcessVoluntaryExits_ValidatorNotActive(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: 0,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "non-active validator cannot exit"

	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_InvalidExitEpoch(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				Epoch: 10,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "expected current epoch >= exit epoch"

	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_NotActiveLongEnoughToExit(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
				Epoch:          0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "validator has not been active long enough to exit"
	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_AppliesCorrectStatus(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
				Epoch:          0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = state.SetSlot(state.Slot() + (params.BeaconConfig().ShardCommitteePeriod * params.BeaconConfig().SlotsPerEpoch))
	if err != nil {
		t.Fatal(err)
	}

	priv := bls.RandKey()
	val, err := state.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	val.PublicKey = priv.PublicKey().Marshal()[:]
	if err := state.UpdateValidatorAtIndex(0, val); err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(exits[0].Exit, domain)
	if err != nil {
		t.Error(err)
	}
	sig := priv.Sign(signingRoot[:])
	exits[0].Signature = sig.Marshal()
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	newState, err := blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.Validators()
	if newRegistry[0].ExitEpoch != helpers.ActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.ActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}

func TestVerifyAttestations_VerifiesMultipleAttestations(t *testing.T) {
	ctx := context.Background()
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	comm1, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 0 /*committeeIndex*/)
	if err != nil {
		t.Fatal(err)
	}
	att1 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm1))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		Signature: nil,
	}
	domain, err := helpers.Domain(st.Fork(), st.Fork().Epoch, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	root, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		t.Fatal(err)
	}
	var sigs []bls.Signature
	for i, u := range comm1 {
		att1.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()

	comm2, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 1 /*committeeIndex*/)
	if err != nil {
		t.Fatal(err)
	}
	att2 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm2))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 1,
		},
		Signature: nil,
	}
	root, err = helpers.ComputeSigningRoot(att2.Data, domain)
	if err != nil {
		t.Fatal(err)
	}
	sigs = nil
	for i, u := range comm2 {
		att2.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()

	if err := blocks.VerifyAttestations(ctx, st, []*ethpb.Attestation{att1, att2}); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyAttestations_HandlesPlannedFork(t *testing.T) {
	// In this test, att1 is from the prior fork and att2 is from the new fork.
	ctx := context.Background()
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       35,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           1,
			CurrentVersion:  []byte{0, 1, 2, 3},
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	comm1, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 0 /*committeeIndex*/)
	if err != nil {
		t.Fatal(err)
	}
	att1 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm1))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		Signature: nil,
	}
	prevDomain, err := helpers.Domain(st.Fork(), st.Fork().Epoch-1, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	root, err := helpers.ComputeSigningRoot(att1.Data, prevDomain)
	if err != nil {
		t.Fatal(err)
	}
	var sigs []bls.Signature
	for i, u := range comm1 {
		att1.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()

	comm2, err := helpers.BeaconCommitteeFromState(st, 1*params.BeaconConfig().SlotsPerEpoch+1 /*slot*/, 1 /*committeeIndex*/)
	if err != nil {
		t.Fatal(err)
	}
	att2 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm2))),
		Data: &ethpb.AttestationData{
			Slot:           1*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex: 1,
		},
		Signature: nil,
	}
	currDomain, err := helpers.Domain(st.Fork(), st.Fork().Epoch, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	root, err = helpers.ComputeSigningRoot(att2.Data, currDomain)
	if err != nil {
		t.Fatal(err)
	}
	sigs = nil
	for i, u := range comm2 {
		att2.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()

	if err := blocks.VerifyAttestations(ctx, st, []*ethpb.Attestation{att1, att2}); err != nil {
		t.Fatal(err)
	}
}

func TestAreEth1DataEqual(t *testing.T) {
	type args struct {
		a *ethpb.Eth1Data
		b *ethpb.Eth1Data
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true when both are nil",
			args: args{
				a: nil,
				b: nil,
			},
			want: true,
		},
		{
			name: "false when only one is nil",
			args: args{
				a: nil,
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
			},
			want: false,
		},
		{
			name: "true when real equality",
			args: args{
				a: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
			},
			want: true,
		},
		{
			name: "false is field value differs",
			args: args{
				a: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 0,
					BlockHash:    make([]byte, 32),
				},
				b: &ethpb.Eth1Data{
					DepositRoot:  make([]byte, 32),
					DepositCount: 64,
					BlockHash:    make([]byte, 32),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blocks.AreEth1DataEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("AreEth1DataEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
