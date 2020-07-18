package precompute

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessRewardsAndPenaltiesPrecompute(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+3, validatorCount)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts

	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	vp, bp, err := New(context.Background(), state)
	if err != nil {
		t.Error(err)
	}
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		t.Fatal(err)
	}

	// Indices that voted everything except for head, lost a bit money
	wanted := uint64(31999810265)
	if state.Balances()[4] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances()[4])
	}

	// Indices that did not vote, lost more money
	wanted = uint64(31999873505)
	if state.Balances()[0] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances()[0])
	}
}

func TestAttestationDeltaPrecompute(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+2, validatorCount)
	atts := make([]*pb.PendingAttestation, 3)
	var emptyRoot [32]byte
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				Source: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				BeaconBlockRoot: emptyRoot[:],
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	slashedAttestedIndices := []uint64{1413}
	for _, i := range slashedAttestedIndices {
		vs := state.Validators()
		vs[i].Slashed = true
		if state.SetValidators(vs) != nil {
			t.Fatal(err)
		}
	}

	vp, bp, err := New(context.Background(), state)
	if err != nil {
		t.Error(err)
	}
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	// Add some variances to target and head balances.
	// See: https://github.com/prysmaticlabs/prysm/issues/5593
	bp.PrevEpochTargetAttested = bp.PrevEpochTargetAttested / 2
	bp.PrevEpochHeadAttested = bp.PrevEpochHeadAttested * 2 / 3
	rewards, penalties, err := AttestationsDelta(state, bp, vp)
	if err != nil {
		t.Fatal(err)
	}
	attestedBalance, err := epoch.AttestingBalance(state, atts)
	if err != nil {
		t.Error(err)
	}
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedIndices := []uint64{55, 1339, 1746, 1811, 1569}
	for _, i := range attestedIndices {
		base, err := epoch.BaseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}

		// Base rewards for getting source right
		wanted := attestedBalance*base/totalBalance +
			bp.PrevEpochTargetAttested*base/totalBalance +
			bp.PrevEpochHeadAttested*base/totalBalance
		// Base rewards for proposer and attesters working together getting attestation
		// on chain in the fatest manner
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		wanted += (base-proposerReward)*params.BeaconConfig().MinAttestationInclusionDelay - 1
		if rewards[i] != wanted {
			t.Errorf("Wanted reward balance %d, got %d for validator with index %d", wanted, rewards[i], i)
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance 0, got %d", penalties[i])
		}
	}

	for _, i := range slashedAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		if rewards[i] != 0 {
			t.Errorf("Wanted slashed indices reward balance 0, got %d", penalties[i])
		}
		if penalties[i] != 3*base {
			t.Errorf("Wanted slashed indices penalty balance %d, got %d", 3*base, penalties[i])
		}
	}

	nonAttestedIndices := []uint64{434, 677, 872, 791}
	for _, i := range nonAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		wanted := 3 * base
		// Since all these validators did not attest, they shouldn't get rewarded.
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Base penalties for not attesting.
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d", wanted, penalties[i])
		}
	}
}

func TestAttestationDeltas_ZeroEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+2, validatorCount)
	atts := make([]*pb.PendingAttestation, 3)
	var emptyRoot [32]byte
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				Source: &ethpb.Checkpoint{
					Root: emptyRoot[:],
				},
				BeaconBlockRoot: emptyRoot[:],
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	pVals, pBal, err := New(context.Background(), state)
	if err != nil {
		t.Error(err)
	}
	pVals, pBal, err = ProcessAttestations(context.Background(), state, pVals, pBal)
	if err != nil {
		t.Fatal(err)
	}

	pBal.ActiveCurrentEpoch = 0 // Could cause a divide by zero panic.

	_, _, err = AttestationsDelta(state, pBal, pVals)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessRewardsAndPenaltiesPrecompute_SlashedInactivePenalty(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(2048)
	base := buildState(e+3, validatorCount)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	base.PreviousEpochAttestations = atts

	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.SetSlot(params.BeaconConfig().SlotsPerEpoch * 10); err != nil {
		t.Fatal(err)
	}

	slashedAttestedIndices := []uint64{14, 37, 68, 77, 139}
	for _, i := range slashedAttestedIndices {
		vs := state.Validators()
		vs[i].Slashed = true
		if state.SetValidators(vs) != nil {
			t.Fatal(err)
		}
	}

	vp, bp, err := New(context.Background(), state)
	if err != nil {
		t.Error(err)
	}
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	if err != nil {
		t.Fatal(err)
	}
	rewards, penalties, err := AttestationsDelta(state, bp, vp)
	if err != nil {
		t.Fatal(err)
	}

	finalityDelay := helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	for _, i := range slashedAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		penalty := 3 * base
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		penalty += params.BeaconConfig().BaseRewardsPerEpoch*base - proposerReward
		penalty += vp[i].CurrentEpochEffectiveBalance * finalityDelay / params.BeaconConfig().InactivityPenaltyQuotient
		if penalties[i] != penalty {
			t.Errorf("Wanted slashed indices penalty balance %d, got %d", penalty, penalties[i])
		}

		if rewards[i] != 0 {
			t.Errorf("Wanted slashed indices reward balance 0, got %d", penalties[i])
		}
	}
}

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &pb.BeaconState{
		Slot:                        slot,
		Balances:                    validatorBalances,
		Validators:                  validators,
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{},
	}
}

