package validators

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.Attestation{
		ParticipationBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.ParticipationBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.ParticipationBitfield, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d with : %v", i, err)
		}

		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 01010101.
	pendingAttestation = &pb.Attestation{
		ParticipationBitfield: []byte{85},
	}

	for i := 0; i < len(pendingAttestation.ParticipationBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.ParticipationBitfield, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d : %v", i, err)
		}

		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestInitialValidatorRegistry(t *testing.T) {
	validators := InitialValidatorRegistry()
	for idx, validator := range validators {
		if !isActiveValidator(validator, 1) {
			t.Errorf("validator %d status is not active", idx)
		}
	}
}

func TestProposerShardAndIndex(t *testing.T) {
	state := &pb.BeaconState{
		ShardCommitteesAtSlots: []*pb.ShardCommitteeArray{
			{ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4}},
				{Shard: 1, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 2, Committee: []uint32{10, 11, 12, 13, 14}},
				{Shard: 3, Committee: []uint32{15, 16, 17, 18, 19}},
			}},
			{ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 4, Committee: []uint32{20, 21, 22, 23, 24}},
				{Shard: 5, Committee: []uint32{25, 26, 27, 28, 29}},
			}},
		}}

	if _, _, err := ProposerShardAndIdx(state, 150); err == nil {
		t.Error("ProposerShardAndIdx should have failed with invalid lcs")
	}
	shard, idx, err := ProposerShardAndIdx(state, 2)
	if err != nil {
		t.Fatalf("ProposerShardAndIdx failed with %v", err)
	}
	if shard != 4 {
		t.Errorf("Invalid shard ID. Wanted 4, got %d", shard)
	}
	if idx != 2 {
		t.Errorf("Invalid proposer index. Wanted 2, got %d", idx)
	}
}

func TestValidatorIdx(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, ExitSlot: params.BeaconConfig().FarFutureSlot})
	}
	if _, err := ValidatorIdx([]byte("100"), validators); err == nil {
		t.Fatalf("ValidatorIdx should have failed,  there's no validator with pubkey 100")
	}
	validators[5].Pubkey = []byte("100")
	idx, err := ValidatorIdx([]byte("100"), validators)
	if err != nil {
		t.Fatalf("call ValidatorIdx failed: %v", err)
	}
	if idx != 5 {
		t.Errorf("Incorrect validator index. Wanted 5, Got %v", idx)
	}
}

func TestValidatorShard(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 21; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, ExitSlot: params.BeaconConfig().FarFutureSlot})
	}
	shardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{Shard: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{Shard: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
	}
	validators[19].Pubkey = []byte("100")
	Shard, err := ValidatorShardID([]byte("100"), validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorShard failed: %v", err)
	}
	if Shard != 2 {
		t.Errorf("Incorrect validator shard ID. Wanted 2, Got %v", Shard)
	}

	validators[19].Pubkey = []byte{}
	if _, err := ValidatorShardID([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShard should have failed, there's no validator with pubkey 100")
	}

	validators[20].Pubkey = []byte("100")
	if _, err := ValidatorShardID([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorShard should have failed, validator indexed at 20 is not in the committee")
	}
}

func TestValidatorSlotAndResponsibility(t *testing.T) {
	var validators []*pb.ValidatorRecord
	for i := 0; i < 61; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{}, ExitSlot: params.BeaconConfig().FarFutureSlot})
	}
	shardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 0, Committee: []uint32{0, 1, 2, 3, 4, 5, 6}},
			{Shard: 1, Committee: []uint32{7, 8, 9, 10, 11, 12, 13}},
			{Shard: 2, Committee: []uint32{14, 15, 16, 17, 18, 19}},
		}},
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 3, Committee: []uint32{20, 21, 22, 23, 24, 25, 26}},
			{Shard: 4, Committee: []uint32{27, 28, 29, 30, 31, 32, 33}},
			{Shard: 5, Committee: []uint32{34, 35, 36, 37, 38, 39}},
		}},
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 6, Committee: []uint32{40, 41, 42, 43, 44, 45, 46}},
			{Shard: 7, Committee: []uint32{47, 48, 49, 50, 51, 52, 53}},
			{Shard: 8, Committee: []uint32{54, 55, 56, 57, 58, 59}},
		}},
	}
	if _, _, err := ValidatorSlotAndRole([]byte("100"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, there's no validator with pubkey 100")
	}

	validators[59].Pubkey = []byte("100")
	slot, _, err := ValidatorSlotAndRole([]byte("100"), validators, shardCommittees)
	if err != nil {
		t.Fatalf("call ValidatorSlot failed: %v", err)
	}
	if slot != 2 {
		t.Errorf("Incorrect validator slot ID. Wanted 1, Got %v", slot)
	}

	validators[60].Pubkey = []byte("101")
	if _, _, err := ValidatorSlotAndRole([]byte("101"), validators, shardCommittees); err == nil {
		t.Fatalf("ValidatorSlot should have failed, validator indexed at 60 is not in the committee")
	}
}

