package balances

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBaseRewardQuotient(t *testing.T) {
	if params.BeaconConfig().BaseRewardQuotient != 1<<10 {
		t.Errorf("BaseRewardQuotient should be 1024 for these tests to pass")
	}
	if params.BeaconConfig().Gwei != 1e9 {
		t.Errorf("BaseRewardQuotient should be 1e9 for these tests to pass")
	}

	tests := []struct {
		a uint64
		b uint64
	}{
		{0, 0},
		{1e6 * params.BeaconConfig().Gwei, 1024000},  //1M ETH staked, 9.76% interest.
		{2e6 * params.BeaconConfig().Gwei, 1447936},  //2M ETH staked, 6.91% interest.
		{5e6 * params.BeaconConfig().Gwei, 2289664},  //5M ETH staked, 4.36% interest.
		{10e6 * params.BeaconConfig().Gwei, 3237888}, // 10M ETH staked, 3.08% interest.
		{20e6 * params.BeaconConfig().Gwei, 4579328}, // 20M ETH staked, 2.18% interest.
	}
	for _, tt := range tests {
		b := baseRewardQuotient(tt.a)
		if b != tt.b {
			t.Errorf("BaseRewardQuotient(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}

func TestBaseReward(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
	}{
		{0, 0},
		{params.BeaconConfig().MinDepositinGwei, 61},
		{30 * 1e9, 1853},
		{params.BeaconConfig().MaxDepositInGwei, 1976},
		{40 * 1e9, 1976},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{tt.a},
		}
		// Assume 10M Eth staked (base reward quotient: 3237888).
		b := baseReward(state, 0, 3237888)
		if b != tt.b {
			t.Errorf("BaseReward(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}

func TestInactivityPenalty(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
	}{
		{1, 2929},
		{2, 3883},
		{5, 6744},
		{10, 11512},
		{50, 49659},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositInGwei},
		}
		// Assume 10 ETH staked (base reward quotient: 3237888).
		b := inactivityPenalty(state, 0, 3237888, tt.a)
		if b != tt.b {
			t.Errorf("InactivityPenalty(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}

func TestFFGSrcRewardsPenalties(t *testing.T) {
	tests := []struct {
		voted                          []uint32
		balanceAfterSrcRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for FFG source,
		// balanceAfterSrcRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint32{}, []uint64{31999431819, 31999431819, 31999431819, 31999431819}},
		{[]uint32{0, 1}, []uint64{32000284090, 32000284090, 31999431819, 31999431819}},
		{[]uint32{0, 1, 2, 3}, []uint64{32000568181, 32000568181, 32000568181, 32000568181}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = FFGSrcRewardsPenalties(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDepositInGwei,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterSrcRewardPenalties) {
			t.Errorf("FFGSrcRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterSrcRewardPenalties)
		}
	}
}

func TestFFGTargetRewardsPenalties(t *testing.T) {
	tests := []struct {
		voted                          []uint32
		balanceAfterTgtRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for FFG target,
		// balanceAfterTgtRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint32{}, []uint64{31999431819, 31999431819, 31999431819, 31999431819}},
		{[]uint32{0, 1}, []uint64{32000284090, 32000284090, 31999431819, 31999431819}},
		{[]uint32{0, 1, 2, 3}, []uint64{32000568181, 32000568181, 32000568181, 32000568181}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = FFGTargetRewardsPenalties(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDepositInGwei,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterTgtRewardPenalties) {
			t.Errorf("FFGTargetRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterTgtRewardPenalties)
		}
	}
}

func TestChainHeadRewardsPenalties(t *testing.T) {
	tests := []struct {
		voted                           []uint32
		balanceAfterHeadRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for canonical chain,
		// balanceAfterHeadRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint32{}, []uint64{31999431819, 31999431819, 31999431819, 31999431819}},
		{[]uint32{0, 1}, []uint64{32000284090, 32000284090, 31999431819, 31999431819}},
		{[]uint32{0, 1, 2, 3}, []uint64{32000568181, 32000568181, 32000568181, 32000568181}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = ChainHeadRewardsPenalties(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDepositInGwei,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterHeadRewardPenalties) {
			t.Errorf("ChainHeadRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterHeadRewardPenalties)
		}
	}
}

func TestInclusionDistRewards_Ok(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          5},
	}

	tests := []struct {
		voted                        []uint32
		balanceAfterInclusionRewards []uint64
	}{
		// voted represents the validator indices that voted this epoch,
		// balanceAfterInclusionRewards represents their final balances after
		// applying rewards with inclusion.
		//
		// Validators shouldn't get penalized.
		{[]uint32{}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
		// Validators inclusion rewards are constant.
		{[]uint32{0, 1}, []uint64{32000454544, 32000454544, 32000000000, 32000000000}},
		{[]uint32{0, 1, 2, 3}, []uint64{32000454544, 32000454544, 32000454544, 32000454544}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
			LatestAttestations:        attestation,
		}
		state, err := InclusionDistRewards(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)
		if err != nil {
			t.Fatalf("could not execute InclusionDistRewards:%v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterInclusionRewards) {
			t.Errorf("InclusionDistRewards(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterInclusionRewards)
		}
	}
}

func TestInclusionDistRewards_NotOk(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff}},
	}

	tests := []struct {
		voted                        []uint32
		balanceAfterInclusionRewards []uint64
	}{
		{[]uint32{0, 1, 2, 3}, []uint64{}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			LatestAttestations:        attestation,
		}
		_, err := InclusionDistRewards(state, tt.voted, 0)
		if err == nil {
			t.Fatal("InclusionDistRewards should have failed")
		}
	}
}

func TestInactivityFFGSrcPenalty(t *testing.T) {
	tests := []struct {
		voted                     []uint32
		balanceAfterFFGSrcPenalty []uint64
		epochsSinceFinality       uint64
	}{
		// The higher the epochs since finality, the more penalties applied.
		{[]uint32{0, 1}, []uint64{32000000000, 32000000000, 31999427051, 31999427051}, 5},
		{[]uint32{}, []uint64{31999427051, 31999427051, 31999427051, 31999427051}, 5},
		{[]uint32{}, []uint64{31999422283, 31999422283, 31999422283, 31999422283}, 10},
		{[]uint32{}, []uint64{31999412746, 31999412746, 31999412746, 31999412746}, 20},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityFFGSrcPenalties(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei,
			tt.epochsSinceFinality)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterFFGSrcPenalty) {
			t.Errorf("InactivityFFGSrcPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterFFGSrcPenalty)
		}
	}
}

func TestInactivityFFGTargetPenalty(t *testing.T) {
	tests := []struct {
		voted                        []uint32
		balanceAfterFFGTargetPenalty []uint64
		epochsSinceFinality          uint64
	}{
		// The higher the epochs since finality, the more penalties applied.
		{[]uint32{0, 1}, []uint64{32000000000, 32000000000, 31999427051, 31999427051}, 5},
		{[]uint32{}, []uint64{31999427051, 31999427051, 31999427051, 31999427051}, 5},
		{[]uint32{}, []uint64{31999422283, 31999422283, 31999422283, 31999422283}, 10},
		{[]uint32{}, []uint64{31999412746, 31999412746, 31999412746, 31999412746}, 20},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityFFGTargetPenalties(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei,
			tt.epochsSinceFinality)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterFFGTargetPenalty) {
			t.Errorf("InactivityFFGTargetPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterFFGTargetPenalty)
		}
	}
}

func TestInactivityHeadPenalty(t *testing.T) {
	tests := []struct {
		voted                             []uint32
		balanceAfterInactivityHeadPenalty []uint64
	}{
		{[]uint32{}, []uint64{31999431819, 31999431819, 31999431819, 31999431819}},
		{[]uint32{0, 1}, []uint64{32000000000, 32000000000, 31999431819, 31999431819}},
		{[]uint32{0, 1, 2, 3}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityHeadPenalties(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterInactivityHeadPenalty) {
			t.Errorf("InactivityHeadPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterInactivityHeadPenalty)
		}
	}
}

func TestInactivityExitedPenality(t *testing.T) {
	tests := []struct {
		balanceAfterExitedPenalty []uint64
		epochsSinceFinality       uint64
	}{
		{[]uint64{31998285921, 31998285921, 31998285921, 31998285921}, 5},
		{[]uint64{31998276385, 31998276385, 31998276385, 31998276385}, 10},
		{[]uint64{31997341783, 31997341783, 31997341783, 31997341783}, 500},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.ValidatorRecord{
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot},
				{ExitSlot: params.BeaconConfig().FarFutureSlot}},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityExitedPenalties(
			state,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei,
			tt.epochsSinceFinality,
		)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterExitedPenalty) {
			t.Errorf("InactivityExitedPenalty(epochSinceFinality=%v) = %v, wanted: %v",
				tt.epochsSinceFinality, state.ValidatorBalances, tt.balanceAfterExitedPenalty)
		}
	}
}

func TestInactivityInclusionPenalty_Ok(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          5},
	}

	tests := []struct {
		voted                        []uint32
		balanceAfterInclusionPenalty []uint64
	}{
		{[]uint32{}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
		{[]uint32{0, 1}, []uint64{31999886363, 31999886363, 32000000000, 32000000000}},
		{[]uint32{0, 1, 2, 3}, []uint64{31999886363, 31999886363, 31999886363, 31999886363}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
			LatestAttestations:        attestation,
		}
		state, err := InactivityInclusionPenalties(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei)
		if err != nil {
			t.Fatalf("could not execute InactivityInclusionPenalty:%v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterInclusionPenalty) {
			t.Errorf("InactivityInclusionPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterInclusionPenalty)
		}
	}
}

func TestInactivityInclusionPenalty_NotOk(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff}},
	}

	tests := []struct {
		voted                        []uint32
		balanceAfterInclusionRewards []uint64
	}{
		{[]uint32{0, 1, 2, 3}, []uint64{}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			LatestAttestations:        attestation,
		}
		_, err := InactivityInclusionPenalties(state, tt.voted, 0)
		if err == nil {
			t.Fatal("InclusionDistRewards should have failed")
		}
	}
}

func TestAttestationInclusionRewards(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          0},
	}

	tests := []struct {
		voted                            []uint32
		balanceAfterAttestationInclusion []uint64
	}{
		{[]uint32{}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
		{[]uint32{0}, []uint64{32000071022, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
			LatestAttestations:        attestation,
		}
		state, err := AttestationInclusionRewards(
			state,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei,
			tt.voted)
		if err != nil {
			t.Fatalf("could not execute InactivityInclusionPenalty:%v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterAttestationInclusion) {
			t.Errorf("AttestationInclusionRewards(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterAttestationInclusion)
		}
	}
}

func TestAttestationInclusionRewards_NoInclusionSlot(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}

	tests := []struct {
		voted                            []uint32
		balanceAfterAttestationInclusion []uint64
	}{
		{[]uint32{0, 1, 2, 3}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
		}
		if _, err := AttestationInclusionRewards(state, 0, tt.voted); err == nil {
			t.Fatal("AttestationInclusionRewards should have failed with no inclusion slot")
		}
	}
}

func TestAttestationInclusionRewards_NoProposerIndex(t *testing.T) {
	shardAndCommittees := []*pb.ShardAndCommitteeArray{
		{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
			{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
		}}}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          0},
	}

	tests := []struct {
		voted                            []uint32
		balanceAfterAttestationInclusion []uint64
	}{
		{[]uint32{0}, []uint64{32000071022, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			Slot:                      1000,
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
			LatestAttestations:        attestation,
		}
		if _, err := AttestationInclusionRewards(state, 0, tt.voted); err == nil {
			t.Fatal("AttestationInclusionRewards should have failed with no proposer index")
		}
	}
}

func TestCrosslinksRewardsPenalties(t *testing.T) {
	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Shard: 1, Committee: []uint32{0, 1, 2, 3, 4, 5, 6, 7}},
			},
		})
	}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			ParticipationBitfield: []byte{0xff},
			SlotIncluded:          0},
	}

	tests := []struct {
		voted                        []uint32
		balanceAfterCrosslinkRewards []uint64
	}{
		{[]uint32{}, []uint64{
			32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9}},
		{[]uint32{0}, []uint64{
			32003124992, 31996875183, 31996875183, 31996875183, 31996875183, 31996875183, 31996875183, 31996875183}},
		{[]uint32{1, 3, 5, 7}, []uint64{
			31987502435, 32012499968, 31987502435, 32012499968, 31987502435, 32012499968, 31987502435, 32012499968}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 8)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDepositInGwei
		}
		state := &pb.BeaconState{
			ShardAndCommitteesAtSlots: shardAndCommittees,
			ValidatorBalances:         validatorBalances,
			LatestAttestations:        attestation,
		}
		state = CrosslinksRewardsPenalties(
			state,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDepositInGwei,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDepositInGwei,
			tt.voted)
		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterCrosslinkRewards) {
			t.Errorf("CrosslinksRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterCrosslinkRewards)
		}
	}
}
