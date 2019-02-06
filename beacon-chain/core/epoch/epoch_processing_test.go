package epoch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestCanProcessEpoch(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		},
		{
			slot:            63,
			canProcessEpoch: false,
		},
		{
			slot:            64,
			canProcessEpoch: true,
		}, {
			slot:            128,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: true,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessEpoch(state) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessEpoch(state),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestCanProcessEth1Data(t *testing.T) {
	if config.Eth1DataVotingPeriod != 16 {
		t.Errorf("Eth1DataVotingPeriod should be 16 for these tests to pass")
	}
	tests := []struct {
		slot               uint64
		canProcessEth1Data bool
	}{
		{
			slot:               1,
			canProcessEth1Data: false,
		},
		{
			slot:               15,
			canProcessEth1Data: false,
		},
		{
			slot:               16,
			canProcessEth1Data: true,
		},
		{
			slot:               32,
			canProcessEth1Data: true,
		},
		{
			slot:               234234,
			canProcessEth1Data: false,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessEth1Data(state) != tt.canProcessEth1Data {
			t.Errorf(
				"CanProcessEth1Data(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessEth1Data(state),
				tt.canProcessEth1Data,
			)
		}
	}
}

func TestProcessEth1Data(t *testing.T) {
	requiredVoteCount := config.Eth1DataVotingPeriod
	state := &pb.BeaconState{
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: nil,
			BlockHash32:       nil,
		},
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'A'},
					BlockHash32:       []byte{'B'},
				},
				VoteCount: 0,
			},
			// DepositRootHash32 ['B'] gets to process with sufficient vote count.
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'C'},
					BlockHash32:       []byte{'D'},
				},
				VoteCount: requiredVoteCount/2 + 1,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'E'},
					BlockHash32:       []byte{'F'},
				},
				VoteCount: requiredVoteCount / 2,
			},
		},
	}
	newState := ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'C'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'C'}, newState.LatestEth1Data.DepositRootHash32)
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	state.Eth1DataVotes = append(state.Eth1DataVotes,
		&pb.Eth1DataVote{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'G'},
				BlockHash32:       []byte{'H'},
			},
			VoteCount: requiredVoteCount,
		},
	)
	newState = ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'G'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'G'}, newState.LatestEth1Data.DepositRootHash32)
	}

	if len(newState.Eth1DataVotes) != 0 {
		t.Errorf("Failed to clean up Eth1DataVotes slice. Length: %d",
			len(newState.Eth1DataVotes))
	}
}

func TestProcessEth1Data_InactionSlot(t *testing.T) {
	requiredVoteCount := config.Eth1DataVotingPeriod
	state := &pb.BeaconState{
		Slot: 4,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{'A'},
			BlockHash32:       []byte{'B'},
		},
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'C'},
					BlockHash32:       []byte{'D'},
				},
				VoteCount: requiredVoteCount/2 + 1,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'E'},
					BlockHash32:       []byte{'F'},
				},
				VoteCount: requiredVoteCount / 2,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'G'},
					BlockHash32:       []byte{'H'},
				},
				VoteCount: requiredVoteCount,
			},
		},
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	newState := ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'A'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'A'}, newState.LatestEth1Data.DepositRootHash32)
	}
}

func TestProcessJustification(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		Slot:                  300,
		JustifiedSlot:         200,
		JustificationBitfield: 4,
	}
	newState := ProcessJustification(state, 1, 1, 1)

	if newState.PreviousJustifiedSlot != 200 {
		t.Errorf("New state's prev justified slot %d != old state's justified slot %d",
			newState.PreviousJustifiedSlot, state.JustifiedSlot)
	}
	// Since this epoch was justified (not prev), justified_slot = state.slot - EPOCH_LENGTH.
	if newState.JustifiedSlot != state.Slot-config.EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-config.EpochLength)
	}
	// The new JustificationBitfield is 11, it went from 0100 to 1011. Two 1's were appended because both
	// prev epoch and this epoch were justified.
	if newState.JustificationBitfield != 11 {
		t.Errorf("New state's justification bitfield %d != 11", newState.JustificationBitfield)
	}

	// Assume for the case where only prev epoch got justified. Verify
	// justified_slot = state.slot - 2 * EPOCH_LENGTH.
	newState = ProcessJustification(state, 0, 1, 1)
	if newState.JustifiedSlot != state.Slot-2*config.EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - 2 * EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-config.EpochLength)
	}
}