func TestShardCommitteesAtSlot_OK(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: i},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	tests := []struct {
		slot          uint64
		stateSlot     uint64
		expectedShard uint64
	}{
		{
			slot:          0,
			stateSlot:     0,
			expectedShard: 0,
		},
		{
			slot:          1,
			stateSlot:     5,
			expectedShard: 1,
		},
		{
			stateSlot:     1024,
			slot:          1024,
			expectedShard: 64 - 0,
		}, {
			stateSlot:     2048,
			slot:          2000,
			expectedShard: 64 - 48,
		}, {
			stateSlot:     2048,
			slot:          2058,
			expectedShard: 64 + 10,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot

		result, err := ShardCommitteesAtSlot(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}

		if result.ArrayShardCommittee[0].Shard != tt.expectedShard {
			t.Errorf(
				"Result shard was an unexpected value. Wanted %d, got %d",
				tt.expectedShard,
				result.ArrayShardCommittee[0].Shard,
			)
		}
	}
}

func TestShardCommitteesAtSlot_OutOfBounds(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().EpochLength,
	}

	tests := []struct {
		expectedErr string
		slot        uint64
	}{
		{
			expectedErr: "slot 5000 out of bounds: 0 <= slot < 128",
			slot:        5000,
		},
		{
			expectedErr: "slot 129 out of bounds: 0 <= slot < 128",
			slot:        129,
		},
	}

	for _, tt := range tests {
		_, err := ShardCommitteesAtSlot(state, tt.slot)
		if err != nil && err.Error() != tt.expectedErr {
			t.Fatalf("Expected error \"%s\" got \"%v\"", tt.expectedErr, err)
		}

	}
}

func TestEffectiveBalance(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDeposit * params.BeaconConfig().Gwei

	tests := []struct {
		a uint64
		b uint64
	}{
		{a: 0, b: 0},
		{a: defaultBalance - 1, b: defaultBalance - 1},
		{a: defaultBalance, b: defaultBalance},
		{a: defaultBalance + 1, b: defaultBalance},
		{a: defaultBalance * 100, b: defaultBalance},
	}
	for _, test := range tests {
		state := &pb.BeaconState{ValidatorBalances: []uint64{test.a}}
		if EffectiveBalance(state, 0) != test.b {
			t.Errorf("EffectiveBalance(%d) = %d, want = %d", test.a, EffectiveBalance(state, 0), test.b)
		}
	}
}

func TestTotalEffectiveBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		27 * 1e9, 28 * 1e9, 32 * 1e9, 40 * 1e9,
	}}

	// 27 + 28 + 32 + 32 = 119
	if TotalEffectiveBalance(state, []uint32{0, 1, 2, 3}) != 119*1e9 {
		t.Errorf("Incorrect TotalEffectiveBalance. Wanted: 119, got: %d",
			TotalEffectiveBalance(state, []uint32{0, 1, 2, 3})/1e9)
	}
}

func TestIsActiveValidator(t *testing.T) {

	tests := []struct {
		a uint64
		b bool
	}{
		{a: 0, b: false},
		{a: 10, b: true},
		{a: 100, b: false},
		{a: 1000, b: false},
		{a: 64, b: true},
	}
	for _, test := range tests {
		validator := &pb.ValidatorRecord{ActivationSlot: 10, ExitSlot: 100}
		if isActiveValidator(validator, test.a) != test.b {
			t.Errorf("isActiveValidator(%d) = %v, want = %v",
				test.a, isActiveValidator(validator, test.a), test.b)
		}
	}
}

