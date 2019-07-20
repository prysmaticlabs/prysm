package blocks_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
	t.Skip("Skip until bls.Verify is finished")
	// TODO(#2307) unskip after bls.Verify is finished
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	validators[5896].Slashed = false

	lbhsr, err := ssz.HashTreeRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	priv2, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	wrongBlockSig := priv2.Sign([]byte("hello"), dt)
	validators[5896].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: lbhsr[:],
		Signature:  wrongBlockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block, false)
	want := "verify signature failed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}

}

func TestProcessBlockHeader_DifferentSlots(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	lbhsr, err := ssz.HashTreeRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5896].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 1,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: lbhsr[:],
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block, false)
	want := "is different then block slot"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_PreviousBlockRootNotSignedRoot(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5896].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: []byte{'A'},
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block, false)
	want := "does not match"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_SlashedProposer(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	parentRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Errorf("failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[12683].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: parentRoot[:],
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block, false)
	want := "was previously slashed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_OK(t *testing.T) {
	helpers.ClearAllCaches()

	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Fatalf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		Validators:        validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	validators[63463].Slashed = false

	latestBlockSignedRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.Domain(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[6033].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: latestBlockSignedRoot[:],
		Signature:  blockSig.Marshal(),
	}
	bodyRoot, err := ssz.HashTreeRoot(block.Body)
	if err != nil {
		t.Fatalf("Failed to hash block bytes got: %v", err)
	}
	newState, err := blocks.ProcessBlockHeader(state, block, false)
	if err != nil {
		t.Fatalf("Failed to process block header got: %v", err)
	}
	var zeroHash [32]byte
	var zeroSig [96]byte
	nsh := newState.LatestBlockHeader
	expected := &pb.BeaconBlockHeader{
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

	deposits, privKeys := testutil.SetupInitialDeposits(t, 100, true)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
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
	domain := helpers.Domain(beaconState, epoch, params.BeaconConfig().DomainRandao)

	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(buf, domain)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	want := "block randao reveal signature did not verify"
	if _, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
		true, /* verify signatures */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100, true)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}

	epoch := helpers.CurrentEpoch(beaconState)
	epochSignature, err := helpers.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	newState, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
		true, /* verify signatures */
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
		Eth1DataVotes: []*pb.Eth1Data{},
	}

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Eth1Data: &pb.Eth1Data{
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
func TestProcessProposerSlashings_UnmatchedHeaderEpochs(t *testing.T) {
	registry := make([]*pb.Validator, 2)
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &pb.BeaconBlockHeader{
				Slot: params.BeaconConfig().SlotsPerEpoch + 1,
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot: 0,
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header epochs"
	if _, err := blocks.ProcessProposerSlashings(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_SameHeaders(t *testing.T) {
	registry := make([]*pb.Validator, 2)
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &pb.BeaconBlockHeader{
				Slot: 0,
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot: 0,
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	if _, err := blocks.ProcessProposerSlashings(

		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_ValidatorNotSlashable(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey:            []byte("key"),
			Slashed:           true,
			ActivationEpoch:   0,
			WithdrawableEpoch: 0,
		},
	}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("B"),
			},
		},
	}

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		beaconState.Validators[0].Pubkey,
	)

	if _, err := blocks.ProcessProposerSlashings(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	helpers.ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
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

	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("B"),
			},
		},
	}
	currentSlot := uint64(0)
	beaconState := &pb.BeaconState{
		Validators:       validators,
		Slot:             currentSlot,
		Balances:         validatorBalances,
		Slashings:        make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessProposerSlashings(
		beaconState,
		block.Body,
		false,
	)
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
	att1 := &pb.AttestationData{
		Target: &pb.Checkpoint{Epoch: 1},
		Source: &pb.Checkpoint{Root: []byte{'A'}},
	}
	att2 := &pb.AttestationData{
		Target: &pb.Checkpoint{Epoch: 1},
		Source: &pb.Checkpoint{Root: []byte{'B'}},
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
	slashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 1},
					Crosslink: &pb.Crosslink{
						Shard: 3,
					},
				},
			},
		},
	}
	registry := []*pb.Validator{}
	currentSlot := uint64(0)

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("attestations are not slashable")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_IndexedAttestationFailedToVerify(t *testing.T) {
	slashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1, 2},
				CustodyBit_1Indices: []uint64{0, 1, 2},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1, 2},
				CustodyBit_1Indices: []uint64{0, 1, 2},
			},
		},
	}
	registry := []*pb.Validator{}
	currentSlot := uint64(0)

	beaconState := &pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("expected no bit 1 indices")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	slashings = []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
		},
	}

	block.Body.AttesterSlashings = slashings
	want = fmt.Sprint("over max number of allowed indices")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ActivationEpoch:   0,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			Slashed:           false,
			WithdrawableEpoch: 1 * params.BeaconConfig().SlotsPerEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	slashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	beaconState := &pb.BeaconState{
		Validators:       validators,
		Slot:             currentSlot,
		Balances:         validatorBalances,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:        make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	newState, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block.Body,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.Validators

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be slashed and exited. We confirm this below.
	if newRegistry[1].ExitEpoch != validators[1].ExitEpoch {
		t.Errorf(
			`
			Expected validator at index 1's exit epoch to match
			%d, received %d instead
			`,
			validators[1].ExitEpoch,
			newRegistry[1].ExitEpoch,
		)
	}
}

func TestProcessAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Target: &pb.Checkpoint{Epoch: 0},
				Crosslink: &pb.Crosslink{
					Shard: 0,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, 100, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}

	attestationSlot, err := helpers.AttestationDataSlot(beaconState, attestations[0].Data)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf(
		"attestation slot %d + inclusion delay %d > state slot %d",
		attestationSlot,
		params.BeaconConfig().MinAttestationInclusionDelay,
		beaconState.Slot,
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	helpers.ClearActiveIndicesCache()
	helpers.ClearActiveCountCache()
	helpers.ClearStartShardCache()

	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Source: &pb.Checkpoint{Epoch: 0},
				Target: &pb.Checkpoint{Epoch: 0},
				Crosslink: &pb.Crosslink{
					Shard: 0,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, 100, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	helpers.ClearAllCaches()
	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay

	want := fmt.Sprintf(
		"expected target epoch %d == %d or %d",
		attestations[0].Data.Target.Epoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_CurrentEpochFFGDataMismatches(t *testing.T) {
	helpers.ClearAllCaches()

	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Target: &pb.Checkpoint{Epoch: 0},
				Source: &pb.Checkpoint{Epoch: 1},
				Crosslink: &pb.Crosslink{
					Shard: 0,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, 100, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
		},
	}
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.CurrentEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Source.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint.Root,
		attestations[0].Data.Source.Root,
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	helpers.ClearAllCaches()

	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Source: &pb.Checkpoint{Epoch: 1},
				Target: &pb.Checkpoint{Epoch: 0},
				Crosslink: &pb.Crosslink{
					Shard: 0,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, 100, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	helpers.ClearAllCaches()
	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.PreviousCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
		},
	}
	beaconState.PreviousJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.PreviousEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Source.Epoch = helpers.PrevEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.PreviousJustifiedCheckpoint.Root,
		attestations[0].Data.Source.Root,
	)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_CrosslinkMismatches(t *testing.T) {
	helpers.ClearAllCaches()

	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Source: &pb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
				Target: &pb.Checkpoint{Epoch: 0},
				Crosslink: &pb.Crosslink{
					Shard: 0,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, 100, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard:      0,
			StartEpoch: 0,
		},
	}
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	want := "mismatched parent crosslink root"
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Crosslink.StartEpoch = 0
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
	encoded, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = encoded[:]

	want = fmt.Sprintf("expected data root %#x == ZERO_HASH", encoded)
	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttestations_OK(t *testing.T) {
	helpers.ClearAllCaches()

	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Source: &pb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
				Target: &pb.Checkpoint{Epoch: 0},
				Crosslink: &pb.Crosslink{
					Shard:      0,
					StartEpoch: 0,
				},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := testutil.SetupInitialDeposits(t, params.BeaconConfig().MinGenesisActiveValidatorCount/8, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard:      0,
			StartEpoch: 0,
		},
	}
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	encoded, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]

	if _, err := blocks.ProcessAttestations(
		beaconState,
		block.Body,
		false,
	); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestConvertToIndexed_OK(t *testing.T) {
	helpers.ClearActiveIndicesCache()
	helpers.ClearActiveCountCache()
	helpers.ClearStartShardCache()
	helpers.ClearShuffledValidatorCache()

	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Slot:             5,
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
			wantedCustodyBit0Indices: []uint64{71},
			wantedCustodyBit1Indices: []uint64{127},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x06},
			wantedCustodyBit0Indices: []uint64{127},
			wantedCustodyBit1Indices: []uint64{71},
		},
		{
			aggregationBitfield:      bitfield.Bitlist{0x07},
			custodyBitfield:          bitfield.Bitlist{0x07},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{71, 127},
		},
	}

	attestation := &pb.Attestation{
		Signature: []byte("signed"),
		Data: &pb.AttestationData{
			Source: &pb.Checkpoint{Epoch: 0},
			Target: &pb.Checkpoint{Epoch: 0},
			Crosslink: &pb.Crosslink{
				Shard: 3,
			},
		},
	}
	for _, tt := range tests {
		helpers.ClearAllCaches()

		attestation.AggregationBits = tt.aggregationBitfield
		attestation.CustodyBits = tt.custodyBitfield
		wanted := &pb.IndexedAttestation{
			CustodyBit_0Indices: tt.wantedCustodyBit0Indices,
			CustodyBit_1Indices: tt.wantedCustodyBit1Indices,
			Data:                attestation.Data,
			Signature:           attestation.Signature,
		}
		ia, err := blocks.ConvertToIndexed(state, attestation)
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

func TestValidateIndexedAttestation_AboveMaxLength(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+5),
		CustodyBit_1Indices: []uint64{},
	}

	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee+5; i++ {
		indexedAtt1.CustodyBit_0Indices[i] = i
	}

	want := "over max number of allowed indices"
	if err := blocks.VerifyIndexedAttestation(
		&pb.BeaconState{},
		indexedAtt1,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected verification to fail return false, received: %v", err)
	}
}

