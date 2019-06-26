package helpers

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEpochCommitteeCount_OK(t *testing.T) {
	// this defines the # of validators required to have 1 committee
	// per slot for epoch length.
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		committeeCount uint64
	}{
		{0, params.BeaconConfig().SlotsPerEpoch},
		{1000, params.BeaconConfig().SlotsPerEpoch},
		{2 * validatorsPerEpoch, 2 * params.BeaconConfig().SlotsPerEpoch},
		{5 * validatorsPerEpoch, 5 * params.BeaconConfig().SlotsPerEpoch},
		{16 * validatorsPerEpoch, 16 * params.BeaconConfig().SlotsPerEpoch},
		{32 * validatorsPerEpoch, 16 * params.BeaconConfig().SlotsPerEpoch},
	}
	for _, test := range tests {
		ClearAllCaches()
		vals := make([]*pb.Validator, test.validatorCount)
		for i := 0; i < len(vals); i++ {
			vals[i] = &pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}
		s := &pb.BeaconState{
			ValidatorRegistry: vals,
		}
		count, err := EpochCommitteeCount(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		if test.committeeCount != count {
			t.Errorf("wanted: %d, got: %d",
				test.committeeCount, count)
		}
	}
}

func TestEpochCommitteeCount_LessShardsThanEpoch(t *testing.T) {
	validatorCount := uint64(8)
	productionConfig := params.BeaconConfig()
	testConfig := &params.BeaconChainConfig{
		ShardCount:          1,
		SlotsPerEpoch:       4,
		TargetCommitteeSize: 2,
	}
	params.OverrideBeaconConfig(testConfig)
	vals := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(vals); i++ {
		vals[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	s := &pb.BeaconState{
		ValidatorRegistry: vals,
	}
	count, err := EpochCommitteeCount(s, 1)
	if err != nil {
		t.Fatal(err)
	}
	if count != validatorCount/testConfig.TargetCommitteeSize {
		t.Errorf("wanted: %d, got: %d",
			validatorCount/testConfig.TargetCommitteeSize, count)
	}
	params.OverrideBeaconConfig(productionConfig)
}

func TestShardDelta_OK(t *testing.T) {
	minShardDelta := params.BeaconConfig().ShardCount -
		params.BeaconConfig().ShardCount/params.BeaconConfig().SlotsPerEpoch
	tests := []struct {
		validatorCount uint64
		shardCount     uint64
	}{
		{0, params.BeaconConfig().SlotsPerEpoch},    // Empty minimum shards
		{1000, params.BeaconConfig().SlotsPerEpoch}, // 1000 Validators minimum shards,
		{100000, 768 /*len(active_validators) // TARGET_COMMITTEE_SIZE*/},
		{500000, minShardDelta}, // 5 Mil, above shard delta
	}
	for _, test := range tests {
		ClearAllCaches()
		vals := make([]*pb.Validator, test.validatorCount)
		for i := 0; i < len(vals); i++ {
			vals[i] = &pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}
		s := &pb.BeaconState{
			ValidatorRegistry: vals,
		}
		delta, err := ShardDelta(s, 1)
		if err != nil {
			t.Fatal(err)
		}
		if test.shardCount != delta {
			t.Errorf("wanted: %d, got: %d",
				test.shardCount, delta)
		}
	}
}

func TestComputeCommittee_WithoutCache(t *testing.T) {
	// Create 10 committees
	committeeCount := uint64(10)
	validatorCount := committeeCount * params.BeaconConfig().TargetCommitteeSize
	validators := make([]*pb.Validator, validatorCount)

	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   200,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		t.Fatal(err)
	}
	committees, err := ComputeCommittee(indices, seed, 0, 1 /* Total committee*/)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}

	// Test shuffled indices are correct for shard 5 committee
	shard := uint64(5)
	committee5, err := ComputeCommittee(indices, seed, shard, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}
	start := utils.SplitOffset(validatorCount, committeeCount, shard)
	end := utils.SplitOffset(validatorCount, committeeCount, shard+1)

	if !reflect.DeepEqual(committees[start:end], committee5) {
		t.Error("committee has different shuffled indices")
	}

	// Test shuffled indices are correct for shard 9 committee
	shard = uint64(9)
	committee9, err := ComputeCommittee(indices, seed, shard, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}
	start = utils.SplitOffset(validatorCount, committeeCount, shard)
	end = utils.SplitOffset(validatorCount, committeeCount, shard+1)

	if !reflect.DeepEqual(committees[start:end], committee9) {
		t.Error("committee has different shuffled indices")
	}
}