func TestProcessFinalization(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := config.EpochLength

	// 2 consecutive justified slot in a row,
	// and previous justified slot is state slot - 2 * EPOCH_LENGTH.
	state := &pb.BeaconState{
		Slot:                  200,
		JustifiedSlot:         200 - epochLength,
		PreviousJustifiedSlot: 200 - 2*epochLength,
		JustificationBitfield: 3,
	}
	newState := ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// 3 consecutive justified slot in a row.
	// and previous justified slot is state slot - 3 * EPOCH_LENGTH.
	state = &pb.BeaconState{
		Slot:                  300,
		JustifiedSlot:         300 - epochLength,
		PreviousJustifiedSlot: 300 - 3*epochLength,
		JustificationBitfield: 7,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// 4 consecutive justified slot in a row.
	// and previous justified slot is state slot - 3 * EPOCH_LENGTH.
	state = &pb.BeaconState{
		Slot:                  400,
		JustifiedSlot:         400 - epochLength,
		PreviousJustifiedSlot: 400 - 4*epochLength,
		JustificationBitfield: 15,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != state.JustifiedSlot {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			state.JustifiedSlot, newState.FinalizedSlot)
	}

	// if nothing gets finalized it just returns the same state.
	state = &pb.BeaconState{
		Slot:                  100,
		JustifiedSlot:         65,
		PreviousJustifiedSlot: 0,
		JustificationBitfield: 1,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedSlot != 0 {
		t.Errorf("Wanted finalized slot to be %d, got %d:",
			0, newState.FinalizedSlot)
	}
}

func TestProcessCrosslinksOk(t *testing.T) {
	state := buildState(5, config.DepositsForChainStart)
	state.LatestCrosslinks = []*pb.CrosslinkRecord{{}, {}}
	epoch := uint64(5)
	state.Slot = epoch * config.EpochLength

	byteLength := int(config.DepositsForChainStart / config.TargetCommitteeSize / 8)
	var participationBitfield []byte
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 state.Slot,
				ShardBlockRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	newState, err := ProcessCrosslinks(
		state,
		attestations,
		nil,
	)
	if err != nil {
		t.Fatalf("Could not execute ProcessCrosslinks: %v", err)
	}
	// Verify crosslink for shard 0([1]) was processed at state.slot (5).
	if newState.LatestCrosslinks[0].Epoch != epoch {
		t.Errorf("Shard 0s got crosslinked at epoch %d, wanted: %d",
			newState.LatestCrosslinks[0].Epoch, epoch)
	}
	// Verify crosslink for shard 0 was root hashed for []byte{'A'}.
	if !bytes.Equal(newState.LatestCrosslinks[0].ShardBlockRootHash32,
		attestations[0].Data.ShardBlockRootHash32) {
		t.Errorf("Shard 0's root hash is %#x, wanted: %#x",
			newState.LatestCrosslinks[0].ShardBlockRootHash32,
			attestations[0].Data.ShardBlockRootHash32)
	}
}

func TestProcessCrosslinksNoParticipantsBitField(t *testing.T) {
	state := buildState(5, config.DepositsForChainStart)
	state.LatestCrosslinks = []*pb.CrosslinkRecord{{}, {}}

	attestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{},
			// Empty participation bitfield will trigger error.
			AggregationBitfield: []byte{}}}

	wanted := fmt.Sprintf(
		"wanted participants bitfield length %d, got: %d",
		16, 0,
	)
	if _, err := ProcessCrosslinks(state, attestations, nil); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestProcessEjectionsOk(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorBalances: []uint64{
			config.EjectionBalance - 1,
			config.EjectionBalance + 1},
		LatestPenalizedBalances: []uint64{0},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitEpoch: config.FarFutureEpoch},
			{ExitEpoch: config.FarFutureEpoch}},
	}

	state, err := ProcessEjections(state)
	if err != nil {
		t.Fatalf("Could not execute ProcessEjections: %v", err)
	}

	if state.ValidatorRegistry[0].ExitEpoch !=
		config.EntryExitDelay+state.Slot {
		t.Errorf("Expected exit epoch %d, but got %d",
			state.ValidatorRegistry[0].ExitEpoch, config.EntryExitDelay)
	}
	if state.ValidatorRegistry[1].ExitEpoch !=
		config.FarFutureEpoch {
		t.Errorf("Expected exit epoch 0, but got %v", state.ValidatorRegistry[1].ExitEpoch)
	}
}

