package blocks_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"

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
		depositInput := &pb.DepositInput{
			Pubkey: priv.PublicKey().Marshal(),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func TestProcessBlockRandao_IncorrectProposerFailsVerification(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, params.BeaconConfig().GenesisSlot)
	if err != nil {
		t.Fatal(err)
	}
	epoch := helpers.SlotToEpoch(params.BeaconConfig().GenesisSlot)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)

	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(buf, domain)
	block := &pb.BeaconBlock{
		RandaoReveal: epochSignature.Marshal(),
	}

	want := "block randao reveal signature did not verify"
	if _, err := blocks.ProcessBlockRandao(

		beaconState,
		block,
		true,  /* verify signatures */
		false, /* disable logging */
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessBlockRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, params.BeaconConfig().GenesisSlot)
	if err != nil {
		t.Fatal(err)
	}
	epoch := helpers.SlotToEpoch(params.BeaconConfig().GenesisSlot)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)

	block := &pb.BeaconBlock{
		RandaoReveal: epochSignature.Marshal(),
	}

	newState, err := blocks.ProcessBlockRandao(

		beaconState,
		block,
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

func TestProcessEth1Data_SameRootHash(t *testing.T) {
	beaconState := &pb.BeaconState{
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{1},
					BlockHash32:       []byte{2},
				},
				VoteCount: 5,
			},
		},
	}
	block := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{1},
			BlockHash32:       []byte{2},
		},
	}
	beaconState = blocks.ProcessEth1DataInBlock(beaconState, block)
	newETH1DataVotes := beaconState.Eth1DataVotes
	if newETH1DataVotes[0].VoteCount != 6 {
		t.Errorf("expected votes to increase from 5 to 6, received %d", newETH1DataVotes[0].VoteCount)
	}
}

