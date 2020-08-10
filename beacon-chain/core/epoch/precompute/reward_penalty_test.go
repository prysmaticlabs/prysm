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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err)

	vp, bp, err := New(context.Background(), state)
	require.NoError(t, err)
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	require.NoError(t, err)

	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	require.NoError(t, err)

	// Indices that voted everything except for head, lost a bit money
	wanted := uint64(31999810265)
	assert.Equal(t, wanted, state.Balances()[4], "Unexpected balance")

	// Indices that did not vote, lost more money
	wanted = uint64(31999873505)
	assert.Equal(t, wanted, state.Balances()[0], "Unexpected balance")
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
	require.NoError(t, err)
	slashedAttestedIndices := []uint64{1413}
	for _, i := range slashedAttestedIndices {
		vs := state.Validators()
		vs[i].Slashed = true
		require.Equal(t, nil, state.SetValidators(vs))
	}

	vp, bp, err := New(context.Background(), state)
	require.NoError(t, err)
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	require.NoError(t, err)

	// Add some variances to target and head balances.
	// See: https://github.com/prysmaticlabs/prysm/issues/5593
	bp.PrevEpochTargetAttested = bp.PrevEpochTargetAttested / 2
	bp.PrevEpochHeadAttested = bp.PrevEpochHeadAttested * 2 / 3
	rewards, penalties, err := AttestationsDelta(state, bp, vp)
	require.NoError(t, err)
	attestedBalance, err := epoch.AttestingBalance(state, atts)
	require.NoError(t, err)
	totalBalance, err := helpers.TotalActiveBalance(state)
	require.NoError(t, err)

	attestedIndices := []uint64{55, 1339, 1746, 1811, 1569}
	for _, i := range attestedIndices {
		base, err := epoch.BaseReward(state, i)
		require.NoError(t, err, "Could not get base reward")

		// Base rewards for getting source right
		wanted := attestedBalance*base/totalBalance +
			bp.PrevEpochTargetAttested*base/totalBalance +
			bp.PrevEpochHeadAttested*base/totalBalance
		// Base rewards for proposer and attesters working together getting attestation
		// on chain in the fatest manner
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		wanted += (base-proposerReward)*params.BeaconConfig().MinAttestationInclusionDelay - 1
		assert.Equal(t, wanted, rewards[i], "Unexpected reward balance for validator with index %d", i)
		// Since all these validators attested, they shouldn't get penalized.
		assert.Equal(t, uint64(0), penalties[i], "Unexpected penalty balance")
	}

	for _, i := range slashedAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		assert.NoError(t, err, "Could not get base reward")
		assert.Equal(t, uint64(0), rewards[i], "Unexpected slashed indices reward balance")
		assert.Equal(t, uint64(3*base), penalties[i], "Unexpected slashed indices penalty balance")
	}

	nonAttestedIndices := []uint64{434, 677, 872, 791}
	for _, i := range nonAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		assert.NoError(t, err, "Could not get base reward")
		wanted := 3 * base
		// Since all these validators did not attest, they shouldn't get rewarded.
		assert.Equal(t, uint64(0), rewards[i], "Unexpected reward balance")
		// Base penalties for not attesting.
		assert.Equal(t, wanted, penalties[i], "Unexpected penalty balance")
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
	require.NoError(t, err)

	pVals, pBal, err := New(context.Background(), state)
	assert.NoError(t, err)
	pVals, pBal, err = ProcessAttestations(context.Background(), state, pVals, pBal)
	require.NoError(t, err)

	pBal.ActiveCurrentEpoch = 0 // Could cause a divide by zero panic.

	_, _, err = AttestationsDelta(state, pBal, pVals)
	require.NoError(t, err)
}

