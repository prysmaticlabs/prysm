package blocks_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func setupInitialDeposits(t *testing.T, numDeposits int) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositData := &pb.DepositData{
			Pubkey: priv.PublicKey().Marshal(),
			Amount: params.BeaconConfig().MaxDepositAmount,
		}

		deposits[i] = &pb.Deposit{Data: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func TestProcessBlockHeader_WrongProposerSig(t *testing.T) {
	t.Skip()
	// TODO(#2307) unskip after bls.Verify is finished
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	validators[5896].Slashed = false

	lbhsr, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.DomainVersion(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
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

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "verify signature failed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}

}

func TestProcessBlockHeader_DifferentSlots(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	lbhsr, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.DomainVersion(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
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

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "is different then block slot"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_PreviousBlockRootNotSignedRoot(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.DomainVersion(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
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

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "does not match"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_SlashedProposer(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	lbhsr, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.DomainVersion(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
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
		ParentRoot: lbhsr[:],
		Signature:  blockSig.Marshal(),
	}

	_, err = blocks.ProcessBlockHeader(state, block)
	want := "was previously slashed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockHeader_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Fatalf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			Slashed:   true,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
		LatestBlockHeader: &pb.BeaconBlockHeader{Slot: 9},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	validators[5297].Slashed = false

	latestBlockSignedRoot, err := ssz.SigningRoot(state.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	dt := helpers.DomainVersion(state, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key got: %v", err)
	}
	blockSig := priv.Sign([]byte("hello"), dt)
	validators[5297].Pubkey = priv.PublicKey().Marshal()
	block := &pb.BeaconBlock{
		Slot: 0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
		},
		ParentRoot: latestBlockSignedRoot[:],
		Signature:  blockSig.Marshal(),
	}
	bodyRoot, err := ssz.TreeHash(block.Body)
	if err != nil {
		t.Fatalf("Failed to hash block bytes got: %v", err)
	}
	newState, err := blocks.ProcessBlockHeader(state, block)
	if err != nil {
		t.Fatalf("Failed to process block header got: %v", err)
	}
	nsh := newState.LatestBlockHeader
	expected := &pb.BeaconBlockHeader{
		Slot:       block.Slot,
		ParentRoot: latestBlockSignedRoot[:],
		BodyRoot:   bodyRoot[:],
	}
	if !proto.Equal(nsh, expected) {
		t.Errorf("Expected %v, received %vk9k", expected, nsh)
	}
}

func TestProcessRandao_IncorrectProposerFailsVerification(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
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
	domain := helpers.DomainVersion(beaconState, epoch, params.BeaconConfig().DomainRandao)

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
		true,  /* verify signatures */
		false, /* disable logging */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
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
	domain := helpers.DomainVersion(beaconState, epoch, params.BeaconConfig().DomainRandao)
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	newState, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
		true,  /* verify signatures */
		false, /* disable logging */
	)
	if err != nil {
		t.Errorf("Unexpected error processing block randao: %v", err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	mix := newState.LatestRandaoMixes[currentEpoch%params.BeaconConfig().LatestRandaoMixesLength]

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
				BlockRoot:   []byte{3},
			},
		},
	}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEth1VotingPeriod; i++ {
		beaconState = blocks.ProcessEth1DataInBlock(beaconState, block)
	}

	newETH1DataVotes := beaconState.Eth1DataVotes
	if len(newETH1DataVotes) <= 1 {
		t.Error("Expected new ETH1 data votes to have length > 1")
	}
	if !proto.Equal(beaconState.LatestEth1Data, block.Body.Eth1Data) {
		t.Errorf(
			"Expected latest eth1 data to have been set to %v, received %v",
			block.Body.Eth1Data,
			beaconState.LatestEth1Data,
		)
	}
}

func TestProcessProposerSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.Validator{}
	currentSlot := uint64(0)

	want := fmt.Sprintf(
		"number of proposer slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxProposerSlashings+1,
		params.BeaconConfig().MaxProposerSlashings,
	)
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	if _, err := blocks.ProcessProposerSlashings(

		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
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
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header epochs"
	if _, err := blocks.ProcessProposerSlashings(
		beaconState,
		block,
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
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	if _, err := blocks.ProcessProposerSlashings(

		beaconState,
		block,
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
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		beaconState.ValidatorRegistry[0].Pubkey,
	)

	if _, err := blocks.ProcessProposerSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	helpers.RestartShuffledValidatorCache()
	validators := make([]*pb.Validator, 100)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			EffectiveBalance:  params.BeaconConfig().MaxDepositAmount,
			Slashed:           false,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			SlashedEpoch:      1,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:   0,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
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
		ValidatorRegistry:      validators,
		Slot:                   currentSlot,
		Balances:               validatorBalances,
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessProposerSlashings(
		beaconState,
		block,
		false,
	)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	newStateVals := newState.ValidatorRegistry
	if newStateVals[1].ExitEpoch != validators[1].ExitEpoch {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			newStateVals[1].ExitEpoch, validators[1].ExitEpoch)
	}
}

func TestProcessAttesterSlashings_ThresholdReached(t *testing.T) {
	slashings := make([]*pb.AttesterSlashing, params.BeaconConfig().MaxAttesterSlashings+1)
	registry := []*pb.Validator{}
	currentSlot := uint64(0)

	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"number of attester slashings (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxAttesterSlashings+1,
		params.BeaconConfig().MaxAttesterSlashings,
	)

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_DataNotSlashable(t *testing.T) {
	slashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 1,
					TargetEpoch: 1,
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
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("attestations are not slashable")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
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
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1, 2},
				CustodyBit_1Indices: []uint64{0, 1, 2},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
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
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("expected no bit 1 indices")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	slashings = []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxIndicesPerAttestation+1),
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxIndicesPerAttestation+1),
			},
		},
	}

	block.Body.AttesterSlashings = slashings
	want = fmt.Sprint("exceeded max number of bit indices")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	slashings = []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{3, 2, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{3, 2, 1},
			},
		},
	}

	block.Body.AttesterSlashings = slashings
	want = fmt.Sprint("bit indices not sorted")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
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
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	slashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					SourceEpoch: 0,
					TargetEpoch: 0,
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
		ValidatorRegistry:      validators,
		Slot:                   currentSlot,
		Balances:               validatorBalances,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	newState, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.ValidatorRegistry

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