func TestCanProcessValidatorRegistry(t *testing.T) {
	crosslinks := make([]*pb.CrosslinkRecord, config.DepositsForChainStart)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pb.CrosslinkRecord{
			Epoch: 101,
		}
	}

	state := &pb.BeaconState{
		FinalizedSlot:                100,
		ValidatorRegistryUpdateEpoch: 99,
		LatestCrosslinks:             crosslinks,
	}

	if !CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted True for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestCanNotProcessValidatorRegistry(t *testing.T) {
	state := &pb.BeaconState{
		FinalizedSlot:                100,
		ValidatorRegistryUpdateEpoch: 101,
	}

	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
	state = &pb.BeaconState{
		ValidatorRegistryUpdateEpoch: 101,
		FinalizedSlot:                102,
		LatestCrosslinks: []*pb.CrosslinkRecord{
			{Epoch: 100},
		},
	}
	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestProcessPrevSlotShardOk(t *testing.T) {
	state := &pb.BeaconState{
		CurrentCalculationEpoch: 1,
		CurrentEpochStartShard:  2,
		CurrentEpochSeedHash32:  []byte{'A'},
	}

	newState := ProcessPrevSlotShardSeed(
		proto.Clone(state).(*pb.BeaconState))

	if newState.PreviousCalculationEpoch != state.CurrentCalculationEpoch {
		t.Errorf("Incorret prev epoch calculation slot: Wanted: %d, got: %d",
			newState.PreviousCalculationEpoch, state.CurrentCalculationEpoch)
	}
	if newState.PreviousEpochStartShard != state.CurrentEpochStartShard {
		t.Errorf("Incorret prev epoch start shard: Wanted: %d, got: %d",
			newState.PreviousEpochStartShard, state.CurrentEpochStartShard)
	}
	if !bytes.Equal(newState.PreviousEpochSeedHash32, state.CurrentEpochSeedHash32) {
		t.Errorf("Incorret prev epoch randao mix hash: Wanted: %v, got: %v",
			state.CurrentEpochSeedHash32, newState.PreviousEpochSeedHash32)
	}
}

func TestProcessValidatorRegistryOk(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                     config.SeedLookahead,
		LatestRandaoMixesHash32S: [][]byte{{'A'}, {'B'}},
		CurrentEpochSeedHash32:   []byte{'C'},
	}
	newState, err := ProcessValidatorRegistry(
		proto.Clone(state).(*pb.BeaconState))
	if err != nil {
		t.Fatalf("Could not execute ProcessValidatorRegistry: %v", err)
	}
	if newState.CurrentCalculationEpoch != state.Slot {
		t.Errorf("Incorret curr epoch calculation slot: Wanted: %d, got: %d",
			newState.CurrentCalculationEpoch, state.Slot)
	}
	if !bytes.Equal(newState.CurrentEpochSeedHash32, state.LatestRandaoMixesHash32S[0]) {
		t.Errorf("Incorret current epoch randao mix hash: Wanted: %v, got: %v",
			state.LatestRandaoMixesHash32S[0], newState.CurrentEpochSeedHash32)
	}
}

