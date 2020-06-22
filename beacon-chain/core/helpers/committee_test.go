package helpers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

func TestComputeCommittee_WithoutCache(t *testing.T) {
	// Create 10 committees
	committeeCount := uint64(10)
	validatorCount := committeeCount * params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)

	for i := 0; i < len(validators); i++ {
		k := make([]byte, 48)
		copy(k, strconv.Itoa(i))
		validators[i] = &ethpb.Validator{
			PublicKey:             k,
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		Slot:        200,
		BlockRoots:  make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		StateRoots:  make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	committees, err := ComputeCommittee(indices, seed, 0, 1 /* Total committee*/)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}

	// Test shuffled indices are correct for index 5 committee
	index := uint64(5)
	committee5, err := ComputeCommittee(indices, seed, index, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}
	start := sliceutil.SplitOffset(validatorCount, committeeCount, index)
	end := sliceutil.SplitOffset(validatorCount, committeeCount, index+1)

	if !reflect.DeepEqual(committees[start:end], committee5) {
		t.Error("committee has different shuffled indices")
	}

	// Test shuffled indices are correct for index 9 committee
	index = uint64(9)
	committee9, err := ComputeCommittee(indices, seed, index, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}
	start = sliceutil.SplitOffset(validatorCount, committeeCount, index)
	end = sliceutil.SplitOffset(validatorCount, committeeCount, index+1)

	if !reflect.DeepEqual(committees[start:end], committee9) {
		t.Error("committee has different shuffled indices")
	}
}

func TestAttestationParticipants_NoCommitteeCache(t *testing.T) {
	committeeSize := uint64(16)
	validators := make([]*ethpb.Validator, committeeSize*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Slot:        params.BeaconConfig().SlotsPerEpoch,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	attestationData := &ethpb.AttestationData{}

	tests := []struct {
		attestationSlot uint64
		bitfield        bitfield.Bitlist
		wanted          []uint64
	}{
		{
			attestationSlot: 3,
			bitfield:        bitfield.Bitlist{0x07},
			wanted:          []uint64{344, 221},
		},
		{
			attestationSlot: 2,
			bitfield:        bitfield.Bitlist{0x05},
			wanted:          []uint64{207},
		},
		{
			attestationSlot: 11,
			bitfield:        bitfield.Bitlist{0x07},
			wanted:          []uint64{409, 213},
		},
	}

	for _, tt := range tests {
		attestationData.Target = &ethpb.Checkpoint{Epoch: 0}
		attestationData.Slot = tt.attestationSlot

		committee, err := BeaconCommitteeFromState(state, tt.attestationSlot, 0 /* committee index */)
		if err != nil {
			t.Error(err)
		}
		result, err := AttestingIndices(tt.bitfield, committee)
		if err != nil {
			t.Errorf("Failed to get attestation participants: %v", err)
		}

		if !reflect.DeepEqual(tt.wanted, result) {
			t.Errorf(
				"Result indices was an unexpected value. Wanted %d, got %d",
				tt.wanted,
				result,
			)
		}
	}
}

func TestAttestationParticipants_EmptyBitfield(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	attestationData := &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}

	committee, err := BeaconCommitteeFromState(state, attestationData.Slot, attestationData.CommitteeIndex)
	if err != nil {
		t.Fatal(err)
	}
	indices, err := AttestingIndices(bitfield.NewBitlist(128), committee)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 0 {
		t.Errorf("Attesting indices are non-zero despite an empty bitfield being provided; Size %d", len(indices))
	}
}

func TestVerifyBitfieldLength_OK(t *testing.T) {
	bf := bitfield.Bitlist{0xFF, 0x01}
	committeeSize := uint64(8)
	if err := VerifyBitfieldLength(bf, committeeSize); err != nil {
		t.Errorf("bitfield is not validated when it was supposed to be: %v", err)
	}

	bf = bitfield.Bitlist{0xFF, 0x07}
	committeeSize = 10
	if err := VerifyBitfieldLength(bf, committeeSize); err != nil {
		t.Errorf("bitfield is not validated when it was supposed to be: %v", err)
	}
}

func TestCommitteeAssignments_CannotRetrieveFutureEpoch(t *testing.T) {
	ClearCache()
	epoch := uint64(1)
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Slot: 0, // Epoch 0.
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = CommitteeAssignments(state, epoch+1)
	if err == nil {
		t.Fatal("Expected error, received nil")
	}
	if !strings.Contains(err.Error(), "can't be greater than next epoch") {
		t.Errorf("Expected unable to get greater than next epoch, received %v", err)
	}
}

