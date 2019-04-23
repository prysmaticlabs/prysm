package epoch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	return &pb.BeaconState{
		ValidatorRegistry: validators,
		ValidatorBalances: validatorBalances,
		Slot:              slot,
	}
}

func TestWinningRoot_AccurateRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, 100)
	var participationBitfield []byte
	participationBitfield = append(participationBitfield, byte(0x80))

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    params.BeaconConfig().GenesisSlot,
				CrosslinkDataRootHash32: []byte{byte(i + 100)},
			},
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	// Since all 10 roots have the balance of 64 ETHs
	// winningRoot chooses the lowest hash: []byte{100}
	winnerRoot, err := winningRoot(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute winningRoot: %v", err)
	}
	if !bytes.Equal(winnerRoot, []byte{100}) {
		t.Errorf("Incorrect winner root, wanted:[100], got: %v", winnerRoot)
	}
}

func TestWinningRoot_EmptyParticipantBitfield(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{
			Slot:                    params.BeaconConfig().GenesisSlot,
			CrosslinkDataRootHash32: []byte{},
		},
			AggregationBitfield: []byte{},
		},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := winningRoot(state, 0, attestations, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestAttestingValidators_MatchActive(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().SlotsPerEpoch*2)

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    params.BeaconConfig().GenesisSlot,
				CrosslinkDataRootHash32: []byte{byte(i + 100)},
			},
			AggregationBitfield: []byte{0xC0},
		}
		attestations = append(attestations, attestation)
	}

	helpers.RestartCommitteeCache()

	attestedValidators, err := AttestingValidators(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute AttestingValidators: %v", err)
	}

	// Verify the winner root is attested by validators based on shuffling.
	if !reflect.DeepEqual(attestedValidators, []uint64{123, 65}) {
		t.Errorf("Active validators don't match. Wanted:[123,65], Got: %v", attestedValidators)
	}
}

func TestAttestingValidators_EmptyWinningRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                    params.BeaconConfig().GenesisSlot,
			CrosslinkDataRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := AttestingValidators(state, 0, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalAttestingBalance_CorrectBalance(t *testing.T) {
	validatorsPerCommittee := uint64(2)
	state := buildState(params.BeaconConfig().GenesisSlot, 2*params.BeaconConfig().SlotsPerEpoch)

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    params.BeaconConfig().GenesisSlot,
				CrosslinkDataRootHash32: []byte{byte(i + 100)},
			},
			// All validators attested to the above roots.
			AggregationBitfield: []byte{0xC0},
		}
		attestations = append(attestations, attestation)
	}

	helpers.RestartCommitteeCache()

	attestedBalance, err := TotalAttestingBalance(
		state,
		0,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute totalAttestingBalance: %v", err)
	}

	if attestedBalance != params.BeaconConfig().MaxDepositAmount*validatorsPerCommittee {
		t.Errorf("Incorrect attested balance. Wanted:64*1e9, Got: %d", attestedBalance)
	}
}

func TestTotalAttestingBalance_EmptyWinningRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                    params.BeaconConfig().GenesisSlot,
			CrosslinkDataRootHash32: []byte{},
		},
		AggregationBitfield: []byte{},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := TotalAttestingBalance(state, 0, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalBalance_CorrectBalance(t *testing.T) {
	// Assign validators to different balances.
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorBalances: []uint64{20 * 1e9, 25 * 1e9, 30 * 1e9, 30 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	// 20 + 25 + 30 + 30 + 32 + 32 + 32 + 32 = 233
	totalBalance := TotalBalance(state, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	if totalBalance != 233*1e9 {
		t.Errorf("Incorrect total balance. Wanted: 233*1e9, got: %d", totalBalance)
	}
}

func TestInclusionSlot_GetsCorrectSlot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       101},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       100},
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       102},
	}
	slot, err := InclusionSlot(state, 251)
	if err != nil {
		t.Fatalf("Could not execute InclusionSlot: %v", err)
	}
	// validator 45's attestation got included in slot 100.
	if slot != 100 {
		t.Errorf("Incorrect slot. Wanted: 100, got: %d", slot)
	}
}

func TestInclusionSlot_InvalidBitfield(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: []byte{},
			InclusionSlot:       100},
	}

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := InclusionSlot(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestInclusionSlot_SlotNotFound(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().SlotsPerEpoch)

	badIndex := uint64(10000)
	want := fmt.Sprintf("could not find inclusion slot for validator index %d", badIndex)
	if _, err := InclusionSlot(state, badIndex); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}
