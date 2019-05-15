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
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxDepositAmount,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &pb.BeaconState{
		Slot:                   slot,
		Balances:               validatorBalances,
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
	}
}

func TestWinningRoot_AccurateRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, 100)
	var participationBitfield []byte
	participationBitfield = append(participationBitfield, byte(0x01))

	// Generate 10 roots ([]byte{100}...[]byte{110})
	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              params.BeaconConfig().GenesisSlot,
				TargetEpoch:       params.BeaconConfig().GenesisEpoch,
				CrosslinkDataRoot: []byte{byte(i + 100)},
				Shard:             1,
			},
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	// Since all 10 roots have the balance of 64 ETHs
	// winningRoot chooses the lowest hash: []byte{100}
	winnerRoot, err := winningRoot(
		state,
		1,
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
		{
			Data: &pb.AttestationData{
				Slot:              params.BeaconConfig().GenesisSlot,
				TargetEpoch:       params.BeaconConfig().GenesisEpoch,
				CrosslinkDataRoot: []byte{},
				Shard:             2,
			},
			AggregationBitfield: []byte{},
		},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := winningRoot(state, 2, attestations, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestAttestingValidators_MatchActive(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().SlotsPerEpoch*2)

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              params.BeaconConfig().GenesisSlot,
				TargetEpoch:       params.BeaconConfig().GenesisEpoch,
				CrosslinkDataRoot: []byte{byte(i + 100)},
				Shard:             1,
			},
			AggregationBitfield: []byte{0x03},
		}
		attestations = append(attestations, attestation)
	}

	helpers.RestartCommitteeCache()

	attestedValidators, err := AttestingValidators(
		state,
		1,
		attestations,
		nil)
	if err != nil {
		t.Fatalf("Could not execute AttestingValidators: %v", err)
	}

	// Verify the winner root is attested by validators based on shuffling.
	if !reflect.DeepEqual(attestedValidators, []uint64{25, 87}) {
		t.Errorf("Active validators don't match. Wanted: [25, 87], Got: %v", attestedValidators)
	}
}

func TestAttestingValidators_EmptyWinningRoot(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:              params.BeaconConfig().GenesisSlot,
			TargetEpoch:       params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRoot: []byte{},
			Shard:             2,
		},
		AggregationBitfield: []byte{},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := AttestingValidators(state, 2, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
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
				Slot:              params.BeaconConfig().GenesisSlot,
				CrosslinkDataRoot: []byte{byte(i + 100)},
				TargetEpoch:       params.BeaconConfig().GenesisEpoch,
				Shard:             1,
			},
			// All validators attested to the above roots.
			AggregationBitfield: []byte{0x03},
		}
		attestations = append(attestations, attestation)
	}

	helpers.RestartCommitteeCache()

	attestedBalance, err := TotalAttestingBalance(
		state,
		1,
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
			Slot:              params.BeaconConfig().GenesisSlot,
			TargetEpoch:       params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRoot: []byte{},
			Shard:             2,
		},
		AggregationBitfield: []byte{},
	}

	helpers.RestartCommitteeCache()

	want := fmt.Sprintf("wanted participants bitfield length %d, got: %d", 16, 0)
	if _, err := TotalAttestingBalance(state, 2, []*pb.PendingAttestation{attestation}, nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestTotalBalance_CorrectBalance(t *testing.T) {
	// Assign validators to different balances.
	state := &pb.BeaconState{
		Slot: 5,
		Balances: []uint64{20 * 1e9, 25 * 1e9, 30 * 1e9, 30 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	// 20 + 25 + 30 + 30 + 32 + 32 + 32 + 32 = 233
	totalBalance := TotalBalance(state, []uint64{0, 1, 2, 3, 4, 5, 6, 7})
	if totalBalance != 233*1e9 {
		t.Errorf("Incorrect total balance. Wanted: 233*1e9, got: %d", totalBalance)
	}
}

func TestInclusionSlot_GetsCorrectSlot(t *testing.T) {
	t.Skip()

	state := buildState(params.BeaconConfig().GenesisSlot, params.BeaconConfig().DepositsForChainStart)
	var participationBitfield []byte
	for i := 0; i < 16; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	state.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{
			Slot:  params.BeaconConfig().GenesisSlot,
			Shard: 2,
		},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       101},
		{Data: &pb.AttestationData{
			Slot:  params.BeaconConfig().GenesisSlot,
			Shard: 2,
		},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       100},
		{Data: &pb.AttestationData{
			Slot:  params.BeaconConfig().GenesisSlot,
			Shard: 2,
		},
			AggregationBitfield: participationBitfield,
			InclusionSlot:       102},
	}
	slot, err := InclusionSlot(state, 849)
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
		{
			Data: &pb.AttestationData{
				Slot:        params.BeaconConfig().GenesisSlot,
				TargetEpoch: params.BeaconConfig().GenesisEpoch,
				Shard:       2,
			},
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

func TestProcessFinalUpdates_CanProcess(t *testing.T) {
	s := buildState(params.BeaconConfig().SlotsPerHistoricalRoot-1, params.BeaconConfig().SlotsPerEpoch)
	ce := helpers.CurrentEpoch(s)
	ne := ce + 1
	s.Eth1DataVotes = []*pb.Eth1DataVote{{VoteCount: 100}}
	s.Balances[0] = 29 * 1e9
	s.LatestSlashedBalances[ce] = 100
	s.LatestRandaoMixes[ce] = []byte{'A'}
	newS, err := ProcessFinalUpdates(s)
	if err != nil {
		t.Fatal(err)
	}

	// Verify we can reset ETH1 data votes.
	if len(newS.Eth1DataVotes) != 0 {
		t.Errorf("Eth1 data votes didnt get reset, got %d", len(newS.Eth1DataVotes))
	}

	// Verify effective balance is correctly updated.
	if newS.ValidatorRegistry[0].EffectiveBalance != 29*1e9 {
		t.Errorf("effective balance incorrectly updated, got %d", s.ValidatorRegistry[0].EffectiveBalance)
	}

	// Verify start shard is correctly updated.
	if newS.LatestStartShard != 64 {
		t.Errorf("start shard incorrectly updated, got %d", 64)
	}

	// Verify latest active index root is correctly updated in the right position.
	pos := (ne + params.BeaconConfig().ActivationExitDelay) % params.BeaconConfig().LatestActiveIndexRootsLength
	if bytes.Equal(newS.LatestActiveIndexRoots[pos], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest active index roots still zero hashes")
	}

	// Verify slashed balances correctly updated.
	if newS.LatestSlashedBalances[ce] != newS.LatestSlashedBalances[ne] {
		t.Errorf("wanted slashed balance %d, got %d",
			newS.LatestSlashedBalances[ce],
			newS.LatestSlashedBalances[ne])
	}

	// Verify randao is correctly updated in the right position.
	if bytes.Equal(newS.LatestRandaoMixes[ne], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest RANDAO still zero hashes")
	}

	// Verify historical root accumulator was appended.
	if len(newS.HistoricalRoots) != 1 {
		t.Errorf("wanted slashed balance %d, got %d", 1, len(newS.HistoricalRoots[ce]))
	}
}
