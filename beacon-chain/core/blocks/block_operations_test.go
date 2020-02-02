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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
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
	beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{Slot: 9})

	lbhsr, err := ssz.HashTreeRoot(beaconState.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}

	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 0,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: lbhsr[:],
		},
	}
	signingRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	dt := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer)
	blockSig := privKeys[proposerIdx+1].Sign(signingRoot[:], dt)
	block.Signature = blockSig.Marshal()[:]

	_, err = blocks.ProcessBlockHeader(beaconState, block)
	want := "signature did not verify"
	if !strings.Contains(err.Error(), want) {
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

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	lbhsr, err := ssz.HashTreeRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv := bls.RandKey()
	blockSig := priv.Sign([]byte("hello"), dt)
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
	want := "is different then block slot"
	if !strings.Contains(err.Error(), want) {
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

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv := bls.RandKey()
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 0,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: []byte{'A'},
		},
		Signature: blockSig.Marshal(),
	}

	_, err := blocks.ProcessBlockHeader(state, block)
	want := "does not match"
	if !strings.Contains(err.Error(), want) {
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

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	parentRoot, err := ssz.HashTreeRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv := bls.RandKey()
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[12683].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 0,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: parentRoot[:],
		},
		Signature: blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "was previously slashed"
	if !strings.Contains(err.Error(), want) {
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

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	latestBlockSignedRoot, err := ssz.HashTreeRoot(state.LatestBlockHeader())
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork(), currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv := bls.RandKey()
	block := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot: 0,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: []byte{'A', 'B', 'C'},
			},
			ParentRoot: latestBlockSignedRoot[:],
		},
	}
	signingRoot, err := ssz.HashTreeRoot(block.Block)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := priv.Sign(signingRoot[:], dt)
	block.Signature = blockSig.Marshal()[:]
	bodyRoot, err := ssz.HashTreeRoot(block.Block.Body)
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
		Slot:       block.Block.Slot,
		ParentRoot: latestBlockSignedRoot[:],
		BodyRoot:   bodyRoot[:],
		StateRoot:  zeroHash[:],
	}
	if !proto.Equal(nsh, expected) {
		t.Errorf("Expected %v, received %v", expected, nsh)
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
	domain := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao)

	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(buf, domain)
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	want := "block randao: signature did not verify"
	if _, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
	); !strings.Contains(err.Error(), want) {
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
	})

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}
	var err error
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEth1VotingPeriod; i++ {
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
			ProposerIndex: 1,
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: params.BeaconConfig().SlotsPerEpoch + 1,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: 0,
				},
			},
		},
	}
	beaconState.SetSlot(currentSlot)

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header slots"
	if _, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_SameHeaders(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: 0,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: 0,
				},
			},
		},
	}

	beaconState.SetSlot(currentSlot)
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	if _, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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
			ProposerIndex: 0,
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: 0,
				},
				Signature: []byte("A"),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					Slot: 0,
				},
				Signature: []byte("B"),
			},
		},
	}

	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		beaconState.Validators()[0].PublicKey,
	)

	if _, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	proposerIdx := uint64(1)

	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer)
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:      0,
			StateRoot: []byte("A"),
		},
	}
	signingRoot, err := ssz.HashTreeRoot(header1.Header)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()[:]

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:      0,
			StateRoot: []byte("B"),
		},
	}
	signingRoot, err = ssz.HashTreeRoot(header2.Header)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()[:]

	slashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: proposerIdx,
			Header_1:      header1,
			Header_2:      header2,
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

	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("attestations are not slashable")

	if _, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_IndexedAttestationFailedToVerify(t *testing.T) {
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})

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
	if _, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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
	hashTreeRoot, err := ssz.HashTreeRoot(att1.Data)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
	sig0 := privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 := privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig := bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AttestingIndices: []uint64{0, 1},
	}
	hashTreeRoot, err = ssz.HashTreeRoot(att2.Data)
	if err != nil {
		t.Error(err)
	}
	sig0 = privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 = privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig = bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	beaconState.SetSlot(currentSlot)

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
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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
	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	beaconState.SetPreviousJustifiedCheckpoint(pfc)
	beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{})

	want := fmt.Sprintf(
		"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
		att.Data.Target.Epoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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
	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	beaconState.SetCurrentJustifiedCheckpoint(cfc)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.CurrentEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Source.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint().Root,
		attestations[0].Data.Source.Root,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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

	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	beaconState.SetPreviousJustifiedCheckpoint(pfc)
	beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{})

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
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

	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	beaconState.SetCurrentJustifiedCheckpoint(cfc)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	expected := "failed to verify aggregation bitfield: wanted participants bitfield length 3, got: 4"
	_, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if !strings.Contains(err.Error(), expected) {
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
	beaconState.SetCurrentJustifiedCheckpoint(cfc)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		t.Error(err)
	}
	hashTreeRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)

	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestProcessAggregatedAttestation_OverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
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
	beaconState.SetCurrentJustifiedCheckpoint(cfc)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices1, err := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := ssz.HashTreeRoot(att1.Data)
	if err != nil {
		t.Fatal(err)
	}
	sigs := make([]*bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
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
	attestingIndices2, err := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err = ssz.HashTreeRoot(data)
	if err != nil {
		t.Fatal(err)
	}
	sigs = make([]*bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	if _, err = helpers.AggregateAttestation(att1, att2); err != helpers.ErrAttestationAggregationBitsOverlap {
		t.Error("Did not receive wanted error")
	}
}

func TestProcessAggregatedAttestation_NoOverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 300)

	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
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
	beaconState.SetCurrentJustifiedCheckpoint(cfc)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices1, err := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := ssz.HashTreeRoot(data)
	if err != nil {
		t.Fatal(err)
	}
	sigs := make([]*bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
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
	attestingIndices2, err := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err = ssz.HashTreeRoot(data)
	if err != nil {
		t.Fatal(err)
	}
	sigs = make([]*bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggregatedAtt, err := helpers.AggregateAttestation(att1, att2)
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{aggregatedAtt},
		},
	}

	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)

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
	if _, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att); err.Error() != wanted {
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

	beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	ckp := beaconState.CurrentJustifiedCheckpoint()
	copy(ckp.Root, "hello-world")
	beaconState.SetCurrentJustifiedCheckpoint(ckp)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})

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

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:        5,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
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
		ia, err := attestationutil.ConvertToIndexed(context.Background(), attestation, committee)
		if err != nil {
			t.Errorf("failed to convert attestation to indexed attestation: %v", err)
		}
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
	_, keys, _ := testutil.DeterministicDepositsAndKeys(numOfValidators)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
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
		domain := helpers.Domain(state.Fork(), tt.attestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester)

		root, err := ssz.HashTreeRoot(tt.attestation.Data)
		if err != nil {
			t.Errorf("Could not find the ssz root: %v", err)
			continue
		}
		var sig []*bls.Signature
		for _, idx := range tt.attestation.AttestingIndices {
			validatorSig := keys[idx].Sign(root[:], domain)
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
	}

	want := "validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE"
	if err := blocks.VerifyIndexedAttestation(context.Background(), &stateTrie.BeaconState{}, indexedAtt1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected verification to fail return false, received: %v", err)
	}
}

