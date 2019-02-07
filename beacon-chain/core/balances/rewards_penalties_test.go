package balances

import (
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBaseRewardQuotient(t *testing.T) {
	if params.BeaconConfig().BaseRewardQuotient != 1<<5 {
		t.Errorf("BaseRewardQuotient should be 32 for these tests to pass")
	}

	tests := []struct {
		a uint64
		b uint64
	}{
		{0, 0},
		{1e6 * 1e9, 988211},   //1M ETH staked, 9.76% interest.
		{2e6 * 1e9, 1397542},  //2M ETH staked, 6.91% interest.
		{5e6 * 1e9, 2209708},  //5M ETH staked, 4.36% interest.
		{10e6 * 1e9, 3125000}, // 10M ETH staked, 3.08% interest.
		{20e6 * 1e9, 4419417}, // 20M ETH staked, 2.18% interest.
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
		{params.BeaconConfig().MinDeposit, 61},
		{30 * 1e9, 1853},
		{params.BeaconConfig().MaxDeposit, 1976},
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
			ValidatorBalances: []uint64{params.BeaconConfig().MaxDeposit},
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
		voted                          []uint64
		balanceAfterSrcRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for FFG source,
		// balanceAfterSrcRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint64{}, []uint64{31999427550, 31999427550, 31999427550, 31999427550}},
		{[]uint64{0, 1}, []uint64{32000286225, 32000286225, 31999427550, 31999427550}},
		{[]uint64{0, 1, 2, 3}, []uint64{32000572450, 32000572450, 32000572450, 32000572450}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = ExpectedFFGSource(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDeposit,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterSrcRewardPenalties) {
			t.Errorf("FFGSrcRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterSrcRewardPenalties)
		}
	}
}

func TestFFGTargetRewardsPenalties(t *testing.T) {
	tests := []struct {
		voted                          []uint64
		balanceAfterTgtRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for FFG target,
		// balanceAfterTgtRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint64{}, []uint64{31999427550, 31999427550, 31999427550, 31999427550}},
		{[]uint64{0, 1}, []uint64{32000286225, 32000286225, 31999427550, 31999427550}},
		{[]uint64{0, 1, 2, 3}, []uint64{32000572450, 32000572450, 32000572450, 32000572450}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = ExpectedFFGTarget(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDeposit,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterTgtRewardPenalties) {
			t.Errorf("FFGTargetRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterTgtRewardPenalties)
		}
	}
}

func TestChainHeadRewardsPenalties(t *testing.T) {
	tests := []struct {
		voted                           []uint64
		balanceAfterHeadRewardPenalties []uint64
	}{
		// voted represents the validator indices that voted for canonical chain,
		// balanceAfterHeadRewardPenalties represents their final balances,
		// validators who voted should get an increase, who didn't should get a decrease.
		{[]uint64{}, []uint64{31999427550, 31999427550, 31999427550, 31999427550}},
		{[]uint64{0, 1}, []uint64{32000286225, 32000286225, 31999427550, 31999427550}},
		{[]uint64{0, 1, 2, 3}, []uint64{32000572450, 32000572450, 32000572450, 32000572450}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = ExpectedBeaconChainHead(
			state,
			tt.voted,
			uint64(len(tt.voted))*params.BeaconConfig().MaxDeposit,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterHeadRewardPenalties) {
			t.Errorf("ChainHeadRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterHeadRewardPenalties)
		}
	}
}

func TestInclusionDistRewards_Ok(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	var participationBitfield []byte
	// participation byte length = number of validators / target committee size / bits in a byte.
	byteLength := int(params.BeaconConfig().DepositsForChainStart / params.BeaconConfig().TargetCommitteeSize / 8)
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 0},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        5},
	}

	tests := []struct {
		voted []uint64
	}{
		{[]uint64{}},
		{[]uint64{237, 224}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, len(validators))
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			ValidatorBalances:  validatorBalances,
			LatestAttestations: attestation,
		}
		state, err := InclusionDistance(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)
		if err != nil {
			t.Fatalf("could not execute InclusionDistRewards:%v", err)
		}

		for _, i := range tt.voted {
			validatorBalances[i] = 32000055555
		}

		if !reflect.DeepEqual(state.ValidatorBalances, validatorBalances) {
			t.Errorf("InclusionDistRewards(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, validatorBalances)
		}
	}
}

func TestInclusionDistRewards_NotOk(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			AggregationBitfield: []byte{0xff}},
	}

	tests := []struct {
		voted                        []uint64
		balanceAfterInclusionRewards []uint64
	}{
		{[]uint64{0, 1, 2, 3}, []uint64{}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			LatestAttestations: attestation,
		}
		_, err := InclusionDistance(state, tt.voted, 0)
		if err == nil {
			t.Fatal("InclusionDistRewards should have failed")
		}
	}
}

func TestInactivityFFGSrcPenalty(t *testing.T) {
	tests := []struct {
		voted                     []uint64
		balanceAfterFFGSrcPenalty []uint64
		epochsSinceFinality       uint64
	}{
		// The higher the epochs since finality, the more penalties applied.
		{[]uint64{0, 1}, []uint64{32000000000, 32000000000, 31999422782, 31999422782}, 5},
		{[]uint64{}, []uint64{31999422782, 31999422782, 31999422782, 31999422782}, 5},
		{[]uint64{}, []uint64{31999418014, 31999418014, 31999418014, 31999418014}, 10},
		{[]uint64{}, []uint64{31999408477, 31999408477, 31999408477, 31999408477}, 20},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityFFGSource(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit,
			tt.epochsSinceFinality)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterFFGSrcPenalty) {
			t.Errorf("InactivityFFGSrcPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterFFGSrcPenalty)
		}
	}
}

func TestInactivityFFGTargetPenalty(t *testing.T) {
	tests := []struct {
		voted                        []uint64
		balanceAfterFFGTargetPenalty []uint64
		epochsSinceFinality          uint64
	}{
		// The higher the epochs since finality, the more penalties applied.
		{[]uint64{0, 1}, []uint64{32000000000, 32000000000, 31999422782, 31999422782}, 5},
		{[]uint64{}, []uint64{31999422782, 31999422782, 31999422782, 31999422782}, 5},
		{[]uint64{}, []uint64{31999418014, 31999418014, 31999418014, 31999418014}, 10},
		{[]uint64{}, []uint64{31999408477, 31999408477, 31999408477, 31999408477}, 20},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityFFGTarget(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit,
			tt.epochsSinceFinality)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterFFGTargetPenalty) {
			t.Errorf("InactivityFFGTargetPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, tt.balanceAfterFFGTargetPenalty)
		}
	}
}

func TestInactivityHeadPenalty(t *testing.T) {
	tests := []struct {
		voted                             []uint64
		balanceAfterInactivityHeadPenalty []uint64
	}{
		{[]uint64{}, []uint64{31999427550, 31999427550, 31999427550, 31999427550}},
		{[]uint64{0, 1}, []uint64{32000000000, 32000000000, 31999427550, 31999427550}},
		{[]uint64{0, 1, 2, 3}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityChainHead(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)

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
		{[]uint64{31998273114, 31998273114, 31998273114, 31998273114}, 5},
		{[]uint64{31998263578, 31998263578, 31998263578, 31998263578}, 10},
		{[]uint64{31997328976, 31997328976, 31997328976, 31997328976}, 500},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch}},
			ValidatorBalances: validatorBalances,
		}
		state = InactivityExitedPenalties(
			state,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit,
			tt.epochsSinceFinality,
		)

		if !reflect.DeepEqual(state.ValidatorBalances, tt.balanceAfterExitedPenalty) {
			t.Errorf("InactivityExitedPenalty(epochSinceFinality=%v) = %v, wanted: %v",
				tt.epochsSinceFinality, state.ValidatorBalances, tt.balanceAfterExitedPenalty)
		}
	}
}