func TestProcessDeposits_MerkleBranchFailsVerification(t *testing.T) {
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey:    []byte{1, 2, 3},
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
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	}
	want := "deposit root did not verify"
	if _, err := blocks.ProcessDeposits(
		beaconState,
		block.Body,
		false, /* verifySignatures */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessDeposits_ProcessCorrectly(t *testing.T) {
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey:    []byte{1, 2, 3},
			Amount:    params.BeaconConfig().MaxEffectiveBalance,
			Signature: make([]byte, 96),
		},
	}
	leaf, err := hashutil.DepositHash(deposit.Data)
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
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.Validator{
		{
			Pubkey:                []byte{1},
			WithdrawalCredentials: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
		Eth1Data: &pb.Eth1Data{
			DepositRoot: root[:],
			BlockHash:   root[:],
		},
	}
	newState, err := blocks.ProcessDeposits(
		beaconState,
		block.Body,
		false, /* verifySignatures */
	)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.Balances[1] != deposit.Data.Amount {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			deposit.Data.Amount,
			newState.Balances[0],
		)
	}
}

func TestProcessDeposit_RepeatedDeposit(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{0, 50}
	beaconState := &pb.BeaconState{
		Balances:   balances,
		Validators: registry,
	}

	deposit := &pb.Deposit{
		Proof: [][]byte{},
		Data: &pb.DepositData{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
			Amount:                uint64(1000),
		},
	}

	newState, err := blocks.ProcessDeposit(
		beaconState,
		deposit,
		stateutils.ValidatorIndexMap(beaconState),
		false,
		false,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.Balances[1] != 1050 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.Balances[1])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExist(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey:                []byte{1, 2, 3},
			WithdrawalCredentials: []byte{2},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{1000, 1000}
	beaconState := &pb.BeaconState{
		Balances:   balances,
		Validators: registry,
	}

	deposit := &pb.Deposit{
		Proof: [][]byte{},
		Data: &pb.DepositData{
			Pubkey:                []byte{7, 8, 9},
			WithdrawalCredentials: []byte{1},
			Amount:                uint64(2000),
		},
	}

	newState, err := blocks.ProcessDeposit(
		beaconState,
		deposit,
		stateutils.ValidatorIndexMap(beaconState),
		false,
		false,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.Balances) != 3 {
		t.Errorf("Expected validator balances list to increase by 1, received len %d", len(newState.Balances))
	}
	if newState.Balances[2] != 2000 {
		t.Errorf("Expected new validator have balance of %d, received %d", 2000, newState.Balances[2])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistAndEmptyValidator(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey:                []byte{1, 2, 3},
			WithdrawalCredentials: []byte{2},
		},
		{
			Pubkey:                []byte{4, 5, 6},
			WithdrawalCredentials: []byte{1},
		},
	}
	balances := []uint64{0, 1000}
	beaconState := &pb.BeaconState{
		Slot:       params.BeaconConfig().SlotsPerEpoch,
		Balances:   balances,
		Validators: registry,
	}

	deposit := &pb.Deposit{
		Proof: [][]byte{},
		Data: &pb.DepositData{
			Pubkey:                []byte{7, 8, 9},
			WithdrawalCredentials: []byte{1},
			Amount:                uint64(2000),
		},
	}

	newState, err := blocks.ProcessDeposit(
		beaconState,
		deposit,
		stateutils.ValidatorIndexMap(beaconState),
		false,
		false,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.Balances) != 3 {
		t.Errorf("Expected validator balances list to be 3, received len %d", len(newState.Balances))
	}
	if newState.Balances[len(newState.Balances)-1] != 2000 {
		t.Errorf("Expected validator at last index to have balance of %d, received %d", 2000, newState.Balances[0])
	}
}

func TestProcessVoluntaryExits_ValidatorNotActive(t *testing.T) {
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 0,
		},
	}
	registry := []*pb.Validator{
		{
			ExitEpoch: 0,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "non-active validator cannot exit"

	if _, err := blocks.ProcessVoluntaryExits(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_InvalidExitEpoch(t *testing.T) {
	exits := []*pb.VoluntaryExit{
		{
			Epoch: 10,
		},
	}
	registry := []*pb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Slot:       0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "expected current epoch >= exit epoch"

	if _, err := blocks.ProcessVoluntaryExits(

		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_NotActiveLongEnoughToExit(t *testing.T) {
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 0,
			Epoch:          0,
		},
	}
	registry := []*pb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Slot:       10,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "validator has not been active long enough to exit"
	if _, err := blocks.ProcessVoluntaryExits(

		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_AppliesCorrectStatus(t *testing.T) {
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 0,
			Epoch:          0,
		},
	}
	registry := []*pb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Slot:       params.BeaconConfig().SlotsPerEpoch * 5,
	}
	state.Slot = state.Slot + (params.BeaconConfig().PersistentCommitteePeriod * params.BeaconConfig().SlotsPerEpoch)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}
	newState, err := blocks.ProcessVoluntaryExits(state, block.Body, false)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.Validators
	if newRegistry[0].ExitEpoch != helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}

func TestProcessBeaconTransfers_NotEnoughSenderBalance(t *testing.T) {
	registry := []*pb.Validator{
		{
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	state := &pb.BeaconState{
		Validators: registry,
		Balances:   balances,
	}
	transfers := []*pb.Transfer{
		{
			Fee:    params.BeaconConfig().MaxEffectiveBalance,
			Amount: params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: transfers,
		},
	}
	want := fmt.Sprintf(
		"expected sender balance %d >= %d",
		balances[0],
		transfers[0].Fee+transfers[0].Amount,
	)
	if _, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBeaconTransfers_FailsVerification(t *testing.T) {
	testConfig := params.BeaconConfig()
	testConfig.MaxTransfers = 1
	params.OverrideBeaconConfig(testConfig)
	registry := []*pb.Validator{
		{
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		},
		{
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	balances := []uint64{params.BeaconConfig().MaxEffectiveBalance}
	state := &pb.BeaconState{
		Slot:       0,
		Validators: registry,
		Balances:   balances,
	}
	transfers := []*pb.Transfer{
		{
			Fee: params.BeaconConfig().MaxEffectiveBalance + 1,
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: transfers,
		},
	}
	want := fmt.Sprintf(
		"expected sender balance %d >= %d",
		balances[0],
		transfers[0].Fee,
	)
	if _, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Transfers = []*pb.Transfer{
		{
			Fee:  params.BeaconConfig().MinDepositAmount,
			Slot: state.Slot + 1,
		},
	}
	want = fmt.Sprintf(
		"expected beacon state slot %d == transfer slot %d",
		state.Slot,
		block.Body.Transfers[0].Slot,
	)
	if _, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	state.Validators[0].WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
	state.Validators[0].ActivationEligibilityEpoch = 0
	state.Balances[0] = params.BeaconConfig().MinDepositAmount + params.BeaconConfig().MaxEffectiveBalance
	block.Body.Transfers = []*pb.Transfer{
		{
			Fee:    params.BeaconConfig().MinDepositAmount,
			Amount: params.BeaconConfig().MaxEffectiveBalance,
			Slot:   state.Slot,
		},
	}
	want = "over max transfer"
	if _, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	state.Validators[0].WithdrawableEpoch = 0
	state.Validators[0].ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
	buf := []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("B")
	hashed := hashutil.Hash(pubKey)
	buf = append(buf, hashed[:]...)
	state.Validators[0].WithdrawalCredentials = buf
	block.Body.Transfers = []*pb.Transfer{
		{
			Fee:    params.BeaconConfig().MinDepositAmount,
			Amount: params.BeaconConfig().MinDepositAmount,
			Slot:   state.Slot,
			Pubkey: []byte("A"),
		},
	}
	want = "invalid public key"
	if _, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBeaconTransfers_OK(t *testing.T) {
	helpers.ClearShuffledValidatorCache()
	testConfig := params.BeaconConfig()
	testConfig.MaxTransfers = 1
	params.OverrideBeaconConfig(testConfig)
	validators := make([]*pb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/32)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ActivationEpoch:   0,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			Slashed:           false,
			WithdrawableEpoch: 0,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             0,
		Balances:         validatorBalances,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:        make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	transfers := []*pb.Transfer{
		{
			Sender:    0,
			Recipient: 1,
			Fee:       params.BeaconConfig().MinDepositAmount,
			Amount:    params.BeaconConfig().MinDepositAmount,
			Slot:      state.Slot,
			Pubkey:    []byte("A"),
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: transfers,
		},
	}
	buf := []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("A")
	hashed := hashutil.Hash(pubKey)

	buf = append(buf, hashed[:][1:]...)
	state.Validators[0].WithdrawalCredentials = buf
	state.Validators[0].ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
	newState, err := blocks.ProcessTransfers(
		state,
		block.Body,
		false,
	)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedRecipient := params.BeaconConfig().MaxEffectiveBalance + block.Body.Transfers[0].Amount
	if newState.Balances[1] != expectedRecipient {
		t.Errorf("Expected recipient balance %d, received %d", newState.Balances[1], expectedRecipient)
	}
	expectedSender := params.BeaconConfig().MaxEffectiveBalance - block.Body.Transfers[0].Amount - block.Body.Transfers[0].Fee
	if newState.Balances[0] != expectedSender {
		t.Errorf("Expected sender balance %d, received %d", newState.Balances[0], expectedSender)
	}
}