func TestProposerDeltaPrecompute_HappyCase(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	proposerIndex := uint64(1)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	r, err := ProposersDelta(state, b, v)
	if err != nil {
		t.Fatal(err)
	}

	baseReward := v[0].CurrentEpochEffectiveBalance * params.BeaconConfig().BaseRewardFactor /
		mathutil.IntegerSquareRoot(b.ActiveCurrentEpoch) / params.BeaconConfig().BaseRewardsPerEpoch
	proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient

	if r[proposerIndex] != proposerReward {
		t.Errorf("Wanted proposer reward %d, got %d", proposerReward, r[proposerIndex])
	}
}

func TestProposerDeltaPrecompute_ValidatorIndexOutOfRange(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	proposerIndex := validatorCount + 1
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	_, err = ProposersDelta(state, b, v)
	if err == nil {
		t.Fatal("Expected an error with invalid proposer index")
	}
}

func TestProposerDeltaPrecompute_SlashedCase(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	proposerIndex := uint64(1)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex, IsSlashed: true},
	}
	r, err := ProposersDelta(state, b, v)
	if err != nil {
		t.Fatal(err)
	}

	if r[proposerIndex] != 0 {
		t.Errorf("Wanted proposer reward for slashed %d, got %d", 0, r[proposerIndex])
	}
}

func TestFinalityDelay(t *testing.T) {
	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	prevEpoch := uint64(0)
	finalizedEpoch := uint64(0)
	// Set values for each test case
	setVal := func() {
		prevEpoch = helpers.PrevEpoch(state)
		finalizedEpoch = state.FinalizedCheckpointEpoch()
	}
	setVal()
	d := finalityDelay(prevEpoch, finalizedEpoch)
	w := helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	if d != w {
		t.Error("Did not get wanted finality delay")
	}

	if err := state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}); err != nil {
		t.Fatal(err)
	}
	setVal()
	d = finalityDelay(prevEpoch, finalizedEpoch)
	w = helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	if d != w {
		t.Error("Did not get wanted finality delay")
	}

	if err := state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}); err != nil {
		t.Fatal(err)
	}
	setVal()
	d = finalityDelay(prevEpoch, finalizedEpoch)
	w = helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	if d != w {
		t.Error("Did not get wanted finality delay")
	}
}

func TestIsInInactivityLeak(t *testing.T) {
	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	prevEpoch := uint64(0)
	finalizedEpoch := uint64(0)
	// Set values for each test case
	setVal := func() {
		prevEpoch = helpers.PrevEpoch(state)
		finalizedEpoch = state.FinalizedCheckpointEpoch()
	}
	setVal()
	if !isInInactivityLeak(prevEpoch, finalizedEpoch) {
		t.Error("Wanted inactivity leak true")
	}

	if err := state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}); err != nil {
		t.Fatal(err)
	}
	setVal()
	if !isInInactivityLeak(prevEpoch, finalizedEpoch) {
		t.Error("Wanted inactivity leak true")
	}

	if err := state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}); err != nil {
		t.Fatal(err)
	}
	setVal()
	if isInInactivityLeak(prevEpoch, finalizedEpoch) {
		t.Error("Wanted inactivity leak false")
	}
}
