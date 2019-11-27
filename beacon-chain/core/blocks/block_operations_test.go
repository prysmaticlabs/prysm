package blocks_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	blsintern "github.com/phoreproject/bls"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Error(err)
	}
	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{Slot: 9}

	lbhsr, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}

	block := &ethpb.BeaconBlock{
		Slot: 0,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: lbhsr[:],
	}
	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	dt := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconProposer)
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

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	lbhsr, err := ssz.HashTreeRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.BeaconBlock{
		Slot: 1,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: lbhsr[:],
		Signature:  blockSig.Marshal(),
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

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5896].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.BeaconBlock{
		Slot: 0,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: []byte{'A'},
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
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

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	parentRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[12683].PublicKey = priv.PublicKey().Marshal()
	block := &ethpb.BeaconBlock{
		Slot: 0,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: parentRoot[:],
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "was previously slashed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_OK(t *testing.T) {
	helpers.ClearAllCaches()

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &ethpb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	latestBlockSignedRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state.Fork, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key got: %v", err)
	}
	block := &ethpb.BeaconBlock{
		Slot: 0,
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: latestBlockSignedRoot[:],
	}
	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Fatalf("Failed to get signing root of block: %v", err)
	}
	blockSig := priv.Sign(signingRoot[:], dt)
	block.Signature = blockSig.Marshal()[:]
	bodyRoot, err := ssz.HashTreeRoot(block.Body)
	if err != nil {
		t.Fatalf("Failed to hash block bytes got: %v", err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Fatal(err)
	}
	validators[proposerIdx].Slashed = false
	validators[proposerIdx].PublicKey = priv.PublicKey().Marshal()

	newState, err := blocks.ProcessBlockHeader(state, block)
	if err != nil {
		t.Fatalf("Failed to process block header got: %v", err)
	}
	var zeroHash [32]byte
	var zeroSig [96]byte
	nsh := newState.LatestBlockHeader
	expected := &ethpb.BeaconBlockHeader{
		Slot:       block.Slot,
		ParentRoot: latestBlockSignedRoot[:],
		BodyRoot:   bodyRoot[:],
		StateRoot:  zeroHash[:],
		Signature:  zeroSig[:],
	}
	if !proto.Equal(nsh, expected) {
		t.Errorf("Expected %v, received %vk9k", expected, nsh)
	}
}

func TestProcessRandao_IncorrectProposerFailsVerification(t *testing.T) {
	helpers.ClearAllCaches()

	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.Domain(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)

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
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	epoch := helpers.CurrentEpoch(beaconState)
	epochSignature, err := testutil.CreateRandaoReveal(beaconState, epoch, privKeys)
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
	mix := newState.RandaoMixes[currentEpoch%params.BeaconConfig().EpochsPerHistoricalVector]

	if bytes.Equal(mix, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf(
			"Expected empty signature to be overwritten by randao reveal, received %v",
			params.BeaconConfig().EmptySignature,
		)
	}
}