func TestProcessDeposits_SameValidatorMultipleDepositsSameBlock(t *testing.T) {
	// Same validator created 3 valid deposits within the same block
	testutil.ResetCache()
	dep, _, _ := testutil.DeterministicDepositsAndKeysSameValidator(3)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	})
	want := "deposit root did not verify"
	if _, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessDeposits_AddsNewValidatorDeposit(t *testing.T) {
	dep, _, _ := testutil.DeterministicDepositsAndKeys(1)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body)
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
	sr, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		t.Fatal(err)
	}
	sig := sk.Sign(sr[:], 3)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: root[:],
			BlockHash:   root[:],
		},
	})
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.Balances()[1] != 1000+50 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.Balances()[1])
	}
}

func TestProcessDeposit_AddsNewValidatorDeposit(t *testing.T) {
	//Similar to TestProcessDeposits_AddsNewValidatorDeposit except that this test directly calls ProcessDeposit
	dep, _, _ := testutil.DeterministicDepositsAndKeys(1)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	newState, err := blocks.ProcessDeposit(
		beaconState,
		dep[0],
	)
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
	dep, _, _ := testutil.DeterministicDepositsAndKeys(1)
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
	beaconState, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	})
	newState, err := blocks.ProcessDeposit(
		beaconState,
		dep[0],
	)
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
	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
	})
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "non-active validator cannot exit"

	if _, err := blocks.ProcessVoluntaryExits(context.Background(), state, block.Body); !strings.Contains(err.Error(), want) {
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
	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       0,
	})
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "expected current epoch >= exit epoch"

	if _, err := blocks.ProcessVoluntaryExits(context.Background(), state, block.Body); !strings.Contains(err.Error(), want) {
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
	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       10,
	})
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "validator has not been active long enough to exit"
	if _, err := blocks.ProcessVoluntaryExits(context.Background(), state, block.Body); !strings.Contains(err.Error(), want) {
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
	state, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	})
	state.SetSlot(state.Slot() + (params.BeaconConfig().PersistentCommitteePeriod * params.BeaconConfig().SlotsPerEpoch))

	priv := bls.RandKey()
	val, err := state.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	val.PublicKey = priv.PublicKey().Marshal()[:]
	state.UpdateValidatorAtIndex(0, val)
	signingRoot, err := ssz.HashTreeRoot(exits[0].Exit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit)
	sig := priv.Sign(signingRoot[:], domain)
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
	if newRegistry[0].ExitEpoch != helpers.DelayedActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.DelayedActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}
