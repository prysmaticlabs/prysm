package epoch

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestCanProcessEpoch(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
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

func TestCanProcessReceiptRoots(t *testing.T) {
	if params.BeaconConfig().PowReceiptRootVotingPeriod != 1024 {
		t.Errorf("PowReceiptRootVotingPeriod should be 1024 for these tests to pass")
	}
	tests := []struct {
		slot                   uint64
		canProcessReceiptRoots bool
	}{
		{
			slot:                   1,
			canProcessReceiptRoots: false,
		},
		{
			slot:                   1022,
			canProcessReceiptRoots: false,
		},
		{
			slot:                   1024,
			canProcessReceiptRoots: true,
		}, {
			slot:                   4096,
			canProcessReceiptRoots: true,
		}, {
			slot:                   234234,
			canProcessReceiptRoots: false,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessReceiptRoots(state) != tt.canProcessReceiptRoots {
			t.Errorf(
				"CanProcessReceiptRoots(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessReceiptRoots(state),
				tt.canProcessReceiptRoots,
			)
		}
	}
}

func TestProcessReceipt(t *testing.T) {
	if params.BeaconConfig().PowReceiptRootVotingPeriod != 1024 {
		t.Errorf("PowReceiptRootVotingPeriod should be 1024 for these tests to pass")
	}
	requiredVoteCount := params.BeaconConfig().PowReceiptRootVotingPeriod
	state := &pb.BeaconState{
		CandidatePowReceiptRoots: []*pb.CandidatePoWReceiptRootRecord{
			{VoteCount: 0, CandidatePowReceiptRootHash32: []byte{'A'}},
			// CandidatePowReceiptRootHash32 ['B'] gets to process with sufficient vote count.
			{VoteCount: requiredVoteCount/2 + 1, CandidatePowReceiptRootHash32: []byte{'B'}},
			{VoteCount: requiredVoteCount / 2, CandidatePowReceiptRootHash32: []byte{'C'}},
		},
	}
	newState := ProcessReceipt(state)
	if !bytes.Equal(newState.ProcessedPowReceiptRootHash32, []byte{'B'}) {
		t.Errorf("Incorrect ProcessedPowReceiptRootHash32. Wanted: %v, got: %v",
			[]byte{'B'}, newState.ProcessedPowReceiptRootHash32)
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	state.CandidatePowReceiptRoots = append(state.CandidatePowReceiptRoots,
		&pb.CandidatePoWReceiptRootRecord{VoteCount: requiredVoteCount,
			CandidatePowReceiptRootHash32: []byte{'D'}})
	newState = ProcessReceipt(state)
	if !bytes.Equal(newState.ProcessedPowReceiptRootHash32, []byte{'D'}) {
		t.Errorf("Incorrect ProcessedPowReceiptRootHash32. Wanted: %v, got: %v",
			[]byte{'D'}, newState.ProcessedPowReceiptRootHash32)
	}

	if len(newState.CandidatePowReceiptRoots) != 0 {
		t.Errorf("Failed to clean up CandidatePowReceiptRoots slice. Length: %d",
			len(newState.CandidatePowReceiptRoots))
	}
}

func TestProcessJustification(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
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
	if newState.JustifiedSlot != state.Slot-params.BeaconConfig().EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-params.BeaconConfig().EpochLength)
	}
	// The new JustificationBitfield is 11, it went from 0100 to 1011. Two 1's were appended because both
	// prev epoch and this epoch were justified.
	if newState.JustificationBitfield != 11 {
		t.Errorf("New state's justification bitfield %d != 11", newState.JustificationBitfield)
	}

	// Assume for the case where only prev epoch got justified. Verify
	// justified_slot = state.slot - 2 * EPOCH_LENGTH.
	newState = ProcessJustification(state, 0, 1, 1)
	if newState.JustifiedSlot != state.Slot-2*params.BeaconConfig().EpochLength {
		t.Errorf("New state's justified slot %d != state's slot - 2 * EPOCH_LENGTH %d",
			newState.JustifiedSlot, state.Slot-params.BeaconConfig().EpochLength)
	}
}

func TestProcessFinalization(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := params.BeaconConfig().EpochLength

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

func TestProcessCrosslinks_Ok(t *testing.T) {
	shardCommitteesAtSlot := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardCommitteesAtSlot,
		Slot:                      5,
		LatestCrosslinks:          []*pb.CrosslinkRecord{{}, {}},
		ValidatorBalances: []uint64{16 * 1e9, 18 * 1e9, 20 * 1e9, 31 * 1e9,
			32 * 1e9, 34 * 1e9, 50 * 1e9, 50 * 1e9},
	}

	var attestations []*pb.PendingAttestationRecord
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestationRecord{
			Data: &pb.AttestationData{
				Slot:                 0,
				Shard:                1,
				ShardBlockRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			ParticipationBitfield: []byte{0xff},
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
	// Verify crosslink for shard 1([1]) was processed at state.slot (5).
	if newState.LatestCrosslinks[1].Slot != state.Slot {
		t.Errorf("Shard 0s got crosslinked at slot %d, wanted: %d",
			newState.LatestCrosslinks[1].Slot, state.Slot)
	}
	// Verify crosslink for shard 1 was root hashed for []byte{'A'}.
	if !bytes.Equal(newState.LatestCrosslinks[1].ShardBlockRootHash32,
		attestations[0].Data.ShardBlockRootHash32) {
		t.Errorf("Shard 0's root hash is %#x, wanted: %#x",
			newState.LatestCrosslinks[1].ShardBlockRootHash32,
			attestations[0].Data.ShardBlockRootHash32)
	}
}

func TestProcessCrosslinks_NoRoot(t *testing.T) {
	shardCommitteesAtSlot := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	state := &pb.BeaconState{
		ShardAndCommitteesAtSlots: shardCommitteesAtSlot,
		Slot:                      5,
		LatestCrosslinks:          []*pb.CrosslinkRecord{{}, {}},
		ValidatorBalances:         []uint64{},
	}

	attestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1},
			// Empty participation bitfield will trigger error.
			ParticipationBitfield: []byte{}}}

	_, err := ProcessCrosslinks(state, attestations, nil)
	if err == nil {
		t.Fatalf("ProcessCrosslinks should have failed")
	}
}