func TestAttestationDeltas_ZeroInclusionDelay(t *testing.T) {
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
			// Inclusion delay of 0 is not possible in a valid state and could cause a divide by
			// zero panic.
			InclusionDelay: 0,
		}
	}
	base.PreviousEpochAttestations = atts
	state, err := state.InitializeFromProto(base)
	require.NoError(t, err)

	pVals, pBal, err := New(context.Background(), state)
	require.NoError(t, err)
	pVals, pBal, err = ProcessAttestations(context.Background(), state, pVals, pBal)
	require.ErrorContains(t, "attestation with inclusion delay of 0", err)
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
	require.NoError(t, err)
	require.NoError(t, state.SetSlot(params.BeaconConfig().SlotsPerEpoch*10))

	slashedAttestedIndices := []uint64{14, 37, 68, 77, 139}
	for _, i := range slashedAttestedIndices {
		vs := state.Validators()
		vs[i].Slashed = true
		require.NoError(t, state.SetValidators(vs))
	}

	vp, bp, err := New(context.Background(), state)
	require.NoError(t, err)
	vp, bp, err = ProcessAttestations(context.Background(), state, vp, bp)
	require.NoError(t, err)
	rewards, penalties, err := AttestationsDelta(state, bp, vp)
	require.NoError(t, err)

	finalityDelay := helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	for _, i := range slashedAttestedIndices {
		base, err := epoch.BaseReward(state, i)
		require.NoError(t, err, "Could not get base reward")
		penalty := 3 * base
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		penalty += params.BeaconConfig().BaseRewardsPerEpoch*base - proposerReward
		penalty += vp[i].CurrentEpochEffectiveBalance * finalityDelay / params.BeaconConfig().InactivityPenaltyQuotient
		assert.Equal(t, penalty, penalties[i], "Unexpected slashed indices penalty balance")
		assert.Equal(t, uint64(0), rewards[i], "Unexpected slashed indices reward balance")
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
	require.NoError(t, err)

	proposerIndex := uint64(1)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	r, err := ProposersDelta(state, b, v)
	require.NoError(t, err)

	baseReward := v[0].CurrentEpochEffectiveBalance * params.BeaconConfig().BaseRewardFactor /
		mathutil.IntegerSquareRoot(b.ActiveCurrentEpoch) / params.BeaconConfig().BaseRewardsPerEpoch
	proposerReward := baseReward / params.BeaconConfig().ProposerRewardQuotient

	assert.Equal(t, proposerReward, r[proposerIndex], "Unexpected proposer reward")
}

func TestProposerDeltaPrecompute_ValidatorIndexOutOfRange(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	state, err := state.InitializeFromProto(base)
	require.NoError(t, err)

	proposerIndex := validatorCount
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex},
	}
	_, err = ProposersDelta(state, b, v)
	assert.ErrorContains(t, "proposer index out of range", err)
}

func TestProposerDeltaPrecompute_SlashedCase(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := uint64(10)
	base := buildState(e, validatorCount)
	state, err := state.InitializeFromProto(base)
	require.NoError(t, err)

	proposerIndex := uint64(1)
	b := &Balance{ActiveCurrentEpoch: 1000}
	v := []*Validator{
		{IsPrevEpochAttester: true, CurrentEpochEffectiveBalance: 32, ProposerIndex: proposerIndex, IsSlashed: true},
	}
	r, err := ProposersDelta(state, b, v)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), r[proposerIndex], "Unexpected proposer reward for slashed")
}

func TestFinalityDelay(t *testing.T) {
	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	state, err := state.InitializeFromProto(base)
	require.NoError(t, err)
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
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	d = finalityDelay(prevEpoch, finalizedEpoch)
	w = helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")

	require.NoError(t, state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	d = finalityDelay(prevEpoch, finalizedEpoch)
	w = helpers.PrevEpoch(state) - state.FinalizedCheckpointEpoch()
	assert.Equal(t, w, d, "Did not get wanted finality delay")
}

func TestIsInInactivityLeak(t *testing.T) {
	base := buildState(params.BeaconConfig().SlotsPerEpoch*10, 1)
	base.FinalizedCheckpoint = &ethpb.Checkpoint{Epoch: 3}
	state, err := state.InitializeFromProto(base)
	require.NoError(t, err)
	prevEpoch := uint64(0)
	finalizedEpoch := uint64(0)
	// Set values for each test case
	setVal := func() {
		prevEpoch = helpers.PrevEpoch(state)
		finalizedEpoch = state.FinalizedCheckpointEpoch()
	}
	setVal()
	assert.Equal(t, true, isInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 4}))
	setVal()
	assert.Equal(t, true, isInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak true")
	require.NoError(t, state.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 5}))
	setVal()
	assert.Equal(t, false, isInInactivityLeak(prevEpoch, finalizedEpoch), "Wanted inactivity leak false")
}