func TestProcessEth1Data_NewDepositRootHash(t *testing.T) {
	beaconState := &pb.BeaconState{
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{0},
					BlockHash32:       []byte{1},
				},
				VoteCount: 5,
			},
		},
	}

	block := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{2},
			BlockHash32:       []byte{3},
		},
	}

	beaconState = blocks.ProcessEth1DataInBlock(beaconState, block)
	newETH1DataVotes := beaconState.Eth1DataVotes
	if len(newETH1DataVotes) <= 1 {
		t.Error("expected new ETH1 data votes to have length > 1")
	}
	if newETH1DataVotes[1].VoteCount != 1 {
		t.Errorf(
			"expected new ETH1 data votes to have a new element with votes = 1, received votes = %d",
			newETH1DataVotes[1].VoteCount,
		)
	}
	if !bytes.Equal(newETH1DataVotes[1].Eth1Data.DepositRootHash32, []byte{2}) {
		t.Errorf(
			"expected new ETH1 data votes to have a new element with deposit root = %#x, received deposit root = %#x",
			[]byte{1},
			newETH1DataVotes[1].Eth1Data.DepositRootHash32,
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

func TestProcessProposerSlashings_UnmatchedSlotNumbers(t *testing.T) {
	registry := []*pb.Validator{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot: 1,
			},
			ProposalData_2: &pb.ProposalSignedData{
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
	want := "slashing proposal data slots do not match: 1, 0"
	if _, err := blocks.ProcessProposerSlashings(

		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_UnmatchedShards(t *testing.T) {
	registry := []*pb.Validator{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:  1,
				Shard: 0,
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:  1,
				Shard: 1,
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
	want := "slashing proposal data shards do not match: 0, 1"
	if _, err := blocks.ProcessProposerSlashings(

		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_UnmatchedBlockRoots(t *testing.T) {
	registry := []*pb.Validator{}
	currentSlot := uint64(0)
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           0,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            1,
				Shard:           0,
				BlockRootHash32: []byte{1, 1, 0},
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
		"slashing proposal data block roots do not match: %#x, %#x",
		[]byte{0, 1, 0}, []byte{1, 1, 0},
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

	validators := make([]*pb.Validator, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:       params.BeaconConfig().GenesisEpoch + 1,
			SlashedEpoch:    params.BeaconConfig().GenesisEpoch + 1,
			WithdrawalEpoch: params.BeaconConfig().GenesisEpoch + 1,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:            params.BeaconConfig().GenesisSlot + 1,
				Shard:           1,
				BlockRootHash32: []byte{0, 1, 0},
			},
		},
	}
	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState := &pb.BeaconState{
		ValidatorRegistry:     validators,
		Slot:                  currentSlot,
		ValidatorBalances:     validatorBalances,
		LatestSlashedBalances: []uint64{0},
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
			newStateVals[1], validators[1].ExitEpoch)
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

func TestProcessAttesterSlashings_EmptyCustodyFields(t *testing.T) {
	slashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:  5,
					Shard: 4,
				},
				ValidatorIndices: make(
					[]uint64,
					params.BeaconConfig().MaxIndicesPerSlashableVote,
				),
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:  5,
					Shard: 3,
				},
				ValidatorIndices: make(
					[]uint64,
					params.BeaconConfig().MaxIndicesPerSlashableVote,
				),
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
	want := fmt.Sprint("custody bit field can't all be 0")

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}

	// Perform the same check for SlashableVoteData_2.
	slashings = []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:  5,
					Shard: 4,
				},
				ValidatorIndices: make(
					[]uint64,
					params.BeaconConfig().MaxIndicesPerSlashableVote,
				),
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data: &pb.AttestationData{
					Slot:  5,
					Shard: 3,
				},
				ValidatorIndices: make(
					[]uint64,
					params.BeaconConfig().MaxIndicesPerSlashableVote,
				),
			},
		},
	}
	beaconState = &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              currentSlot,
	}
	block = &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_UnmatchedAttestations(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot: 5,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{2},
				CustodyBitfield:  []byte{0xFF},
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
	want := fmt.Sprintf(
		"attester slashing inner slashable vote data attestation should not match: %v, %v",
		att1,
		att1,
	)

	if _, err := blocks.ProcessAttesterSlashings(
		beaconState,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_EmptyVoteIndexIntersection(t *testing.T) {
	att1 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           5,
		JustifiedEpoch: 4,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{9, 10, 11, 12, 13, 14, 15, 16},
				CustodyBitfield:  []byte{0xFF},
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
	want := "expected a non-empty list"
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
			ActivationEpoch: params.BeaconConfig().GenesisEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			SlashedEpoch:    params.BeaconConfig().FarFutureEpoch,
			WithdrawalEpoch: params.BeaconConfig().GenesisEpoch + 1*params.BeaconConfig().SlotsPerEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}

	att1 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch,
		JustifiedEpoch: 5,
	}
	att2 := &pb.AttestationData{
		Slot:           params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch,
		JustifiedEpoch: 4,
	}
	slashings := []*pb.AttesterSlashing{
		{
			SlashableAttestation_1: &pb.SlashableAttestation{
				Data:             att1,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
			SlashableAttestation_2: &pb.SlashableAttestation{
				Data:             att2,
				ValidatorIndices: []uint64{1, 2, 3, 4, 5, 6, 7, 8},
				CustodyBitfield:  []byte{0xFF},
			},
		},
	}

	currentSlot := params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch
	beaconState := &pb.BeaconState{
		ValidatorRegistry:     validators,
		Slot:                  currentSlot,
		ValidatorBalances:     validatorBalances,
		LatestSlashedBalances: make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
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
				Slot: params.BeaconConfig().GenesisSlot + 5,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 5,
	}

	want := fmt.Sprintf(
		"attestation slot (slot %d) + inclusion delay (%d) beyond current beacon state slot (%d)",
		5,
		params.BeaconConfig().MinAttestationInclusionDelay,
		5,
	)
	if _, err := blocks.ProcessBlockAttestations(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_EpochDistanceFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + 5,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 5 + 2*params.BeaconConfig().SlotsPerEpoch,
	}

	want := fmt.Sprintf(
		"attestation slot (slot %d) + epoch length (%d) less than current beacon state slot (%d)",
		5,
		params.BeaconConfig().SlotsPerEpoch,
		5+2*params.BeaconConfig().SlotsPerEpoch,
	)
	if _, err := blocks.ProcessBlockAttestations(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_JustifiedEpochVerificationFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:           params.BeaconConfig().GenesisSlot + 152,
				JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 2,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot:           params.BeaconConfig().GenesisSlot + 158,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 1,
	}

	want := fmt.Sprintf(
		"expected attestation.JustifiedEpoch == state.JustifiedEpoch, received %d == %d",
		2,
		1,
	)
	if _, err := blocks.ProcessBlockAttestations(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_PreviousJustifiedEpochVerificationFailure(t *testing.T) {
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Slot:           params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch + 1,
				JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 3,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch,
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch + 2,
	}

	want := fmt.Sprintf(
		"expected attestation.JustifiedEpoch == state.PreviousJustifiedEpoch, received %d == %d",
		3,
		2,
	)
	if _, err := blocks.ProcessBlockAttestations(
		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CrosslinkRootFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().SlotsPerEpoch; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}

	// If attestation.latest_cross_link_root != state.latest_crosslinks[shard].shard_block_root
	// AND
	// attestation.data.shard_block_root != state.latest_crosslinks[shard].shard_block_root
	// the attestation should be invalid.
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 70,
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		LatestBlockRootHash32S: blockRoots,
		PreviousJustifiedRoot:  blockRoots[0],
		LatestCrosslinks:       stateLatestCrosslinks,
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Shard:                    0,
				Slot:                     params.BeaconConfig().GenesisSlot + 20,
				JustifiedBlockRootHash32: blockRoots[0],
				LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{2}},
				CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	want := fmt.Sprintf(
		"incoming attestation does not match crosslink in state for shard %d",
		attestations[0].Data.Shard,
	)
	if _, err := blocks.ProcessBlockAttestations(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_ShardBlockRootEqualZeroHashFailure(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < 2*params.BeaconConfig().SlotsPerEpoch; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 70,
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       stateLatestCrosslinks,
		PreviousJustifiedRoot:  blockRoots[0],
	}
	attestations := []*pb.Attestation{
		{
			Data: &pb.AttestationData{
				Shard:                    0,
				Slot:                     params.BeaconConfig().GenesisSlot + 20,
				JustifiedBlockRootHash32: blockRoots[0],
				LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
				CrosslinkDataRootHash32:  []byte{1},
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
			},
		},
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	want := fmt.Sprintf(
		"expected attestation.data.CrosslinkDataRootHash == %#x, received %#x instead",
		params.BeaconConfig().ZeroHash[:],
		[]byte{1},
	)
	if _, err := blocks.ProcessBlockAttestations(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlockAttestations_CreatePendingAttestations(t *testing.T) {
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	stateLatestCrosslinks := []*pb.Crosslink{
		{
			CrosslinkDataRootHash32: []byte{1},
		},
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().GenesisSlot + 70,
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		LatestBlockRootHash32S: blockRoots,
		LatestCrosslinks:       stateLatestCrosslinks,
		PreviousJustifiedRoot:  blockRoots[0],
	}
	att1 := &pb.Attestation{
		Data: &pb.AttestationData{
			Shard:                    0,
			Slot:                     params.BeaconConfig().GenesisSlot + 20,
			JustifiedBlockRootHash32: blockRoots[0],
			LatestCrosslink:          &pb.Crosslink{CrosslinkDataRootHash32: []byte{1}},
			CrosslinkDataRootHash32:  params.BeaconConfig().ZeroHash[:],
			JustifiedEpoch:           params.BeaconConfig().GenesisEpoch,
		},
		AggregationBitfield: []byte{1},
		CustodyBitfield:     []byte{1},
	}
	attestations := []*pb.Attestation{att1}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	newState, err := blocks.ProcessBlockAttestations(

		state,
		block,
		false,
	)
	pendingAttestations := newState.LatestAttestations
	if err != nil {
		t.Fatalf("Could not produce pending attestations: %v", err)
	}
	if !reflect.DeepEqual(pendingAttestations[0].Data, att1.Data) {
		t.Errorf(
			"Did not create pending attestation correctly with inner data, wanted %v, received %v",
			att1.Data,
			pendingAttestations[0].Data,
		)
	}
	if pendingAttestations[0].InclusionSlot != params.BeaconConfig().GenesisSlot+70 {
		t.Errorf(
			"Pending attestation not included at correct slot: wanted %v, received %v",
			64,
			pendingAttestations[0].InclusionSlot,
		)
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
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_DepositDataSizeTooSmall(t *testing.T) {
	hook := logTest.NewGlobal()
	data := []byte{1, 2, 3}
	deposit := &pb.Deposit{
		DepositData: data,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{}
	want := "deposit data slice too small"
	_, _ = blocks.ProcessValidatorDeposits(
		beaconState,
		block,
	)
	testutil.AssertLogsContain(t, hook, want)
}

func TestProcessValidatorDeposits_DepositInputDecodingFails(t *testing.T) {
	hook := logTest.NewGlobal()
	data := make([]byte, 16)
	deposit := &pb.Deposit{
		DepositData: data,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{}
	want := "ssz decode failed"
	_, _ = blocks.ProcessValidatorDeposits(

		beaconState,
		block,
	)
	testutil.AssertLogsContain(t, hook, want)
}

func TestProcessValidatorDeposits_MerkleBranchFailsVerification(t *testing.T) {
	// We create a correctly encoded deposit data using Simple Serialize.
	depositInput := &pb.DepositInput{
		Pubkey: []byte{1, 2, 3},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}
	value := make([]byte, 8)
	timestamp := make([]byte, 8)
	data = append(data, encodedInput...)
	data = append(data, value...)
	data = append(data, timestamp...)

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{data}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit := &pb.Deposit{
		DepositData:        data,
		MerkleProofHash32S: proof,
		MerkleTreeIndex:    0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	beaconState := &pb.BeaconState{
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{0},
			BlockHash32:       []byte{1},
		},
	}
	want := "merkle branch of deposit root did not verify"
	if _, err := blocks.ProcessValidatorDeposits(

		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_ProcessDepositHelperFuncFails(t *testing.T) {
	hook := logTest.NewGlobal()
	// Having mismatched withdrawal credentials will cause the process deposit
	// validator helper function to fail with error when the public key
	// currently exists in the validator registry.
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{1, 2, 3},
		ProofOfPossession:           []byte{},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(1000))

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{data}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}
	deposit := &pb.Deposit{
		DepositData:        data,
		MerkleProofHash32S: proof,
		MerkleTreeIndex:    0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	// The validator will have a mismatched withdrawal credential than
	// the one specified in the deposit input, causing a failure.
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{4, 5, 6},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		ValidatorBalances: balances,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: root[:],
			BlockHash32:       root[:],
		},
		Slot:        currentSlot,
		GenesisTime: uint64(genesisTime),
	}
	want := "expected withdrawal credentials to match"
	_, err = blocks.ProcessValidatorDeposits(

		beaconState,
		block,
	)
	testutil.AssertLogsContain(t, hook, want)
}

func TestProcessValidatorDeposits_IncorrectMerkleIndex(t *testing.T) {
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{1, 2, 3},
		ProofOfPossession:           []byte{},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	depositValue := uint64(1000)
	binary.LittleEndian.PutUint64(value, depositValue)

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	deposit := &pb.Deposit{
		DepositData:        data,
		MerkleProofHash32S: [][]byte{{0}},
		MerkleTreeIndex:    1,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		ValidatorBalances: balances,
		Slot:              currentSlot,
		GenesisTime:       uint64(genesisTime),
	}

	want := "expected deposit merkle tree index to match beacon state deposit index"
	if _, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessValidatorDeposits_ProcessCorrectly(t *testing.T) {
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{1, 2, 3},
		ProofOfPossession:           []byte{},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	depositValue := uint64(1000)
	binary.LittleEndian.PutUint64(value, depositValue)

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{data}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit := &pb.Deposit{
		DepositData:        data,
		MerkleProofHash32S: proof,
		MerkleTreeIndex:    0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		ValidatorBalances: balances,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: root[:],
			BlockHash32:       root[:],
		},
		Slot:        currentSlot,
		GenesisTime: uint64(genesisTime),
	}
	newState, err := blocks.ProcessValidatorDeposits(

		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Expected block deposits to process correctly, received: %v", err)
	}
	if newState.ValidatorBalances[0] != depositValue {
		t.Errorf(
			"Expected state validator balances index 0 to equal %d, received %d",
			depositValue,
			newState.ValidatorBalances[0],
		)
	}
}

func TestProcessValidatorDeposits_InvalidSSZ_DepositIndexIncremented(t *testing.T) {
	hook := logTest.NewGlobal()
	encodedInput := []byte{'A', 'B', 'C', 'D'}
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	depositValue := uint64(1000)
	binary.LittleEndian.PutUint64(value, depositValue)

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	deposit := &pb.Deposit{
		DepositData:     data,
		MerkleTreeIndex: 0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		ValidatorBalances: balances,
		Slot:              currentSlot,
		GenesisTime:       uint64(genesisTime),
		DepositIndex:      1,
	}
	newState, err := blocks.ProcessValidatorDeposits(

		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Expected no errors returned but received this %v", err)
	}
	if newState.DepositIndex != 2 {
		t.Errorf(
			"Expected state deposit index to equal %d, received %d",
			2,
			newState.DepositIndex,
		)
	}
	testutil.AssertLogsContain(t, hook, "could not decode deposit input")
}

func TestProcessValidatorDeposits_InvalidWithdrawalCreds_DepositIndexIncremented(t *testing.T) {
	hook := logTest.NewGlobal()
	depositInput := &pb.DepositInput{
		Pubkey:                      []byte{1},
		WithdrawalCredentialsHash32: []byte{3, 2, 1},
		ProofOfPossession:           []byte{},
	}
	wBuf := new(bytes.Buffer)
	if err := ssz.Encode(wBuf, depositInput); err != nil {
		t.Fatalf("failed to encode deposit input: %v", err)
	}
	encodedInput := wBuf.Bytes()
	data := []byte{}

	// We set a deposit value of 1000.
	value := make([]byte, 8)
	depositValue := uint64(1000)
	binary.LittleEndian.PutUint64(value, depositValue)

	// We then serialize a unix time into the timestamp []byte slice
	// and ensure it has size of 8 bytes.
	timestamp := make([]byte, 8)

	// Set deposit time to 1000 seconds since unix time 0.
	depositTime := time.Unix(1000, 0).Unix()
	// Set genesis time to unix time 0.
	genesisTime := time.Unix(0, 0).Unix()

	currentSlot := 1000 * params.BeaconConfig().SecondsPerSlot
	binary.LittleEndian.PutUint64(timestamp, uint64(depositTime))

	// We then create a serialized deposit data slice of type []byte
	// by appending all 3 items above together.
	data = append(data, value...)
	data = append(data, timestamp...)
	data = append(data, encodedInput...)

	// We then create a merkle branch for the test.
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{data}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate proof: %v", err)
	}

	deposit := &pb.Deposit{
		DepositData:        data,
		MerkleProofHash32S: proof,
		MerkleTreeIndex:    0,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{deposit},
		},
	}
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1},
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		},
	}
	balances := []uint64{0}
	root := depositTrie.Root()
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
		ValidatorBalances: balances,
		DepositIndex:      0,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: root[:],
			BlockHash32:       root[:],
		},
		Slot:        currentSlot,
		GenesisTime: uint64(genesisTime),
	}
	newState, err := blocks.ProcessValidatorDeposits(
		beaconState,
		block,
	)
	if err != nil {
		t.Fatalf("Expected no error returned from processing the deposit: %v", err)
	}
	if newState.DepositIndex != 1 {
		t.Errorf(
			"Expected state deposit index to equal %d, received %d",
			1,
			newState.DepositIndex,
		)
	}

	testutil.AssertLogsContain(t, hook, "could not process deposit into beacon state")
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

	want := "validator exit epoch should be > entry_exit_effect_epoch"

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

	want := "expected current epoch >= exit.epoch"

	if _, err := blocks.ProcessValidatorExits(

		state,
		block,
		false,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessValidatorExits_InvalidStatusChangeSlot(t *testing.T) {
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 0,
			Epoch:          0,
		},
	}
	registry := []*pb.Validator{
		{
			ExitEpoch: 1,
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

	want := "exit epoch should be > entry_exit_effect_epoch"
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
	newState, err := blocks.ProcessValidatorExits(state, block, false)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.ValidatorRegistry
	if newRegistry[0].StatusFlags == pb.Validator_INITIAL {
		t.Error("Expected validator status to change, remained INITIAL")
	}
}