func TestProcessBlockAttestations_ThresholdReached(t *testing.T) {
	attestations := make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{}

	want := fmt.Sprintf(
		"number of attestations in block (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxAttestations+1,
		params.BeaconConfig().MaxAttestations,
	)

	if _, err := blocks.ProcessBlockAttestations(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
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
	deposits, _ := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
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
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
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
	deposits, _ := setupInitialDeposits(t, 200)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay

	want := fmt.Sprintf(
		"expected target epoch %d == %d or %d",
		attestations[0].Data.TargetEpoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CurrentEpochFFGDataMismatches(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
				SourceEpoch: 1,
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
	deposits, _ := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
		},
	}
	beaconState.CurrentJustifiedRoot = []byte("tron-sucks")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.CurrentEpoch(beaconState),
		attestations[0].Data.SourceEpoch,
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.SourceEpoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.SourceRoot = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedRoot,
		attestations[0].Data.SourceRoot,
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
				SourceEpoch: 1,
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
	deposits, _ := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.PreviousCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
		},
	}
	beaconState.PreviousJustifiedRoot = []byte("tron-sucks")
	beaconState.PreviousEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.SourceEpoch,
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.SourceEpoch = helpers.PrevEpoch(beaconState)
	block.Body.Attestations[0].Data.SourceRoot = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.PreviousJustifiedRoot,
		attestations[0].Data.SourceRoot,
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CrosslinkMismatches(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
				SourceEpoch: 0,
				SourceRoot:  []byte("tron-sucks"),
				Crosslink: &pb.Crosslink{
					Shard: 0,
					Epoch: 1,
				},
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits, _ := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
			Epoch: 0,
		},
	}
	beaconState.CurrentJustifiedRoot = []byte("tron-sucks")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	want := fmt.Sprintf(
		"expected crosslink epoch %d, received %d",
		0,
		attestations[0].Data.Crosslink.Epoch,
	)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	block.Body.Attestations[0].Data.Crosslink.Epoch = 0
	want = "mismatched parent crosslink root"
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
	encoded, err := ssz.TreeHash(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = encoded[:]

	want = fmt.Sprintf("expected data root %#x == ZERO_HASH", encoded)
	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_OK(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				TargetEpoch: 0,
				SourceEpoch: 0,
				SourceRoot:  []byte("tron-sucks"),
				Crosslink: &pb.Crosslink{
					Shard: 0,
					Epoch: 0,
				},
			},
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
			CustodyBitfield:     []byte{},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(deposits); i++ {
		depositData := &pb.DepositData{
			Pubkey: []byte(strconv.Itoa(i)),
			Amount: params.BeaconConfig().MaxDepositAmount,
		}

		deposits[i] = &pb.Deposit{Data: depositData}
	}
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard: 0,
			Epoch: 0,
		},
	}
	beaconState.CurrentJustifiedRoot = []byte("tron-sucks")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	encoded, err := ssz.TreeHash(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]

	if _, err := blocks.ProcessBlockAttestations(
		beaconState,
		block,
		false,
	); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateIndexedAttestation_OK(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{1, 3, 5, 10, 12},
		CustodyBit_1Indices: []uint64{},
	}

	if err := blocks.ValidateIndexedAttestation(indexedAtt1, false); err != nil {
		t.Errorf("indexed attestation failed to verify: %v", err)
	}
}

