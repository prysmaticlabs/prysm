package altair_test

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestTranslateParticipation(t *testing.T) {
	s, _ := testutil.DeterministicGenesisStateAltair(t, 64)
	st, ok := s.(*stateAltair.BeaconState)
	require.Equal(t, true, ok)
	require.NoError(t, st.SetSlot(st.Slot()+params.BeaconConfig().MinAttestationInclusionDelay))

	var err error
	newState, err := altair.TranslateParticipation(st, nil)
	require.NoError(t, err)
	participation, err := newState.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]byte, 64), participation)

	aggBits := bitfield.NewBitlist(2)
	aggBits.SetBitAt(0, true)
	aggBits.SetBitAt(1, true)
	r, err := helpers.BlockRootAtSlot(s, 0)
	require.NoError(t, err)
	var pendingAtts []*ethpb.PendingAttestation
	for i := 0; i < 3; i++ {
		pendingAtts = append(pendingAtts, &ethpb.PendingAttestation{
			Data: &ethpb.AttestationData{
				CommitteeIndex:  types.CommitteeIndex(i),
				BeaconBlockRoot: r,
				Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				Target:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			},
			AggregationBits: aggBits,
			InclusionDelay:  1,
		})
	}

	newState, err = altair.TranslateParticipation(newState, pendingAtts)
	require.NoError(t, err)
	participation, err = newState.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepNotSSZEqual(t, make([]byte, 64), participation)

	committee, err := helpers.BeaconCommitteeFromState(st, pendingAtts[0].Data.Slot, pendingAtts[0].Data.CommitteeIndex)
	require.NoError(t, err)
	indices, err := attestationutil.AttestingIndices(pendingAtts[0].AggregationBits, committee)
	require.NoError(t, err)
	for _, index := range indices {
		require.Equal(t, true, altair.HasValidatorFlag(participation[index], params.BeaconConfig().TimelyHeadFlagIndex))
		require.Equal(t, true, altair.HasValidatorFlag(participation[index], params.BeaconConfig().TimelyTargetFlagIndex))
		require.Equal(t, true, altair.HasValidatorFlag(participation[index], params.BeaconConfig().TimelySourceFlagIndex))
	}
}

func TestUpgradeToAltair(t *testing.T) {
	st, _ := testutil.DeterministicGenesisState(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	aState, err := altair.UpgradeToAltair(context.Background(), st)
	require.NoError(t, err)
	_, ok := aState.(state.BeaconStateAltair)
	require.Equal(t, true, ok)

	f := aState.Fork()
	require.DeepSSZEqual(t, &ethpb.Fork{
		PreviousVersion: st.Fork().CurrentVersion,
		CurrentVersion:  params.BeaconConfig().AltairForkVersion,
		Epoch:           helpers.CurrentEpoch(st),
	}, f)
	csc, err := aState.CurrentSyncCommittee()
	require.NoError(t, err)
	nsc, err := aState.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepSSZEqual(t, nsc, csc)
}