func TestGetActiveValidatorRecord(t *testing.T) {
	inputValidators := []*pb.ValidatorRecord{
		{ExitCount: 0},
		{ExitCount: 1},
		{ExitCount: 2},
		{ExitCount: 3},
		{ExitCount: 4},
	}

	outputValidators := []*pb.ValidatorRecord{
		{ExitCount: 1},
		{ExitCount: 3},
	}

	state := &pb.BeaconState{
		ValidatorRegistry: inputValidators,
	}

	validators := ActiveValidator(state, []uint32{1, 3})

	if !reflect.DeepEqual(outputValidators, validators) {
		t.Errorf("Active validators don't match. Wanted: %v, Got: %v", outputValidators, validators)
	}
}

func TestBoundaryAttestingBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		25 * 1e9, 26 * 1e9, 32 * 1e9, 33 * 1e9, 100 * 1e9,
	}}

	attestedBalances := AttestingBalance(state, []uint32{0, 1, 2, 3, 4})

	// 25 + 26 + 32 + 32 + 32 = 147
	if attestedBalances != 147*1e9 {
		t.Errorf("Incorrect attested balances. Wanted: %f, got: %d", 147*1e9, attestedBalances)
	}
}

func TestBoundaryAttesters(t *testing.T) {
	var validators []*pb.ValidatorRecord

	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.ValidatorRecord{Pubkey: []byte{byte(i)}})
	}

	state := &pb.BeaconState{ValidatorRegistry: validators}

	boundaryAttesters := Attesters(state, []uint32{5, 2, 87, 42, 99, 0})

	expectedBoundaryAttesters := []*pb.ValidatorRecord{
		{Pubkey: []byte{byte(5)}},
		{Pubkey: []byte{byte(2)}},
		{Pubkey: []byte{byte(87)}},
		{Pubkey: []byte{byte(42)}},
		{Pubkey: []byte{byte(99)}},
		{Pubkey: []byte{byte(0)}},
	}

	if !reflect.DeepEqual(expectedBoundaryAttesters, boundaryAttesters) {
		t.Errorf("Incorrect boundary attesters. Wanted: %v, got: %v", expectedBoundaryAttesters, boundaryAttesters)
	}
}

func TestBoundaryAttesterIndices(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}
	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: 100, Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
	}

	boundaryAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{'F'}}, // returns indices 1,5,6
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{3}},   // returns indices 6,7
		{Data: &pb.AttestationData{Slot: 2, Shard: 100}, ParticipationBitfield: []byte{'A'}}, // returns indices 1,7
	}

	attesterIndices, err := ValidatorIndices(state, boundaryAttestations)
	if err != nil {
		t.Fatalf("Failed to run BoundaryAttesterIndices: %v", err)
	}

	if !reflect.DeepEqual(attesterIndices, []uint32{1, 5, 6, 7}) {
		t.Errorf("Incorrect boundary attester indices. Wanted: %v, got: %v", []uint32{1, 5, 6, 7}, attesterIndices)
	}
}

func TestBeaconProposerIdx(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Committee: []uint32{9, 8, 311, 12, 92, 1, 23, 17}},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
	}

	tests := []struct {
		slot uint64
		idx  uint32
	}{
		{
			slot: 1,
			idx:  8,
		},
		{
			slot: 10,
			idx:  311,
		},
		{
			slot: 19,
			idx:  12,
		},
		{
			slot: 30,
			idx:  23,
		},
		{
			slot: 39,
			idx:  17,
		},
	}

	for _, tt := range tests {
		result, err := BeaconProposerIdx(state, tt.slot)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}

		if result != tt.idx {
			t.Errorf(
				"Result index was an unexpected value. Wanted %d, got %d",
				tt.idx,
				result,
			)
		}
	}
}