func TestInactivityInclusionPenalty_Ok(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	var participationBitfield []byte
	// participation byte length = number of validators / target committee size / bits in a byte.
	byteLength := int(params.BeaconConfig().DepositsForChainStart / params.BeaconConfig().TargetCommitteeSize / 8)
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 0},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        5},
	}

	tests := []struct {
		voted []uint64
	}{
		{[]uint64{}},
		{[]uint64{237, 224}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, params.BeaconConfig().EpochLength*4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			ValidatorBalances:  validatorBalances,
			LatestAttestations: attestation,
		}
		state, err := InactivityInclusionDistance(
			state,
			tt.voted,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit)

		for _, i := range tt.voted {
			validatorBalances[i] = 32000055555
		}

		if err != nil {
			t.Fatalf("could not execute InactivityInclusionPenalty:%v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, validatorBalances) {
			t.Errorf("InactivityInclusionPenalty(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, validatorBalances)
		}
	}
}

func TestInactivityInclusionPenalty_NotOk(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			AggregationBitfield: []byte{0xff}},
	}

	tests := []struct {
		voted                        []uint64
		balanceAfterInclusionRewards []uint64
	}{
		{[]uint64{0, 1, 2, 3}, []uint64{}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			LatestAttestations: attestation,
		}
		_, err := InactivityInclusionDistance(state, tt.voted, 0)
		if err == nil {
			t.Fatal("InclusionDistRewards should have failed")
		}
	}
}

