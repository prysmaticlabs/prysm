package helpers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestComputeCommittee_WithoutCache(t *testing.T) {
	// Create 10 committees
	committeeCount := uint64(10)
	validatorCount := committeeCount * params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)

	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             200,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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

func TestComputeCommittee_WithCache(t *testing.T) {
	// Create 10 committees
	committeeCount := uint64(10)
	validatorCount := committeeCount * params.BeaconConfig().TargetCommitteeSize
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             200,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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

	// Test shuffled indices are correct for index 3 committee
	index := uint64(3)
	committee3, err := ComputeCommittee(indices, seed, index, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}

	cachedIndices, err := shuffledIndicesCache.IndicesByIndexSeed(index, seed[:])
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(cachedIndices, committee3) {
		t.Error("committee has different shuffled indices")
	}
}

func TestAttestationParticipants_NoCommitteeCache(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	committeeSize := uint64(16)
	validators := make([]*ethpb.Validator, committeeSize*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch,
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
			wanted:          []uint64{290, 402},
		},
		{
			attestationSlot: 2,
			bitfield:        bitfield.Bitlist{0x05},
			wanted:          []uint64{853},
		},
		{
			attestationSlot: 11,
			bitfield:        bitfield.Bitlist{0x07},
			wanted:          []uint64{872, 103},
		},
	}

	for _, tt := range tests {
		ClearAllCaches()
		attestationData.Target = &ethpb.Checkpoint{Epoch: 0}
		attestationData.Slot = tt.attestationSlot

		result, err := AttestingIndices(state, attestationData, tt.bitfield)
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
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}
	ClearAllCaches()

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	attestationData := &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}

	indices, err := AttestingIndices(state, attestationData, bitfield.NewBitlist(128))
	if err != nil {
		t.Fatalf("attesting indices failed: %v", err)
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

func TestCommitteeAssignment_CanRetrieve(t *testing.T) {
	// Initialize test with 128 validators, each slot and each index gets 2 validators.
	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             params.BeaconConfig().SlotsPerEpoch,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	tests := []struct {
		index      uint64
		slot       uint64
		committee  []uint64
		committeeIndex      uint64
		isProposer bool
	}{
		{
			index:      0,
			slot:       146,
			committee:  []uint64{0, 3},
			committeeIndex:      82,
			isProposer: true,
		},
		{
			index:      105,
			slot:       160,
			committee:  []uint64{105, 20},
			committeeIndex:      32,
			isProposer: true,
		},
		{
			index:      0,
			slot:       146,
			committee:  []uint64{0, 3},
			committeeIndex:      18,
			isProposer: true,
		},
		{
			index:      11,
			slot:       135,
			committee:  []uint64{119, 11},
			committeeIndex:      7,
			isProposer: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ClearAllCaches()
			committee, committeeIndex, slot, isProposer, err := CommitteeAssignment(state, tt.slot/params.BeaconConfig().SlotsPerEpoch, tt.index)
			if err != nil {
				t.Fatalf("failed to execute NextEpochCommitteeAssignment: %v", err)
			}
			if committeeIndex != tt.committeeIndex {
				t.Errorf("wanted committeeIndex %d, got committeeIndex %d for validator index %d",
					tt.committeeIndex, committeeIndex, tt.index)
			}
			if slot != tt.slot {
				t.Errorf("wanted slot %d, got slot %d for validator index %d",
					tt.slot, slot, tt.index)
			}
			if isProposer != tt.isProposer {
				t.Errorf("wanted isProposer %v, got isProposer %v for validator index %d",
					tt.isProposer, isProposer, tt.index)
			}
			if !reflect.DeepEqual(committee, tt.committee) {
				t.Errorf("wanted committee %v, got committee %v for validator index %d",
					tt.committee, committee, tt.index)
			}
		})
	}
}