func TestProcessEjections_Ok(t *testing.T) {
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}
	state := &pb.BeaconState{
		Slot:                      1,
		ShardAndCommitteesAtSlots: shardAndCommittees,
		ValidatorBalances: []uint64{
			params.BeaconConfig().EjectionBalanceInGwei - 1,
			params.BeaconConfig().EjectionBalanceInGwei + 1},
		LatestPenalizedExitBalances: []uint64{0},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().FarFutureSlot},
			{ExitSlot: params.BeaconConfig().FarFutureSlot}},
	}
	state, err := ProcessEjections(state)
	if err != nil {
		t.Fatalf("Could not execute ProcessEjections: %v", err)
	}
	if state.ValidatorRegistry[0].ExitSlot !=
		params.BeaconConfig().EntryExitDelay+state.Slot {
		t.Errorf("Expected exit slot %d, but got %d",
			state.ValidatorRegistry[0].ExitSlot, params.BeaconConfig().EntryExitDelay)
	}
	if state.ValidatorRegistry[1].ExitSlot !=
		params.BeaconConfig().FarFutureSlot {
		t.Errorf("Expected exit slot 0, but got %v", state.ValidatorRegistry[1].ExitSlot)
	}
}

func TestCanProcessValidatorRegistry(t *testing.T) {
	state := &pb.BeaconState{
		FinalizedSlot:                   100,
		ValidatorRegistryLastChangeSlot: 99,
		LatestCrosslinks: []*pb.CrosslinkRecord{
			{Slot: 101}, {Slot: 102}, {Slot: 103}, {Slot: 104},
		},
		ShardAndCommitteesAtSlots: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 0}, {Shard: 1}, {Shard: 2}, {Shard: 3},
			}},
		},
	}
	if !CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted True for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestCanNotProcessValidatorRegistry(t *testing.T) {
	state := &pb.BeaconState{
		FinalizedSlot:                   100,
		ValidatorRegistryLastChangeSlot: 101,
	}
	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
	state = &pb.BeaconState{
		FinalizedSlot:                   100,
		ValidatorRegistryLastChangeSlot: 99,
		LatestCrosslinks: []*pb.CrosslinkRecord{
			{Slot: 99},
		},
		ShardAndCommitteesAtSlots: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 0}},
			}},
	}
	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestProcessValidatorRegistry(t *testing.T) {
	epochLength := params.BeaconConfig().EpochLength
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, epochLength*2)
	for i := 0; i < len(shardAndCommittees); i++ {
		shardAndCommittees[i] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{{Shard: uint64(i)}},
		}
	}

	state := &pb.BeaconState{
		Slot:                            64,
		ValidatorRegistryLastChangeSlot: 1,
		ShardAndCommitteesAtSlots:       shardAndCommittees,
		LatestRandaoMixesHash32S:        [][]byte{{'A'}},
	}
	copiedState := proto.Clone(state).(*pb.BeaconState)
	newState, err := ProcessValidatorRegistry(copiedState)
	if err != nil {
		t.Fatalf("Could not execute ProcessValidatorRegistry: %v", err)
	}

	if newState.ShardAndCommitteesAtSlots[0].ArrayShardAndCommittee[0].Shard != state.ShardAndCommitteesAtSlots[epochLength].ArrayShardAndCommittee[0].Shard {
		t.Errorf("Incorrect rotation for shard committees, wanted shard: %d, got shard: %d",
			state.ShardAndCommitteesAtSlots[0].ArrayShardAndCommittee[0].Shard,
			newState.ShardAndCommitteesAtSlots[epochLength].ArrayShardAndCommittee[0].Shard)
	}
}