func TestComputeCommittee_WithCache(t *testing.T) {
	// Create 10 committees
	committeeCount := uint64(10)
	validatorCount := committeeCount * params.BeaconConfig().TargetCommitteeSize
	validators := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   200,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		t.Fatal(err)
	}

	// Test shuffled indices are correct for shard 3 committee
	shard := uint64(3)
	committee3, err := ComputeCommittee(indices, seed, shard, committeeCount)
	if err != nil {
		t.Errorf("could not compute committee: %v", err)
	}

	cachedIndices, err := shuffledIndicesCache.IndicesByIndexSeed(shard, seed[:])
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

	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	attestationData := &pb.AttestationData{}

	tests := []struct {
		attestationSlot uint64
		stateSlot       uint64
		bitfield        []byte
		wanted          []uint64
	}{
		{
			attestationSlot: 3,
			stateSlot:       5,
			bitfield:        []byte{0x03},
			wanted:          []uint64{71, 127},
		},
		{
			attestationSlot: 2,
			stateSlot:       10,
			bitfield:        []byte{0x01},
			wanted:          []uint64{85},
		},
		{
			attestationSlot: 11,
			stateSlot:       10,
			bitfield:        []byte{0x03},
			wanted:          []uint64{68, 102},
		},
	}

	for _, tt := range tests {
		ClearAllCaches()
		state.Slot = tt.stateSlot
		attestationData.Crosslink = &pb.Crosslink{
			Shard: tt.attestationSlot,
		}
		attestationData.TargetEpoch = 0

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

func TestAttestationParticipants_IncorrectBitfield(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	attestationData := &pb.AttestationData{Crosslink: &pb.Crosslink{}}

	if _, err := AttestingIndices(state, attestationData, []byte{}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}

func TestAttestationParticipants_EmptyBitfield(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}
	ClearAllCaches()

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	attestationData := &pb.AttestationData{Crosslink: &pb.Crosslink{}}

	var zeroByte [16]byte

	indices, err := AttestingIndices(state, attestationData, zeroByte[:])
	if err != nil {
		t.Fatalf("attesting indices failed: %v", err)
	}

	if len(indices) != 0 {
		t.Errorf("Attesting indices are non-zero despite an empty bitfield being provided; Size %d", len(indices))
	}
}

func TestVerifyBitfield_OK(t *testing.T) {
	bitfield := []byte{0xFF}
	committeeSize := 8

	isValidated, err := VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if !isValidated {
		t.Error("bitfield is not validated when it was supposed to be")
	}

	bitfield = []byte{0xff, 0x80}
	committeeSize = 9

	isValidated, err = VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if isValidated {
		t.Error("bitfield is validated when it was supposed to be")
	}

	bitfield = []byte{0xff, 0x03}
	committeeSize = 10
	isValidated, err = VerifyBitfield(bitfield, committeeSize)
	if err != nil {
		t.Fatal(err)
	}

	if !isValidated {
		t.Error("bitfield is not validated when it was supposed to be")
	}
}

func TestCommitteeAssignment_CanRetrieve(t *testing.T) {
	// Initialize test with 128 validators, each slot and each shard gets 2 validators.
	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		Slot:                   params.BeaconConfig().SlotsPerEpoch,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	tests := []struct {
		index      uint64
		slot       uint64
		committee  []uint64
		shard      uint64
		isProposer bool
	}{
		{
			index:      0,
			slot:       146,
			committee:  []uint64{0, 3},
			shard:      82,
			isProposer: false,
		},
		{
			index:      105,
			slot:       160,
			committee:  []uint64{105, 20},
			shard:      96,
			isProposer: false,
		},
		{
			index:      64,
			slot:       183,
			committee:  []uint64{64, 33},
			shard:      119,
			isProposer: true,
		},
		{
			index:      11,
			slot:       135,
			committee:  []uint64{119, 11},
			shard:      71,
			isProposer: false,
		},
	}

	for _, tt := range tests {
		ClearAllCaches()
		committee, shard, slot, isProposer, err := CommitteeAssignment(state, tt.slot/params.BeaconConfig().SlotsPerEpoch, tt.index)
		if err != nil {
			t.Fatalf("failed to execute NextEpochCommitteeAssignment: %v", err)
		}
		if shard != tt.shard {
			t.Errorf("wanted shard %d, got shard %d for validator index %d",
				tt.shard, shard, tt.index)
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
	}
}

func TestCommitteeAssignment_CantFindValidator(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
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

func TestShardDelta_Ok(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	min := params.BeaconConfig().ShardCount - params.BeaconConfig().ShardCount/params.BeaconConfig().SlotsPerEpoch
	tests := []struct {
		validatorCount uint64
		shardDelta     uint64
	}{
		{0, params.BeaconConfig().SlotsPerEpoch},
		{1000, params.BeaconConfig().SlotsPerEpoch},
		{2 * validatorsPerEpoch, 2 * params.BeaconConfig().SlotsPerEpoch},
		{5 * validatorsPerEpoch, 5 * params.BeaconConfig().SlotsPerEpoch},
		{16 * validatorsPerEpoch, min},
		{32 * validatorsPerEpoch, min},
	}
	for _, test := range tests {
		ClearAllCaches()
		validators := make([]*pb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}
		state := &pb.BeaconState{ValidatorRegistry: validators}
		delta, err := ShardDelta(state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if test.shardDelta != delta {
			t.Errorf("wanted: %d, got: %d",
				test.shardDelta, delta)
		}
	}
}

func TestEpochStartShard_EpochOutOfBound(t *testing.T) {
	_, err := EpochStartShard(&pb.BeaconState{}, 2)
	want := "epoch 2 can't be greater than 1"
	if err.Error() != want {
		t.Fatalf("Did not generate correct error. Want: %s, got: %s",
			err.Error(), want)
	}
}

func TestEpochStartShard_AccurateShard(t *testing.T) {
	validatorsPerEpoch := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		startShard     uint64
	}{
		{0, 676},
		{1000, 676},
		{2 * validatorsPerEpoch, 228},
		{5 * validatorsPerEpoch, 932},
		{16 * validatorsPerEpoch, 212},
		{32 * validatorsPerEpoch, 212},
	}
	for _, test := range tests {
		ClearAllCaches()
		validators := make([]*pb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}
		state := &pb.BeaconState{ValidatorRegistry: validators, LatestStartShard: 100, Slot: 500}
		startShard, err := EpochStartShard(state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if test.startShard != startShard {
			t.Errorf("wanted: %d, got: %d", test.startShard, startShard)
		}
	}
}

func TestVerifyAttestationBitfield_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	activeRoots := make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		activeRoots[i] = []byte{'A'}
	}

	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestActiveIndexRoots: activeRoots,
		LatestRandaoMixes:      activeRoots,
	}

	tests := []struct {
		attestation         *pb.Attestation
		stateSlot           uint64
		errorExists         bool
		verificationFailure bool
	}{
		{
			attestation: &pb.Attestation{
				AggregationBitfield: []byte{0x01},
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 5,
					},
				},
			},
			stateSlot: 5,
		},
		{

			attestation: &pb.Attestation{
				AggregationBitfield: []byte{0x02},
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 10,
					},
				},
			},
			stateSlot: 10,
		},
		{
			attestation: &pb.Attestation{
				AggregationBitfield: []byte{0x02},
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 20,
					},
				},
			},
			stateSlot: 20,
		},
		{
			attestation: &pb.Attestation{
				AggregationBitfield: []byte{0xFF, 0xC0},
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 5,
					},
				},
			},
			stateSlot:   5,
			errorExists: true,
		},
		{
			attestation: &pb.Attestation{
				AggregationBitfield: []byte{0xFF},
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 20,
					},
				},
			},
			stateSlot:           20,
			verificationFailure: true,
		},
	}

	for _, tt := range tests {
		ClearAllCaches()
		state.Slot = tt.stateSlot
		verified, err := VerifyAttestationBitfield(state, tt.attestation)
		if tt.errorExists {
			if err == nil {
				t.Error("error is nil, when verification is supposed to fail")
			}
			continue
		}
		if tt.verificationFailure {
			if verified {
				t.Error("verification succeeded when it was supposed to fail")
			}
			continue
		}
		if err != nil {
			t.Errorf("Failed to verify bitfield: %v", err)
			continue
		}
		if !verified {
			t.Errorf("Bitfield isnt verified: %08b", tt.attestation.AggregationBitfield)
		}
	}
}

