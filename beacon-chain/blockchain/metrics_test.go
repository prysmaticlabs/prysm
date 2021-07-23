package blockchain

import (
	"context"
	"testing"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestReportEpochMetrics_BadHeadState(t *testing.T) {
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	h, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, h.SetValidators(nil))
	err = reportEpochMetrics(context.Background(), s, h)
	require.ErrorContains(t, "failed to initialize precompute: nil validators in state", err)
}

func TestReportEpochMetrics_BadAttestation(t *testing.T) {
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	h, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, h.AppendCurrentEpochAttestations(&statepb.PendingAttestation{InclusionDelay: 0}))
	err = reportEpochMetrics(context.Background(), s, h)
	require.ErrorContains(t, "attestation with inclusion delay of 0", err)
}

func TestReportEpochMetrics_SlashedValidatorOutOfBound(t *testing.T) {
	h, _ := testutil.DeterministicGenesisState(t, 1)
	v, err := h.ValidatorAtIndex(0)
	require.NoError(t, err)
	v.Slashed = true
	require.NoError(t, h.UpdateValidatorAtIndex(0, v))
	require.NoError(t, h.AppendCurrentEpochAttestations(&statepb.PendingAttestation{InclusionDelay: 1, Data: testutil.HydrateAttestationData(&eth.AttestationData{})}))
	err = reportEpochMetrics(context.Background(), h, h)
	require.ErrorContains(t, "slot 0 out of bounds", err)
}