func TestProcessEth1Data_SetsCorrectly(t *testing.T) {
	beaconState := &pb.BeaconState{
		Eth1DataVotes: []*ethpb.Eth1Data{},
	}

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

	newETH1DataVotes := beaconState.Eth1DataVotes
	if len(newETH1DataVotes) <= 1 {
		t.Error("Expected new ETH1 data votes to have length > 1")
	}
	if !proto.Equal(beaconState.Eth1Data, block.Body.Eth1Data) {
		t.Errorf(
			"Expected latest eth1 data to have been set to %v, received %v",
			block.Body.Eth1Data,
			beaconState.Eth1Data,
		)
	}
}
func TestProcessProposerSlashings_UnmatchedHeaderSlots(t *testing.T) {
	registry := make([]*ethpb.Validator, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &ethpb.BeaconBlockHeader{
				Slot: params.BeaconConfig().SlotsPerEpoch + 1,
			},
			Header_2: &ethpb.BeaconBlockHeader{
				Slot: 0,
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
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
	registry := make([]*ethpb.Validator, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &ethpb.BeaconBlockHeader{
				Slot: 0,
			},
			Header_2: &ethpb.BeaconBlockHeader{
				Slot: 0,
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
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
			Header_1: &ethpb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("A"),
			},
			Header_2: &ethpb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("B"),
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		beaconState.Validators[0].PublicKey,
	)

	if _, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	validators := make([]*ethpb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
			Slashed:           false,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:   0,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	currentSlot := uint64(0)
	beaconState := &pb.BeaconState{
		Validators: validators,
		Slot:       currentSlot,
		Balances:   validatorBalances,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			Epoch:           0,
		},
		Slashings:   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	domain := helpers.Domain(
		beaconState.Fork,
		helpers.CurrentEpoch(beaconState),
		params.BeaconConfig().DomainBeaconProposer,
	)
	privKey, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("Could not generate random private key: %v", err)
	}

	header1 := &ethpb.BeaconBlockHeader{
		Slot:      0,
		StateRoot: []byte("A"),
	}
	signingRoot, err := ssz.SigningRoot(header1)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKey.Sign(signingRoot[:], domain).Marshal()[:]

	header2 := &ethpb.BeaconBlockHeader{
		Slot:      0,
		StateRoot: []byte("B"),
	}
	signingRoot, err = ssz.SigningRoot(header2)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKey.Sign(signingRoot[:], domain).Marshal()[:]

	slashings := []*ethpb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1:      header1,
			Header_2:      header2,
		},
	}

	beaconState.Validators[1].PublicKey = privKey.PublicKey().Marshal()[:]

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	newStateVals := newState.Validators
	if newStateVals[1].ExitEpoch != validators[1].ExitEpoch {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			newStateVals[1].ExitEpoch, validators[1].ExitEpoch)
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

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
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
	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: []uint64{0, 1, 2},
				CustodyBit_1Indices: []uint64{0, 1, 2},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: []uint64{0, 1, 2},
				CustodyBit_1Indices: []uint64{0, 1, 2},
			},
		},
	}
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("expected no bit 1 indices")

	if _, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	slashings = []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
		},
	}

	block.Body.AttesterSlashings = slashings
	want = fmt.Sprint("over max number of allowed indices")

	if _, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	for _, vv := range beaconState.Validators {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	sig0 := privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 := privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig := bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit = &pb.AttestationDataAndCustodyBit{
		Data:       att2.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit)
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
	beaconState.Slot = currentSlot

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.Validators

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be slashed and exited. We confirm this below.
	if newRegistry[1].ExitEpoch != beaconState.Validators[1].ExitEpoch {
		t.Errorf(
			`
			Expected validator at index 1's exit epoch to match
			%d, received %d instead
			`,
			beaconState.Validators[1].ExitEpoch,
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
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(
		"attestation slot %d + inclusion delay %d > state slot %d",
		attestations[0].Data.Slot,
		params.BeaconConfig().MinAttestationInclusionDelay,
		beaconState.Slot,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	helpers.ClearActiveIndicesCache()
	helpers.ClearActiveCountCache()

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}}}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	helpers.ClearAllCaches()
	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.PreviousJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.PreviousEpochAttestations = []*pb.PendingAttestation{}

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
	helpers.ClearAllCaches()

	aggBits := bitfield.NewBitlist(3)
	custodyBits := bitfield.NewBitlist(3)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0},
				Source: &ethpb.Checkpoint{Epoch: 1},
			},
			AggregationBits: aggBits,
			CustodyBits:     custodyBits,
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

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
		beaconState.CurrentJustifiedCheckpoint.Root,
		attestations[0].Data.Source.Root,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	helpers.ClearAllCaches()

	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	custodyBits := bitfield.NewBitlist(3)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{Epoch: 1},
				Slot:   1,
			},
			AggregationBits: aggBits,
			CustodyBits:     custodyBits,
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	helpers.ClearAllCaches()

	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.PreviousJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.PreviousEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
	helpers.ClearAllCaches()

	block.Body.Attestations[0].Data.Source.Epoch = helpers.PrevEpoch(beaconState)
	block.Body.Attestations[0].Data.Target.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint.Root,
		attestations[0].Data.Source.Root,
	)
	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_InvalidAggregationBitsLength(t *testing.T) {
	helpers.ClearAllCaches()
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(4)
	custodyBits := bitfield.NewBitlist(4)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	expected := "failed to verify aggregation bitfield: wanted participants bitfield length 3, got: 4"
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Did not receive wanted error")
	}
}