func TestProcessValidatorRegistry_ReachedUpperBound(t *testing.T) {
	epochLength := params.BeaconConfig().EpochLength
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, epochLength*2)
	for i := 0; i < len(shardAndCommittees); i++ {
		shardAndCommittees[i] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{{Shard: uint64(i)}},
		}
	}
	validators := make([]*pb.ValidatorRecord, 1<<params.BeaconConfig().MaxNumLog2Validators-1)
	validator := &pb.ValidatorRecord{ExitSlot: params.BeaconConfig().FarFutureSlot}
	for i := 0; i < len(validators); i++ {
		validators[i] = validator
	}
	state := &pb.BeaconState{
		Slot:                            64,
		ValidatorRegistryLastChangeSlot: 1,
		ShardAndCommitteesAtSlots:       shardAndCommittees,
		LatestRandaoMixesHash32S:        [][]byte{{'A'}},
		ValidatorRegistry:               validators,
	}

	if _, err := ProcessValidatorRegistry(state); err == nil {
		t.Fatalf("ProcessValidatorRegistry should have failed with upperbound")
	}
}

func TestProcessPartialValidatorRegistry(t *testing.T) {
	epochLength := params.BeaconConfig().EpochLength
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, epochLength*2)
	for i := 0; i < len(shardAndCommittees); i++ {
		shardAndCommittees[i] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{{Shard: uint64(i)}},
		}
	}

	state := &pb.BeaconState{
		Slot:                            64,
		ValidatorRegistryLastChangeSlot: 1,
		ShardAndCommitteesAtSlots:       shardAndCommittees,
		LatestRandaoMixesHash32S:        [][]byte{{'A'}},
	}
	copiedState := proto.Clone(state).(*pb.BeaconState)
	newState, err := ProcessPartialValidatorRegistry(copiedState)
	if err != nil {
		t.Fatalf("Could not execute ProcessValidatorRegistryNoUpdate: %v", err)
	}
	if newState.ValidatorRegistryLastChangeSlot != state.ValidatorRegistryLastChangeSlot {
		t.Errorf("Incorrect ValidatorRegistryLastChangeSlot, wanted: %d, got: %d",
			state.ValidatorRegistryLastChangeSlot, newState.ValidatorRegistryLastChangeSlot)
	}

	if newState.ShardAndCommitteesAtSlots[0].ArrayShardAndCommittee[0].Shard != state.ShardAndCommitteesAtSlots[epochLength].ArrayShardAndCommittee[0].Shard {
		t.Errorf("Incorrect rotation for shard committees, wanted shard: %d, got shard: %d",
			state.ShardAndCommitteesAtSlots[0].ArrayShardAndCommittee[0].Shard,
			newState.ShardAndCommitteesAtSlots[epochLength].ArrayShardAndCommittee[0].Shard)
	}
}

func TestProcessPartialValidatorRegistry_ReachedUpperBound(t *testing.T) {
	epochLength := params.BeaconConfig().EpochLength
	shardAndCommittees := make([]*pb.ShardAndCommitteeArray, epochLength*2)
	for i := 0; i < len(shardAndCommittees); i++ {
		shardAndCommittees[i] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{{Shard: uint64(i)}},
		}
	}
	validators := make([]*pb.ValidatorRecord, 1<<params.BeaconConfig().MaxNumLog2Validators-1)
	validator := &pb.ValidatorRecord{ExitSlot: params.BeaconConfig().FarFutureSlot}
	for i := 0; i < len(validators); i++ {
		validators[i] = validator
	}
	state := &pb.BeaconState{
		Slot:                            64,
		ValidatorRegistryLastChangeSlot: 1,
		ShardAndCommitteesAtSlots:       shardAndCommittees,
		LatestRandaoMixesHash32S:        [][]byte{{'A'}},
		ValidatorRegistry:               validators,
	}

	if _, err := ProcessPartialValidatorRegistry(state); err == nil {
		t.Fatalf("ProcessValidatorRegistry should have failed with upperbound")
	}
}

func TestCleanupAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := params.BeaconConfig().EpochLength
	state := &pb.BeaconState{
		Slot: 2 * epochLength,
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
		Slot: 2 * epochLength,
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