func TestCommitteeAssignment_EveryValidatorShouldPropose(t *testing.T) {
	// Initialize 64 validators with 64 slots per epoch. Every validator
	// in the epoch should be a proposer.
	validators := make([]*ethpb.Validator, params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             params.BeaconConfig().SlotsPerEpoch,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	ClearAllCaches()
	for i := 0; i < len(validators); i++ {
		_, _, _, isProposer, err := CommitteeAssignment(state, state.Slot/params.BeaconConfig().SlotsPerEpoch, uint64(i))
		if err != nil {
			t.Fatal(err)
		}
		if !isProposer {
			t.Errorf("validator %d should be a proposer", i)
		}
	}
}

func TestCommitteeAssignment_CantFindValidator(t *testing.T) {
	state := &pb.BeaconState{
		Slot:             params.BeaconConfig().SlotsPerEpoch,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	index := uint64(10000)
	_, _, _, _, err := CommitteeAssignment(state, 1, index)
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatal(err)
	}
	if statusErr.Code() != codes.NotFound {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestVerifyAttestationBitfieldLengths_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	activeRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		activeRoots[i] = []byte{'A'}
	}

	state := &pb.BeaconState{
		Validators:       validators,
		ActiveIndexRoots: activeRoots,
		RandaoMixes:      activeRoots,
	}

	tests := []struct {
		attestation         *ethpb.Attestation
		stateSlot           uint64
		invalidCustodyBits  bool
		verificationFailure bool
	}{
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x05},
				CustodyBits:     bitfield.Bitlist{0x05},
				Data: &ethpb.AttestationData{
					Index: 5,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot: 5,
		},
		{

			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				CustodyBits:     bitfield.Bitlist{0x06},
				Data: &ethpb.AttestationData{
					Index: 10,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot: 10,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				CustodyBits:     bitfield.Bitlist{0x06},
				Data: &ethpb.AttestationData{
					Index: 20,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot: 20,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x06},
				CustodyBits:     bitfield.Bitlist{0x10},
				Data: &ethpb.AttestationData{
					Index: 20,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot:           20,
			verificationFailure: true,
			invalidCustodyBits:  true,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0xFF, 0xC0, 0x01},
				CustodyBits:     bitfield.Bitlist{0xFF, 0xC0, 0x01},
				Data: &ethpb.AttestationData{
					Index: 5,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot:           5,
			verificationFailure: true,
		},
		{
			attestation: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0xFF, 0x01},
				CustodyBits:     bitfield.Bitlist{0xFF, 0x01},
				Data: &ethpb.AttestationData{
					Index: 20,
					Target: &ethpb.Checkpoint{},
				},
			},
			stateSlot:           20,
			verificationFailure: true,
		},
	}

	for i, tt := range tests {
		ClearAllCaches()
		state.Slot = tt.stateSlot
		err := VerifyAttestationBitfieldLengths(state, tt.attestation)
		if tt.verificationFailure {
			if tt.invalidCustodyBits {
				if !strings.Contains(err.Error(), "custody bitfield") {
					t.Errorf("%d expected custody bits to fail: %v", i, err)
				}
			}
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
	ClearAllCaches()

	valiatorCount := 1000
	validators := make([]*ethpb.Validator, valiatorCount)
	indices := make([]uint64, valiatorCount)
	for i := 0; i < valiatorCount; i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		indices[i] = uint64(i)
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
	ClearAllCaches()

	validatorCount := int(params.BeaconConfig().MinGenesisActiveValidatorCount)
	validators := make([]*ethpb.Validator, validatorCount)
	indices := make([]uint64, validatorCount)
	for i := 0; i < validatorCount; i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		indices[i] = uint64(i)
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	if err := UpdateCommitteeCache(state); err != nil {
		t.Fatal(err)
	}
	savedEpochs, err := committeeCache.Epochs()
	if err != nil {
		t.Fatal(err)
	}
	if len(savedEpochs) != 2 {
		t.Error("Did not save correct epoch lengths")
	}
	epoch := uint64(1)
	idx := uint64(512)
	indices, err = committeeCache.ShuffledIndices(epoch, idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != int(params.BeaconConfig().TargetCommitteeSize) {
		t.Error("Did not save correct indices lengths")
	}
}

func TestCompressValidator(t *testing.T) {
	tests := []struct {
		validator *ethpb.Validator
		idx       uint64
		want      uint64
	}{
		{
			validator: &ethpb.Validator{
				EffectiveBalance: 32e9,
				Slashed:          true,
			},
			idx:  128,
			want: 8421408, // (128 << 16) + (1 << 15) + (32e9 / (2**0 * 10**9))
		},
		{
			validator: &ethpb.Validator{
				EffectiveBalance: 32e9,
				Slashed:          false,
			},
			idx:  128,
			want: 8388640, // (128 << 16) + (0 << 15) + (32e9 / (2**0 * 10**9))
		},
		{
			validator: &ethpb.Validator{
				EffectiveBalance: 33e9,
				Slashed:          false,
			},
			idx:  128,
			want: 8388641, // (128 << 16) + (0 << 15) + (33e9 / (2**0 * 10**9))
		},
		{
			validator: &ethpb.Validator{
				EffectiveBalance: 33e9,
				Slashed:          false,
			},
			idx:  129,
			want: 8454177, // (129 << 16) + (0 << 15) + (33e9 / (2**0 * 10**9))
		},
	}

	for _, tt := range tests {
		got := compressValidator(tt.validator, tt.idx)
		if got != tt.want {
			t.Errorf(
				"compressValidator({%v}, %d) = %d, wanted %d",
				tt.validator,
				tt.idx,
				got,
				tt.want,
			)
		}
	}
}

func BenchmarkComputeCommittee300000_WithPreCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*ethpb.Validator, 300000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
	ClearShuffledValidatorCache()
	validators := make([]*ethpb.Validator, 3000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
	ClearShuffledValidatorCache()
	validators := make([]*ethpb.Validator, 128000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
	ClearShuffledValidatorCache()
	validators := make([]*ethpb.Validator, 1000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
	ClearShuffledValidatorCache()
	validators := make([]*ethpb.Validator, 4000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