func TestProcessAttestations_OK(t *testing.T) {
	helpers.ClearAllCaches()
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	custodyBits := bitfield.NewBitlist(3)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
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

	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestProcessAggregatedAttestation_OverlappingBits(t *testing.T) {
	helpers.ClearAllCaches()
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 300)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
	}
	aggBits1 := bitfield.NewBitlist(4)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	aggBits1.SetBitAt(2, true)
	custodyBits1 := bitfield.NewBitlist(4)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
		CustodyBits:     custodyBits1,
	}

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	attestingIndices1, err := helpers.AttestingIndices(beaconState, att1.Data, att1.AggregationBits)
	if err != nil {
		t.Fatal(err)
	}
	dataAndCustodyBit1 := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit1)
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
	custodyBits2 := bitfield.NewBitlist(4)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
		CustodyBits:     custodyBits2,
	}

	attestingIndices2, err := helpers.AttestingIndices(beaconState, att2.Data, att2.AggregationBits)
	if err != nil {
		t.Fatal(err)
	}
	dataAndCustodyBit2 := &pb.AttestationDataAndCustodyBit{
		Data:       att2.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit2)
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
	helpers.ClearAllCaches()
	deposits, _, privKeys := testutil.SetupInitialDeposits(t, 300)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
	}
	aggBits1 := bitfield.NewBitlist(9)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	custodyBits1 := bitfield.NewBitlist(9)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
		CustodyBits:     custodyBits1,
	}

	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	attestingIndices1, err := helpers.AttestingIndices(beaconState, att1.Data, att1.AggregationBits)
	if err != nil {
		t.Fatal(err)
	}
	dataAndCustodyBit1 := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit1)
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
	custodyBits2 := bitfield.NewBitlist(9)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
		CustodyBits:     custodyBits2,
	}

	attestingIndices2, err := helpers.AttestingIndices(beaconState, att2.Data, att2.AggregationBits)
	if err != nil {
		t.Fatal(err)
	}
	dataAndCustodyBit2 := &pb.AttestationDataAndCustodyBit{
		Data:       att2.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit2)
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

	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay

	if _, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body); err != nil {
		t.Error(err)
	}
}