func TestCommitteeAssignments_NoProposerForSlot0(t *testing.T) {
	validators := make([]*ethpb.Validator, 4*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		var activationEpoch uint64
		if i >= len(validators)/2 {
			activationEpoch = 3
		}
		validators[i] = &ethpb.Validator{
			ActivationEpoch: activationEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		Slot:        2 * params.BeaconConfig().SlotsPerEpoch, // epoch 2
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	ClearCache()
	_, proposerIndexToSlots, err := CommitteeAssignments(state, 0)
	if err != nil {
		t.Fatalf("failed to determine CommitteeAssignments: %v", err)
	}
	for _, slots := range proposerIndexToSlots {
		for _, s := range slots {
			if s == 0 {
				t.Error("No proposer should be assigned to slot 0")
			}
		}
	}
}

func TestCommitteeAssignments_CanRetrieve(t *testing.T) {
	// Initialize test with 256 validators, each slot and each index gets 4 validators.
	validators := make([]*ethpb.Validator, 4*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		// First 2 epochs only half validators are activated.
		var activationEpoch uint64
		if i >= len(validators)/2 {
			activationEpoch = 3
		}
		validators[i] = &ethpb.Validator{
			ActivationEpoch: activationEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		Slot:        2 * params.BeaconConfig().SlotsPerEpoch, // epoch 2
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		index          uint64
		slot           uint64
		committee      []uint64
		committeeIndex uint64
		isProposer     bool
		proposerSlot   uint64
	}{
		{
			index:          0,
			slot:           78,
			committee:      []uint64{0, 38},
			committeeIndex: 0,
			isProposer:     false,
		},
		{
			index:          1,
			slot:           71,
			committee:      []uint64{1, 4},
			committeeIndex: 0,
			isProposer:     true,
			proposerSlot:   79,
		},
		{
			index:          11,
			slot:           90,
			committee:      []uint64{31, 11},
			committeeIndex: 0,
			isProposer:     false,
		}, {
			index:          2,
			slot:           127, // 3rd epoch has more active validators
			committee:      []uint64{89, 2, 81, 5},
			committeeIndex: 0,
			isProposer:     false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ClearCache()
			validatorIndexToCommittee, proposerIndexToSlots, err := CommitteeAssignments(state, SlotToEpoch(tt.slot))
			if err != nil {
				t.Fatalf("failed to determine CommitteeAssignments: %v", err)
			}
			cac := validatorIndexToCommittee[tt.index]
			if cac.CommitteeIndex != tt.committeeIndex {
				t.Errorf("wanted committeeIndex %d, got committeeIndex %d for validator index %d",
					tt.committeeIndex, cac.CommitteeIndex, tt.index)
			}
			if cac.AttesterSlot != tt.slot {
				t.Errorf("wanted slot %d, got slot %d for validator index %d",
					tt.slot, cac.AttesterSlot, tt.index)
			}
			if len(proposerIndexToSlots[tt.index]) > 0 && proposerIndexToSlots[tt.index][0] != tt.proposerSlot {
				t.Errorf("wanted proposer slot %d, got proposer slot %d for validator index %d",
					tt.proposerSlot, proposerIndexToSlots[tt.index][0], tt.index)
			}
			if !reflect.DeepEqual(cac.Committee, tt.committee) {
				t.Errorf("wanted committee %v, got committee %v for validator index %d",
					tt.committee, cac.Committee, tt.index)
			}
		})
	}
}

func TestCommitteeAssignments_EverySlotHasMin1Proposer(t *testing.T) {
	// Initialize test with 256 validators, each slot and each index gets 4 validators.
	validators := make([]*ethpb.Validator, 4*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch: 0,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		Slot:        2 * params.BeaconConfig().SlotsPerEpoch, // epoch 2
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	ClearCache()
	epoch := uint64(1)
	_, proposerIndexToSlots, err := CommitteeAssignments(state, epoch)
	if err != nil {
		t.Fatalf("failed to determine CommitteeAssignments: %v", err)
	}

	slotsWithProposers := make(map[uint64]bool)
	for _, proposerSlots := range proposerIndexToSlots {
		for _, slot := range proposerSlots {
			slotsWithProposers[slot] = true
		}
	}
	if uint64(len(slotsWithProposers)) != params.BeaconConfig().SlotsPerEpoch {
		t.Errorf(
			"Expected %d slots with proposers, received %d",
			params.BeaconConfig().SlotsPerEpoch,
			len(slotsWithProposers),
		)
	}
	startSlot := StartSlot(epoch)
	endSlot := StartSlot(epoch + 1)
	for i := startSlot; i < endSlot; i++ {
		hasProposer := slotsWithProposers[i]
		if !hasProposer {
			t.Errorf("Expected every slot in epoch 1 to have a proposer, slot %d did not", i)
		}
	}
}

func TestVerifyAttestationBitfieldLengths_OK(t *testing.T) {
	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	activeRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: activeRoots,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		attestation         *ethpb.Attestation
		stateSlot           uint64
		verificationFailure bool
	}{
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x05},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 5,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot: 5,
		},
		{

			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 10,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot: 10,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 20,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot: 20,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 20,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot: 20,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0xFF, 0xC0, 0x01},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 5,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot:           5,
			verificationFailure: true,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0xFF, 0x01},
				Data: &ethpb.AttestationData{
					CommitteeIndex: 20,
					Target:         &ethpb.Checkpoint{},
				},
			},
			stateSlot:           20,
			verificationFailure: true,
		},
	}

	for i, tt := range tests {
		ClearCache()
		if err := state.SetSlot(tt.stateSlot); err != nil {
			t.Fatal(err)
		}
		err := VerifyAttestationBitfieldLengths(state, tt.attestation)
		if tt.verificationFailure {
			if err == nil {
				t.Error("verification succeeded when it was supposed to fail")
			}
			continue
		}
		if err != nil {
			t.Errorf("%d Failed to verify bitfield: %v", i, err)
			continue
		}
	}
}