func TestConvertToIndexed_OK(t *testing.T) {
	helpers.RestartShuffledValidatorCache()
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
		Slot:                   5,
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	tests := []struct {
		aggregationBitfield      []byte
		custodyBitfield          []byte
		wantedCustodyBit0Indices []uint64
		wantedCustodyBit1Indices []uint64
	}{
		{
			aggregationBitfield:      []byte{0x03},
			custodyBitfield:          []byte{0x01},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{82, 84},
		},
		{
			aggregationBitfield:      []byte{0x03},
			custodyBitfield:          []byte{0x02},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{82, 84},
		},
		{
			aggregationBitfield:      []byte{0x03},
			custodyBitfield:          []byte{0x03},
			wantedCustodyBit0Indices: []uint64{},
			wantedCustodyBit1Indices: []uint64{82, 84},
		},
	}

	attestation := &pb.Attestation{
		Signature: []byte("signed"),
		Data: &pb.AttestationData{
			SourceEpoch: 0,
			TargetEpoch: 0,
			Crosslink: &pb.Crosslink{
				Shard: 3,
			},
		},
	}
	for _, tt := range tests {
		attestation.AggregationBitfield = tt.aggregationBitfield
		attestation.CustodyBitfield = tt.custodyBitfield
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
			t.Errorf("convert attestation to indexed attestation didn't result as wanted: %v got: %v", wanted, ia)
		}
	}

}

func TestValidateIndexedAttestation_Custody1Length(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{1, 2, 3, 4},
		CustodyBit_1Indices: []uint64{},
	}

	if err := blocks.ValidateIndexedAttestation(
		indexedAtt1,
		false,
	); err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestValidateIndexedAttestation_Empty(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{},
		CustodyBit_1Indices: []uint64{},
	}

	if err := blocks.ValidateIndexedAttestation(
		indexedAtt1,
		false,
	); err != nil {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestValidateIndexedAttestation_AboveMaxLength(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: make([]uint64, params.BeaconConfig().MaxIndicesPerAttestation+5),
		CustodyBit_1Indices: []uint64{},
	}

	for i := uint64(0); i < params.BeaconConfig().MaxIndicesPerAttestation+5; i++ {
		indexedAtt1.CustodyBit_0Indices[i] = i
	}

	want := "exceeded max number of bit indices"
	if err := blocks.ValidateIndexedAttestation(
		indexedAtt1,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestValidateIndexedAttestation_NotSorted(t *testing.T) {
	indexedAtt1 := &pb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{3, 1, 10, 4, 2},
		CustodyBit_1Indices: []uint64{},
	}

	want := "indices not sorted"
	if err := blocks.ValidateIndexedAttestation(
		indexedAtt1,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestProcessValidatorDeposits_ThresholdReached(t *testing.T) {
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: make([]*pb.Deposit, params.BeaconConfig().MaxDeposits+1),
		},
	}
	beaconState := &pb.BeaconState{}
	want := "exceeds allowed threshold"
	if _, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
		false, /* verifySignatures */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_MerkleBranchFailsVerification(t *testing.T) {
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey: []byte{1, 2, 3},
		},
	}
	leaf, err := ssz.TreeHash(deposit.Data)
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
	deposit.Index = 0
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{
		LatestEth1Data: &pb.Eth1Data{
			DepositRoot: []byte{0},
			BlockRoot:   []byte{1},
		},
	}
	want := "deposit root did not verify"
	if _, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
		false, /* verifySignatures */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_IncorrectMerkleIndex(t *testing.T) {
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey: []byte{1, 2, 3},
		},
	}
	leaf, err := ssz.TreeHash(deposit.Data)
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
	deposit.Index = 0
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
	depositRoot := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Balances:          balances,
		Slot:              0,
		GenesisTime:       uint64(0),
		DepositIndex:      1,
		LatestEth1Data: &pb.Eth1Data{
			DepositRoot: depositRoot[:],
			BlockRoot:   []byte{1},
		},
	}

	want := "expected deposit merkle tree index to match beacon state deposit index"
	if _, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
		false, /* verifySignatures */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_ProcessCorrectly(t *testing.T) {
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey: []byte{1, 2, 3},
			Amount: params.BeaconConfig().MaxDepositAmount,
		},
	}
	leaf, err := ssz.TreeHash(deposit.Data)
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
	deposit.Index = 0
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
		ValidatorRegistry: registry,
		Balances:          balances,
		LatestEth1Data: &pb.Eth1Data{
			DepositRoot: root[:],
			BlockRoot:   root[:],
		},
	}
	newState, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
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