func TestAttestingValidatorIndices_Ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	var committeeIndices []uint32
	for i := uint32(0); i < 8; i++ {
		committeeIndices = append(committeeIndices, i)
	}

	var ShardCommittees []*pb.ShardCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		ShardCommittees = append(ShardCommittees, &pb.ShardCommitteeArray{
			ArrayShardCommittee: []*pb.ShardCommittee{
				{Shard: i, Committee: committeeIndices},
			},
		})
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
	}

	prevAttestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 3,
			Shard:                3,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	thisAttestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 3,
			Shard:                3,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'F'}, // 01000110 = 1,5,6
	}

	indices, err := AttestingValidatorIndices(
		state,
		ShardCommittees[3].ArrayShardCommittee[0],
		[]byte{'B'},
		[]*pb.PendingAttestationRecord{thisAttestation},
		[]*pb.PendingAttestationRecord{prevAttestation})
	if err != nil {
		t.Fatalf("could not execute AttestingValidatorIndices: %v", err)
	}

	// Union(1,7,1,5,6) = 1,5,6,7
	if !reflect.DeepEqual(indices, []uint32{1, 5, 6, 7}) {
		t.Errorf("could not get incorrect validator indices. Wanted: %v, got: %v",
			[]uint32{1, 5, 6, 7}, indices)
	}
}

func TestAttestingValidatorIndices_OutOfBound(t *testing.T) {
	ShardCommittees := []*pb.ShardCommitteeArray{
		{ArrayShardCommittee: []*pb.ShardCommittee{
			{Shard: 1},
		}},
	}

	state := &pb.BeaconState{
		ShardCommitteesAtSlots: ShardCommittees,
		Slot:                   5,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 0,
			Shard:                1,
			ShardBlockRootHash32: []byte{'B'},
		},
		ParticipationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	_, err := AttestingValidatorIndices(
		state,
		ShardCommittees[0].ArrayShardCommittee[0],
		[]byte{'B'},
		[]*pb.PendingAttestationRecord{attestation},
		nil)

	// This will fail because participation bitfield is length:1, committee bitfield is length 0.
	if err == nil {
		t.Fatal("AttestingValidatorIndices should have failed with incorrect bitfield")
	}
}