func TestAttestationInclusionRewards(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	var participationBitfield []byte
	// participation byte length = number of validators / target committee size / bits in a byte.
	byteLength := int(params.BeaconConfig().DepositsForChainStart / params.BeaconConfig().TargetCommitteeSize / 8)
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Slot: 0},
			AggregationBitfield: participationBitfield,
			SlotIncluded:        0},
	}

	tests := []struct {
		voted []uint64
	}{
		{[]uint64{}},
		{[]uint64{237}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, params.BeaconConfig().EpochLength*4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			ValidatorBalances:  validatorBalances,
			LatestAttestations: attestation,
		}
		state, err := AttestationInclusion(
			state,
			uint64(len(validatorBalances))*params.BeaconConfig().MaxDeposit,
			tt.voted)

		for _, i := range tt.voted {
			validatorBalances[i] = 32000008680
		}

		if err != nil {
			t.Fatalf("could not execute InactivityInclusionPenalty:%v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, validatorBalances) {
			t.Errorf("AttestationInclusionRewards(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, validatorBalances)
		}
	}
}

func TestAttestationInclusionRewards_NoInclusionSlot(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	tests := []struct {
		voted                            []uint64
		balanceAfterAttestationInclusion []uint64
	}{
		{[]uint64{0, 1, 2, 3}, []uint64{32000000000, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			ValidatorRegistry: validators,
			ValidatorBalances: validatorBalances,
		}
		if _, err := AttestationInclusion(state, 0, tt.voted); err == nil {
			t.Fatal("AttestationInclusionRewards should have failed with no inclusion slot")
		}
	}
}

func TestAttestationInclusionRewards_NoProposerIndex(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	attestation := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{Shard: 1, Slot: 0},
			AggregationBitfield: []byte{0xff},
			SlotIncluded:        0},
	}

	tests := []struct {
		voted                            []uint64
		balanceAfterAttestationInclusion []uint64
	}{
		{[]uint64{0}, []uint64{32000071022, 32000000000, 32000000000, 32000000000}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, 4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		state := &pb.BeaconState{
			Slot:               1000,
			ValidatorRegistry:  validators,
			ValidatorBalances:  validatorBalances,
			LatestAttestations: attestation,
		}
		if _, err := AttestationInclusion(state, 0, tt.voted); err == nil {
			t.Fatal("AttestationInclusionRewards should have failed with no proposer index")
		}
	}
}

func TestCrosslinksRewardsPenalties(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*4)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	tests := []struct {
		voted                        []byte
		balanceAfterCrosslinkRewards []uint64
	}{
		{[]byte{0x0}, []uint64{
			32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9, 32 * 1e9}},
		{[]byte{0xF}, []uint64{
			31585730498, 31585730498, 31585730498, 31585730498,
			32416931985, 32416931985, 32416931985, 32416931985}},
		{[]byte{0xFF}, []uint64{
			32829149760, 32829149760, 32829149760, 32829149760,
			32829149760, 32829149760, 32829149760, 32829149760}},
	}
	for _, tt := range tests {
		validatorBalances := make([]uint64, params.BeaconConfig().EpochLength*4)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = params.BeaconConfig().MaxDeposit
		}
		attestation := []*pb.PendingAttestationRecord{
			{Data: &pb.AttestationData{Shard: 1, Slot: 0},
				AggregationBitfield: tt.voted,
				SlotIncluded:        0},
		}
		state := &pb.BeaconState{
			ValidatorRegistry:  validators,
			ValidatorBalances:  validatorBalances,
			LatestAttestations: attestation,
		}
		state, err := Crosslinks(
			state,
			attestation,
			nil)
		if err != nil {
			t.Fatalf("Could not apply Crosslinks rewards: %v", err)
		}
		if !reflect.DeepEqual(state.ValidatorBalances, validatorBalances) {
			t.Errorf("CrosslinksRewardsPenalties(%v) = %v, wanted: %v",
				tt.voted, state.ValidatorBalances, validatorBalances)
		}
	}
}