func TestProcessAttestationsNoVerify_OK(t *testing.T) {
	// Attestation with an empty signature
	helpers.ClearAllCaches()
	deposits, _, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(1, true)
	custodyBits := bitfield.NewBitlist(3)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: aggBits,
		CustodyBits:     custodyBits,
	}

	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	if _, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestConvertToIndexed_OK(t *testing.T) {
	helpers.ClearAllCaches()

	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Slot:        5,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	tests := []struct {
		aggregationBitfield      bitfield.Bitlist
		custodyBitfield          bitfield.Bitlist
		wantedCustodyBit0Indices []uint64
		wantedCustodyBit1Indices []uint64
	}{
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x05},
			wantedCustodyBit0Indices: []uint64{4},
			wantedCustodyBit1Indices: []uint64{30},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x06},
			wantedCustodyBit0Indices: []uint64{30},
			wantedCustodyBit1Indices: []uint64{4},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x07},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{4, 30},
		},
	}

	attestation := &ethpb.Attestation{
		Signature: []byte("signed"),
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
	}
	for _, tt := range tests {
		helpers.ClearAllCaches()

		attestation.AggregationBits = tt.aggregationBitfield
		attestation.CustodyBits = tt.custodyBitfield
		wanted := &ethpb.IndexedAttestation{
			CustodyBit_0Indices: tt.wantedCustodyBit0Indices,
			CustodyBit_1Indices: tt.wantedCustodyBit1Indices,
			Data:                attestation.Data,
			Signature:           attestation.Signature,
		}
		ia, err := blocks.ConvertToIndexed(context.Background(), state, attestation)
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
	helpers.ClearAllCaches()

	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, _, keys := testutil.SetupInitialDeposits(t, numOfValidators)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	state := &pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
			CustodyBit_0Indices: []uint64{1},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 1,
				},
			},
			CustodyBit_0Indices: []uint64{47, 99},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 4,
				},
			},
			CustodyBit_0Indices: []uint64{21, 72},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 7,
				},
			},
			CustodyBit_0Indices: []uint64{100, 121},
		}},
	}

	for _, tt := range tests {
		helpers.ClearAllCaches()

		attDataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
			Data:       tt.attestation.Data,
			CustodyBit: false,
		}

		domain := helpers.Domain(state.Fork, tt.attestation.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester)

		root, err := ssz.HashTreeRoot(attDataAndCustodyBit)
		if err != nil {
			t.Errorf("Could not find the ssz root: %v", err)
			continue
		}
		var sig []*bls.Signature
		for _, idx := range tt.attestation.CustodyBit_0Indices {
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
		CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+5),
		CustodyBit_1Indices: []uint64{},
	}

	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee+5; i++ {
		indexedAtt1.CustodyBit_0Indices[i] = i
	}

	want := "over max number of allowed indices"
	if err := blocks.VerifyIndexedAttestation(context.Background(), &pb.BeaconState{}, indexedAtt1); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected verification to fail return false, received: %v", err)
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
	beaconState := &pb.BeaconState{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	}
	want := "deposit root did not verify"
	if _, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessDeposits_AddsNewValidatorDeposit(t *testing.T) {
	dep, _, _ := testutil.SetupInitialDeposits(t, 1)
	eth1Data := testutil.GenerateEth1Data(t, dep)

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
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.Balances[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances[1],
		)
	}
}

func TestProcessDeposits_RepeatedDeposit_IncreasesValidatorBalance(t *testing.T) {
	sk, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey: sk.PublicKey().Marshal(),
			Amount:    1000,
		},
	}
	sr, err := ssz.SigningRoot(deposit.Data)
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
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: root[:],
			BlockHash:   root[:],
		},
	}
	newState, err := blocks.ProcessDeposits(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.Balances[1] != 1000+50 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.Balances[1])
	}
}

func TestProcessDeposit_AddsNewValidatorDeposit(t *testing.T) {
	//Similar to TestProcessDeposits_AddsNewValidatorDeposit except that this test directly calls ProcessDeposit
	dep, _, _ := testutil.SetupInitialDeposits(t, 1)
	eth1Data := testutil.GenerateEth1Data(t, dep)

	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	}
	newState, err := blocks.ProcessDeposit(
		beaconState,
		dep[0],
		stateutils.ValidatorIndexMap(beaconState),
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.Validators) != 2 {
		t.Errorf("Expected validator list to have length 2, received: %v", len(newState.Validators))
	}
	if len(newState.Balances) != 2 {
		t.Fatalf("Expected validator balances list to have length 2, received: %v", len(newState.Balances))
	}
	if newState.Balances[1] != dep[0].Data.Amount {
		t.Errorf(
			"Expected state validator balances index 1 to equal %d, received %d",
			dep[0].Data.Amount,
			newState.Balances[1],
		)
	}
}