func TestAllValidatorIndices(t *testing.T) {
	tests := []struct {
		registries []*pb.ValidatorRecord
		indices    []uint32
	}{
		{registries: []*pb.ValidatorRecord{}, indices: []uint32{}},
		{registries: []*pb.ValidatorRecord{{}}, indices: []uint32{0}},
		{registries: []*pb.ValidatorRecord{{}, {}, {}, {}}, indices: []uint32{0, 1, 2, 3}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{ValidatorRegistry: tt.registries}
		if !reflect.DeepEqual(AllValidatorsIndices(state), tt.indices) {
			t.Errorf("AllValidatorsIndices(%v) = %v, wanted:%v",
				tt.registries, AllValidatorsIndices(state), tt.indices)
		}
	}
}

func TestNewRegistryDeltaChainTip(t *testing.T) {
	tests := []struct {
		flag                         uint64
		idx                          uint32
		pubKey                       []byte
		currentRegistryDeltaChainTip []byte
		newRegistryDeltaChainTip     []byte
	}{
		{0, 100, []byte{'A'}, []byte{'B'},
			[]byte{35, 123, 149, 41, 92, 226, 26, 73, 96, 40, 4, 219, 59, 254, 27,
				38, 220, 125, 83, 177, 78, 12, 187, 74, 72, 115, 64, 91, 16, 144, 37, 245}},
		{2, 64, []byte{'Y'}, []byte{'Z'},
			[]byte{69, 192, 214, 2, 37, 19, 40, 60, 179, 83, 79, 158, 211, 247, 151,
				7, 240, 82, 41, 37, 251, 149, 221, 37, 22, 151, 204, 234, 64, 69, 7, 166}},
	}
	for _, tt := range tests {
		newChainTip, err := NewRegistryDeltaChainTip(
			pb.ValidatorRegistryDeltaBlock_ValidatorRegistryDeltaFlags(tt.flag),
			tt.idx,
			0,
			tt.pubKey,
			tt.currentRegistryDeltaChainTip,
		)
		if err != nil {
			t.Fatalf("could not execute NewRegistryDeltaChainTip:%v", err)
		}
		if !bytes.Equal(newChainTip[:], tt.newRegistryDeltaChainTip) {
			t.Errorf("Incorrect new chain tip. Wanted %#x, got %#x",
				tt.newRegistryDeltaChainTip, newChainTip[:])
		}
	}
}

func TestProcessDeposit_PublicKeyExistsBadWithdrawalCredentials(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{0},
		},
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	want := "expected withdrawal credentials to match"
	if _, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestProcessDeposit_PublicKeyExistsGoodWithdrawalCredentials(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{0, 0}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.ValidatorBalances[1] != 1000 {
		t.Errorf("Expected balance at index 1 to be 1000, received %d", newState.ValidatorBalances[1])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistNoEmptyValidator(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                      []byte{1, 2, 3},
			WithdrawalCredentialsHash32: []byte{2},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{1000, 1000}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 3 {
		t.Errorf("Expected validator balances list to increase by 1, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[2] != 2000 {
		t.Errorf("Expected new validator have balance of %d, received %d", 2000, newState.ValidatorBalances[2])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistEmptyValidatorExists(t *testing.T) {
	registry := []*pb.ValidatorRecord{
		{
			Pubkey:                      []byte{1, 2, 3},
			WithdrawalCredentialsHash32: []byte{2},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{0, 1000}
	beaconState := &pb.BeaconState{
		Slot:              params.BeaconConfig().EpochLength,
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}
	pocCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
		pocCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 3 {
		t.Errorf("Expected validator balances list to be 3, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[len(newState.ValidatorBalances)-1] != 2000 {
		t.Errorf("Expected validator at last index to have balance of %d, received %d", 2000, newState.ValidatorBalances[0])
	}
}

func TestActivateValidatorGenesis_Ok(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, true)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	if newState.ValidatorRegistry[0].ActivationSlot != params.BeaconConfig().GenesisSlot {
		t.Errorf("Wanted activation slot = genesis slot, got %d",
			newState.ValidatorRegistry[0].ActivationSlot)
	}
}

func TestActivateValidator_Ok(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                                 100,
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, false)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	if newState.ValidatorRegistry[0].ActivationSlot !=
		state.Slot+params.BeaconConfig().EntryExitDelay {
		t.Errorf("Wanted activation slot = %d, got %d",
			state.Slot+params.BeaconConfig().EntryExitDelay,
			newState.ValidatorRegistry[0].ActivationSlot)
	}
}

func TestInitiateValidatorExit_Ok(t *testing.T) {
	state := &pb.BeaconState{ValidatorRegistry: []*pb.ValidatorRecord{{}, {}, {}}}
	newState := InitiateValidatorExit(state, 2)
	if newState.ValidatorRegistry[0].StatusFlags != pb.ValidatorRecord_INITIAL {
		t.Errorf("Wanted flag INITIAL, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
	if newState.ValidatorRegistry[2].StatusFlags != pb.ValidatorRecord_INITIATED_EXIT {
		t.Errorf("Wanted flag ACTIVE_PENDING_EXIT, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
}

func TestExitValidator_Ok(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                                 100,
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
		LatestPenalizedExitBalances:          []uint64{0},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().FarFutureSlot, Pubkey: []byte{'B'}},
		},
	}
	newState, err := ExitValidator(state, 0)
	if err != nil {
		t.Fatalf("could not execute ExitValidator:%v", err)
	}

	if newState.ValidatorRegistry[0].ExitSlot !=
		state.Slot+params.BeaconConfig().EntryExitDelay {
		t.Errorf("Wanted exit slot %d, got %d",
			state.Slot+params.BeaconConfig().EntryExitDelay,
			newState.ValidatorRegistry[0].ExitSlot)
	}
	if newState.ValidatorRegistry[0].ExitCount != 1 {
		t.Errorf("Wanted exit count 1, got %d", newState.ValidatorRegistry[0].ExitCount)
	}
}

func TestExitValidator_AlreadyExited(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
		},
	}
	if _, err := ExitValidator(state, 0); err == nil {
		t.Fatal("exitValidator should have failed with exiting again")
	}
}

func TestProcessPenaltiesExits_NothingHappened(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorBalances: []uint64{config.MaxDepositInGwei},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().FarFutureSlot},
		},
	}
	if ProcessPenaltiesAndExits(state).ValidatorBalances[0] !=
		config.MaxDepositInGwei {
		t.Errorf("wanted validator balance %d, got %d",
			config.MaxDepositInGwei,
			ProcessPenaltiesAndExits(state).ValidatorBalances[0])
	}
}

func TestProcessPenaltiesExits_ValidatorPenalized(t *testing.T) {

	latestPenalizedExits := make([]uint64, config.LatestPenalizedExitLength)
	for i := 0; i < len(latestPenalizedExits); i++ {
		latestPenalizedExits[i] = uint64(i) * config.MaxDepositInGwei
	}

	state := &pb.BeaconState{
		Slot:                        config.LatestPenalizedExitLength / 2 * config.EpochLength,
		LatestPenalizedExitBalances: latestPenalizedExits,
		ValidatorBalances:           []uint64{config.MaxDepositInGwei, config.MaxDepositInGwei},
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().FarFutureSlot, ExitCount: 1},
		},
	}

	penalty := EffectiveBalance(state, 0) *
		EffectiveBalance(state, 0) /
		config.MaxDepositInGwei

	newState := ProcessPenaltiesAndExits(state)
	if newState.ValidatorBalances[0] != config.MaxDepositInGwei-penalty {
		t.Errorf("wanted validator balance %d, got %d",
			config.MaxDepositInGwei-penalty,
			newState.ValidatorBalances[0])
	}
}

func TestEligibleToExit(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}

	state = &pb.BeaconState{
		Slot: config.MinValidatorWithdrawalTime,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().EntryExitDelay,
				PenalizedSlot: 1},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}
}