func BenchmarkComputeCommittee300000_WithPreCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 300000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		b.Fatal(err)
	}

	shard := uint64(3)
	_, err = ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkComputeCommittee3000000_WithPreCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 3000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		b.Fatal(err)
	}

	shard := uint64(3)
	_, err = ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkComputeCommittee128000_WithOutPreCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 128000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	shard := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			shard = (shard + 1) % params.BeaconConfig().ShardCount
			i = 0
		}
	}
}

func BenchmarkComputeCommittee1000000_WithOutCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 1000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	shard := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			shard = (shard + 1) % params.BeaconConfig().ShardCount
			i = 0
		}
	}
}

func BenchmarkComputeCommittee4000000_WithOutCache(b *testing.B) {
	ClearShuffledValidatorCache()
	validators := make([]*pb.Validator, 4000000)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	epoch := CurrentEpoch(state)
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		b.Fatal(err)
	}
	seed, err := GenerateSeed(state, epoch)
	if err != nil {
		b.Fatal(err)
	}

	i := uint64(0)
	shard := uint64(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		i++
		_, err := ComputeCommittee(indices, seed, shard, params.BeaconConfig().ShardCount)
		if err != nil {
			panic(err)
		}
		if i < params.BeaconConfig().TargetCommitteeSize {
			shard = (shard + 1) % params.BeaconConfig().ShardCount
			i = 0
		}
	}
}