func TestProcessPartialValidatorRegistry(t *testing.T) {
	offset := uint64(1)
	state := &pb.BeaconState{
		Slot:                         config.SeedLookahead + offset,
		ValidatorRegistryUpdateEpoch: offset,
		LatestRandaoMixesHash32S:     [][]byte{{'A'}, {'B'}},
	}
	copiedState := proto.Clone(state).(*pb.BeaconState)
	newState := ProcessPartialValidatorRegistry(copiedState)
	if newState.CurrentCalculationEpoch != state.Slot {
		t.Errorf("Incorrect CurrentCalculationEpoch, wanted: %d, got: %d",
			state.Slot, newState.CurrentCalculationEpoch)
	}
	if !bytes.Equal(newState.CurrentEpochSeedHash32, state.LatestRandaoMixesHash32S[offset]) {
		t.Errorf("Incorret current epoch randao mix hash: Wanted: %v, got: %v",
			state.LatestRandaoMixesHash32S[offset], newState.CurrentEpochSeedHash32)
	}
}

func TestCleanupAttestations(t *testing.T) {
	if config.EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := config.EpochLength
	state := &pb.BeaconState{
		Slot: epochLength,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Slot: 1}},
			{Data: &pb.AttestationData{Slot: epochLength - 10}},
			{Data: &pb.AttestationData{Slot: epochLength}},
			{Data: &pb.AttestationData{Slot: epochLength + 1}},
			{Data: &pb.AttestationData{Slot: epochLength + 20}},
			{Data: &pb.AttestationData{Slot: 32}},
			{Data: &pb.AttestationData{Slot: 33}},
			{Data: &pb.AttestationData{Slot: 2 * epochLength}},
		},
	}
	wanted := &pb.BeaconState{
		Slot: epochLength,
		LatestAttestations: []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Slot: epochLength}},
			{Data: &pb.AttestationData{Slot: epochLength + 1}},
			{Data: &pb.AttestationData{Slot: epochLength + 20}},
			{Data: &pb.AttestationData{Slot: 2 * epochLength}},
		},
	}
	newState := CleanupAttestations(state)

	if !reflect.DeepEqual(newState, wanted) {
		t.Errorf("Wanted state: %v, got state: %v ",
			wanted, newState)
	}
}

func TestUpdatePenalizedExitBalances(t *testing.T) {
	tests := []struct {
		slot     uint64
		balances uint64
	}{
		{
			slot:     0,
			balances: 100,
		},
		{
			slot:     config.LatestPenalizedExitLength,
			balances: 324,
		},
		{
			slot:     config.LatestPenalizedExitLength + 1,
			balances: 234324,
		}, {
			slot:     config.LatestPenalizedExitLength * 100,
			balances: 34,
		}, {
			slot:     config.LatestPenalizedExitLength * 1000,
			balances: 1,
		},
	}
	for _, tt := range tests {
		epoch := (tt.slot / config.EpochLength) % config.LatestPenalizedExitLength
		latestPenalizedExitBalances := make([]uint64,
			config.LatestPenalizedExitLength)
		latestPenalizedExitBalances[epoch] = tt.balances
		state := &pb.BeaconState{
			Slot:                    tt.slot,
			LatestPenalizedBalances: latestPenalizedExitBalances}
		newState := UpdatePenalizedExitBalances(state)
		if newState.LatestPenalizedBalances[epoch+1] !=
			tt.balances {
			t.Errorf(
				"LatestPenalizedBalances didn't update for epoch %d,"+
					"wanted: %d, got: %d", epoch+1, tt.balances,
				newState.LatestPenalizedBalances[epoch+1],
			)
		}
	}
}