func TestUpdateRegistry_NoRotation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
			{ExitSlot: params.BeaconConfig().EntryExitDelay},
		},
		ValidatorBalances: []uint64{
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitSlot != config.EntryExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, config.EntryExitDelay, validator.ExitSlot)
		}
	}
	if newState.ValidatorRegistryLatestChangeSlot != state.Slot {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryLatestChangeSlot)
	}
}

func TestUpdateRegistry_Activate(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{ExitSlot: params.BeaconConfig().EntryExitDelay,
				ActivationSlot: 5 + config.EntryExitDelay + 1},
			{ExitSlot: params.BeaconConfig().EntryExitDelay,
				ActivationSlot: 5 + config.EntryExitDelay + 1},
		},
		ValidatorBalances: []uint64{
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
		},
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitSlot != config.EntryExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, config.EntryExitDelay, validator.ExitSlot)
		}
	}
	if newState.ValidatorRegistryLatestChangeSlot != state.Slot {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryLatestChangeSlot)
	}

	if bytes.Equal(newState.ValidatorRegistryDeltaChainTipHash32, []byte{'A'}) {
		t.Errorf("validator registry delta chain did not change")
	}
}

func TestUpdateRegistry_Exit(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5,
		ValidatorRegistry: []*pb.ValidatorRecord{
			{
				ExitSlot:    5 + config.EntryExitDelay + 1,
				StatusFlags: pb.ValidatorRecord_INITIATED_EXIT},
			{
				ExitSlot:    5 + config.EntryExitDelay + 1,
				StatusFlags: pb.ValidatorRecord_INITIATED_EXIT},
		},
		ValidatorBalances: []uint64{
			config.MaxDepositInGwei,
			config.MaxDepositInGwei,
		},
		ValidatorRegistryDeltaChainTipHash32: []byte{'A'},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitSlot != config.EntryExitDelay+5 {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i,
				config.EntryExitDelay+5,
				validator.ExitSlot)
		}
	}
	if newState.ValidatorRegistryLatestChangeSlot != state.Slot {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryLatestChangeSlot)
	}

	if bytes.Equal(newState.ValidatorRegistryDeltaChainTipHash32, []byte{'A'}) {
		t.Errorf("validator registry delta chain did not change")
	}
}

func TestMaxBalanceChurn(t *testing.T) {
	tests := []struct {
		totalBalance    uint64
		maxBalanceChurn uint64
	}{
		{totalBalance: 1e9, maxBalanceChurn: config.MaxDepositInGwei},
		{totalBalance: config.MaxDepositInGwei, maxBalanceChurn: 512 * 1e9},
		{totalBalance: config.MaxDepositInGwei * 10, maxBalanceChurn: 512 * 1e10},
		{totalBalance: config.MaxDepositInGwei * 1000, maxBalanceChurn: 512 * 1e12},
	}

	for _, tt := range tests {
		churn := maxBalanceChurn(tt.totalBalance)
		if tt.maxBalanceChurn != churn {
			t.Errorf("MaxBalanceChurn was not an expected value. Wanted: %d, got: %d",
				tt.maxBalanceChurn, churn)
		}
	}
}