func TestProcessValidatorExits_ThresholdReached(t *testing.T) {
	exits := make([]*pb.VoluntaryExit, params.BeaconConfig().MaxVoluntaryExits+1)
	registry := []*pb.Validator{}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := fmt.Sprintf(
		"number of exits (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxVoluntaryExits+1,
		params.BeaconConfig().MaxVoluntaryExits,
	)

	if _, err := blocks.ProcessValidatorExits(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_ValidatorNotActive(t *testing.T) {
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
		ValidatorRegistry: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "non-active validator cannot exit"

	if _, err := blocks.ProcessValidatorExits(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_InvalidExitEpoch(t *testing.T) {
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
		ValidatorRegistry: registry,
		Slot:              0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "expected current epoch >= exit epoch"

	if _, err := blocks.ProcessValidatorExits(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_NotActiveLongEnoughToExit(t *testing.T) {
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
		ValidatorRegistry: registry,
		Slot:              10,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "validator has not been active long enough to exit"
	if _, err := blocks.ProcessValidatorExits(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_AppliesCorrectStatus(t *testing.T) {
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
		ValidatorRegistry: registry,
		Slot:              params.BeaconConfig().SlotsPerEpoch * 5,
	}
	state.Slot = state.Slot + (params.BeaconConfig().PersistentCommitteePeriod * params.BeaconConfig().SlotsPerEpoch)
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}
	newState, err := blocks.ProcessValidatorExits(state, block, false)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.ValidatorRegistry
	if newRegistry[0].ExitEpoch != helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.DelayedActivationExitEpoch(state.Slot/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}

func TestProcessBeaconTransfers_ThresholdReached(t *testing.T) {
	transfers := make([]*pb.Transfer, params.BeaconConfig().MaxTransfers+1)
	registry := []*pb.Validator{}
	state := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: transfers,
		},
	}

	want := fmt.Sprintf(
		"number of transfers (%d) exceeds allowed threshold of %d",
		params.BeaconConfig().MaxTransfers+1,
		params.BeaconConfig().MaxTransfers,
	)

	if _, err := blocks.ProcessTransfers(
		state,
		block,
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
	balances := []uint64{params.BeaconConfig().MaxDepositAmount}
	state := &pb.BeaconState{
		Slot:              0,
		ValidatorRegistry: registry,
		Balances:          balances,
	}
	transfers := []*pb.Transfer{
		{
			Fee: params.BeaconConfig().MaxDepositAmount + 1,
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
		block,
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
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	state.ValidatorRegistry[0].WithdrawableEpoch = params.BeaconConfig().FarFutureEpoch
	state.ValidatorRegistry[0].ActivationEligibilityEpoch = 0
	block.Body.Transfers = []*pb.Transfer{
		{
			Fee:    params.BeaconConfig().MinDepositAmount,
			Amount: params.BeaconConfig().MaxDepositAmount,
			Slot:   state.Slot,
		},
	}
	want = "over max transfer"
	if _, err := blocks.ProcessTransfers(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	state.ValidatorRegistry[0].WithdrawableEpoch = 0
	state.ValidatorRegistry[0].ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
	buf := []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("B")
	hashed := hashutil.Hash(pubKey)
	buf = append(buf, hashed[:]...)
	state.ValidatorRegistry[0].WithdrawalCredentials = buf
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
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBeaconTransfers_OK(t *testing.T) {
	helpers.RestartShuffledValidatorCache()
	testConfig := params.BeaconConfig()
	testConfig.MaxTransfers = 1
	params.OverrideBeaconConfig(testConfig)
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart/32)
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
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   0,
		Balances:               validatorBalances,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
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
	buf = append(buf, hashed[:]...)
	state.ValidatorRegistry[0].WithdrawalCredentials = buf
	state.ValidatorRegistry[0].ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
	newState, err := blocks.ProcessTransfers(
		state,
		block,
		false,
	)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	expectedRecipient := params.BeaconConfig().MaxDepositAmount + block.Body.Transfers[0].Amount
	if newState.Balances[1] != expectedRecipient {
		t.Errorf("Expected recipient balance %d, received %d", newState.Balances[1], expectedRecipient)
	}
	expectedSender := params.BeaconConfig().MaxDepositAmount - block.Body.Transfers[0].Amount - block.Body.Transfers[0].Fee
	if newState.Balances[0] != expectedSender {
		t.Errorf("Expected sender balance %d, received %d", newState.Balances[0], expectedSender)
	}
}