func TestProcessDeposit_SkipsInvalidDeposit(t *testing.T) {
	// Same test settings as in TestProcessDeposit_AddsNewValidatorDeposit, except that we use an invalid signature
	dep, _, _ := testutil.SetupInitialDeposits(t, 1)
	dep[0].Data.Signature = make([]byte, 96)
	eth1Data := testutil.GenerateEth1Data(t, dep)
	testutil.ResetCache() // Can't have an invalid signature in the cache.

	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	}
	newState, err := blocks.ProcessDeposit(
		beaconState,
		dep[0],
		stateutils.ValidatorIndexMap(beaconState),
	)

	if err != nil {
		t.Fatalf("Expected invalid block deposit to be ignored without error, received: %v", err)
	}
	if newState.Eth1DepositIndex != 1 {
		t.Errorf(
			"Expected Eth1DepositIndex to be increased by 1 after processing an invalid deposit, received change: %v",
			newState.Eth1DepositIndex,
		)
	}
	if len(newState.Validators) != 1 {
		t.Errorf("Expected validator list to have length 1, received: %v", len(newState.Validators))
	}
	if len(newState.Balances) != 1 {
		t.Errorf("Expected validator balances list to have length 1, received: %v", len(newState.Balances))
	}
	if newState.Balances[0] != 0 {
		t.Errorf("Expected validator balance at index 0 to stay 0, received: %v", newState.Balances[0])
	}
}

func TestProcessDeposit_SkipsDepositWithUncompressedSignature(t *testing.T) {
	// Same test settings as in TestProcessDeposit_AddsNewValidatorDeposit, except that we use an uncompressed signature
	dep, _, _ := testutil.SetupInitialDeposits(t, 1)
	a, _ := blsintern.DecompressG2(bytesutil.ToBytes96(dep[0].Data.Signature))
	uncompressedSignature := a.SerializeBytes()
	dep[0].Data.Signature = uncompressedSignature[:]
	eth1Data := testutil.GenerateEth1Data(t, dep)
	testutil.ResetCache() // Can't have an uncompressed signature in the cache.

	registry := []*ethpb.Validator{
		{
			PublicKey:             []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data:   eth1Data,
		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		},
	}
	newState, err := blocks.ProcessDeposit(
		beaconState,
		dep[0],
		stateutils.ValidatorIndexMap(beaconState),
	)

	if err != nil {
		t.Fatalf("Expected invalid block deposit to be ignored without error, received: %v", err)
	}
	if newState.Eth1DepositIndex != 1 {
		t.Errorf(
			"Expected Eth1DepositIndex to be increased by 1 after processing an invalid deposit, received change: %v",
			newState.Eth1DepositIndex,
		)
	}
	if len(newState.Validators) != 1 {
		t.Errorf("Expected validator list to have length 1, received: %v", len(newState.Validators))
	}
	if len(newState.Balances) != 1 {
		t.Errorf("Expected validator balances list to have length 1, received: %v", len(newState.Balances))
	}
	if newState.Balances[0] != 0 {
		t.Errorf("Expected validator balance at index 0 to stay 0, received: %v", newState.Balances[0])
	}
}

func TestProcessVoluntaryExits_ValidatorNotActive(t *testing.T) {
	exits := []*ethpb.VoluntaryExit{
		{
			ValidatorIndex: 0,
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: 0,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
	}
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
	exits := []*ethpb.VoluntaryExit{
		{
			Epoch: 10,
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Slot:       0,
	}
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
	exits := []*ethpb.VoluntaryExit{
		{
			ValidatorIndex: 0,
			Epoch:          0,
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Slot:       10,
	}
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
	exits := []*ethpb.VoluntaryExit{
		{
			ValidatorIndex: 0,
			Epoch:          0,
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	}
	state.Slot = state.Slot + (params.BeaconConfig().PersistentCommitteePeriod * params.BeaconConfig().SlotsPerEpoch)

	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
	state.Validators[0].PublicKey = priv.PublicKey().Marshal()[:]
	signingRoot, err := ssz.SigningRoot(exits[0])
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(state.Fork, helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit)
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
	newRegistry := newState.Validators
	if newRegistry[0].ExitEpoch != helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}