func TestShuffledIndices_ShuffleRightLength(t *testing.T) {
	valiatorCount := 1000
	validators := make([]*ethpb.Validator, valiatorCount)
	indices := make([]uint64, valiatorCount)
	for i := 0; i < valiatorCount; i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		indices[i] = uint64(i)
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Test for current epoch
	shuffledIndices, err := ShuffledIndices(state, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(shuffledIndices) != valiatorCount {
		t.Errorf("Incorrect shuffled indices count, wanted: %d, got: %d",
			valiatorCount, len(shuffledIndices))
	}
	if reflect.DeepEqual(indices, shuffledIndices) {
		t.Error("Shuffling did not happen")
	}

	// Test for next epoch
	shuffledIndices, err = ShuffledIndices(state, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(shuffledIndices) != valiatorCount {
		t.Errorf("Incorrect shuffled indices count, wanted: %d, got: %d",
			valiatorCount, len(shuffledIndices))
	}
	if reflect.DeepEqual(indices, shuffledIndices) {
		t.Error("Shuffling did not happen")
	}
}

func TestUpdateCommitteeCache_CanUpdate(t *testing.T) {
	ClearCache()
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount
	validators := make([]*ethpb.Validator, validatorCount)
	indices := make([]uint64, validatorCount)
	for i := uint64(0); i < validatorCount; i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		indices[i] = i
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdateCommitteeCache(state, CurrentEpoch(state)); err != nil {
		t.Fatal(err)
	}

	epoch := uint64(1)
	idx := uint64(1)
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}

	indices, err = committeeCache.Committee(StartSlot(epoch), seed, idx)
	if err != nil {
		t.Fatal(err)
	}
	if uint64(len(indices)) != params.BeaconConfig().TargetCommitteeSize {
		t.Errorf("Did not save correct indices lengths, got %d wanted %d", len(indices), params.BeaconConfig().TargetCommitteeSize)
	}
}

func BenchmarkComputeCommittee300000_WithPreCache(b *testing.B) {
	validators := make([]*ethpb.Validator, 300000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		b.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		b.Fatal(err)
	}

	index := uint64(3)
	_, err = ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkComputeCommittee3000000_WithPreCache(b *testing.B) {
	validators := make([]*ethpb.Validator, 3000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		b.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		b.Fatal(err)
	}

	index := uint64(3)
	_, err = ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkComputeCommittee128000_WithOutPreCache(b *testing.B) {
	validators := make([]*ethpb.Validator, 128000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		b.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	index := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			index = (index + 1) % params.BeaconConfig().MaxCommitteesPerSlot
			i = 0
		}
	}
}

func BenchmarkComputeCommittee1000000_WithOutCache(b *testing.B) {
	validators := make([]*ethpb.Validator, 1000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		b.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	index := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			index = (index + 1) % params.BeaconConfig().MaxCommitteesPerSlot
			i = 0
		}
	}
}

func BenchmarkComputeCommittee4000000_WithOutCache(b *testing.B) {
	validators := make([]*ethpb.Validator, 4000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		b.Fatal(err)
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	index := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, index, params.BeaconConfig().MaxCommitteesPerSlot)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			index = (index + 1) % params.BeaconConfig().MaxCommitteesPerSlot
			i = 0
		}
	}
}

func TestBeaconCommitteeFromState_UpdateCacheForPreviousEpoch(t *testing.T) {
	committeeSize := uint64(16)
	validators := make([]*ethpb.Validator, committeeSize*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Slot:        params.BeaconConfig().SlotsPerEpoch,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := BeaconCommitteeFromState(state, 1 /* previous epoch */, 0); err != nil {
		t.Fatal(err)
	}

	// Verify previous epoch is cached
	seed, err := Seed(state, 0, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	activeIndices, err := committeeCache.ActiveIndices(seed)
	if err != nil {
		t.Fatal(err)
	}
	if activeIndices == nil {
		t.Error("did not cache active indices")
	}
}

func TestPrecomputeProposerIndices_Ok(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	indices, err := ActiveValidatorIndices(state, 0)
	if err != nil {
		t.Fatal(err)
	}

	proposerIndices, err := precomputeProposerIndices(state, indices)
	if err != nil {
		t.Fatal(err)
	}

	var wantedProposerIndices []uint64
	seed, err := Seed(state, 0, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(i)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		index, err := ComputeProposerIndex(state, indices, seedWithSlotHash)
		if err != nil {
			t.Fatal(err)
		}
		wantedProposerIndices = append(wantedProposerIndices, index)
	}

	if !reflect.DeepEqual(wantedProposerIndices, proposerIndices) {
		t.Error("Did not precompute proposer indices correctly")
	}
}
