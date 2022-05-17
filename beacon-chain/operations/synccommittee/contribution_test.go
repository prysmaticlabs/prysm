package synccommittee

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSyncCommitteeContributionCache_Nil(t *testing.T) {
	store := NewStore()
	require.Equal(t, errNilContribution, store.SaveSyncCommitteeContribution(nil))
}

func TestSyncCommitteeContributionCache_RoundTrip(t *testing.T) {
	store := NewStore()

	conts := []*ethpb.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, Signature: []byte{'a'}},
		{Slot: 1, SubcommitteeIndex: 1, Signature: []byte{'b'}},
		{Slot: 2, SubcommitteeIndex: 0, Signature: []byte{'c'}},
		{Slot: 2, SubcommitteeIndex: 1, Signature: []byte{'d'}},
		{Slot: 3, SubcommitteeIndex: 0, Signature: []byte{'e'}},
		{Slot: 3, SubcommitteeIndex: 1, Signature: []byte{'f'}},
		{Slot: 4, SubcommitteeIndex: 0, Signature: []byte{'g'}},
		{Slot: 4, SubcommitteeIndex: 1, Signature: []byte{'h'}},
		{Slot: 5, SubcommitteeIndex: 0, Signature: []byte{'i'}},
		{Slot: 5, SubcommitteeIndex: 1, Signature: []byte{'j'}},
		{Slot: 6, SubcommitteeIndex: 0, Signature: []byte{'k'}},
		{Slot: 6, SubcommitteeIndex: 1, Signature: []byte{'l'}},
	}

	for _, sig := range conts {
		require.NoError(t, store.SaveSyncCommitteeContribution(sig))
	}

	conts, err := store.SyncCommitteeContributions(1)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{}, conts)

	conts, err = store.SyncCommitteeContributions(2)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{}, conts)

	conts, err = store.SyncCommitteeContributions(3)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 3, SubcommitteeIndex: 0, Signature: []byte{'e'}},
		{Slot: 3, SubcommitteeIndex: 1, Signature: []byte{'f'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(4)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 4, SubcommitteeIndex: 0, Signature: []byte{'g'}},
		{Slot: 4, SubcommitteeIndex: 1, Signature: []byte{'h'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(5)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 5, SubcommitteeIndex: 0, Signature: []byte{'i'}},
		{Slot: 5, SubcommitteeIndex: 1, Signature: []byte{'j'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(6)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 6, SubcommitteeIndex: 0, Signature: []byte{'k'}},
		{Slot: 6, SubcommitteeIndex: 1, Signature: []byte{'l'}},
	}, conts)

	// All the contributions should persist after get.
	conts, err = store.SyncCommitteeContributions(1)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{}, conts)
	conts, err = store.SyncCommitteeContributions(2)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{}, conts)

	conts, err = store.SyncCommitteeContributions(3)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 3, SubcommitteeIndex: 0, Signature: []byte{'e'}},
		{Slot: 3, SubcommitteeIndex: 1, Signature: []byte{'f'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(4)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 4, SubcommitteeIndex: 0, Signature: []byte{'g'}},
		{Slot: 4, SubcommitteeIndex: 1, Signature: []byte{'h'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(5)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 5, SubcommitteeIndex: 0, Signature: []byte{'i'}},
		{Slot: 5, SubcommitteeIndex: 1, Signature: []byte{'j'}},
	}, conts)

	conts, err = store.SyncCommitteeContributions(6)
	require.NoError(t, err)
	require.DeepSSZEqual(t, []*ethpb.SyncCommitteeContribution{
		{Slot: 6, SubcommitteeIndex: 0, Signature: []byte{'k'}},
		{Slot: 6, SubcommitteeIndex: 1, Signature: []byte{'l'}},
	}, conts)
}

func TestSyncCommitteeContributionCache_HasSyncCommitteeContribution(t *testing.T) {
	store := NewStore()

	b0 := bitfield.NewBitvector128()
	b0.SetBitAt(0, true)
	b1 := bitfield.NewBitvector128()
	b1.SetBitAt(1, true)
	b2 := bitfield.NewBitvector128()
	b2.SetBitAt(0, true)
	b2.SetBitAt(1, true)
	b2.SetBitAt(2, true)

	conts := []*ethpb.SyncCommitteeContribution{
		{Slot: 1, SubcommitteeIndex: 0, AggregationBits: b0},
		{Slot: 1, SubcommitteeIndex: 1, AggregationBits: b1},
		{Slot: 1, SubcommitteeIndex: 1, BlockRoot: []byte{'a'}, AggregationBits: b1},
		{Slot: 1, SubcommitteeIndex: 2, AggregationBits: b2},
	}

	for _, sig := range conts {
		require.NoError(t, store.SaveSyncCommitteeContribution(sig))
	}

	type test struct {
		name  string
		input *ethpb.SyncCommitteeContribution
		want  bool
	}
	tests := []test{
		{
			name: "slot 0, has = false",
			input: &ethpb.SyncCommitteeContribution{
				AggregationBits: b2,
			},
			want: false,
		},
		{
			name: "slot 1, same root, bit overlaps, has = true",
			input: &ethpb.SyncCommitteeContribution{
				Slot:            1,
				AggregationBits: b0,
			},
			want: true,
		},
		{
			name: "slot 1, same root, bit doesn't overlaps, has = false",
			input: &ethpb.SyncCommitteeContribution{
				Slot:            1,
				AggregationBits: b1,
			},
			want: false,
		},
		{
			name: "slot 1, same root, bit doesn't overlaps, has = false",
			input: &ethpb.SyncCommitteeContribution{
				Slot:            1,
				AggregationBits: b2,
			},
			want: false,
		},
		{
			name: "slot 1, same root, bit overlaps, has = true",
			input: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 1,
				AggregationBits:   b1,
			},
			want: true,
		},
		{
			name: "slot 1, different root, bit overlaps, has = true",
			input: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 1,
				BlockRoot:         []byte{'a'},
				AggregationBits:   b1,
			},
			want: true,
		},
		{
			name: "slot 1, different root, different committee, bit doesn't overlaps, has = false",
			input: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 3,
				BlockRoot:         []byte{'a'},
				AggregationBits:   b1,
			},
			want: false,
		},
		{
			name: "slot 1, different root, bit doesn't overlaps, has = false",
			input: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 1,
				BlockRoot:         []byte{'b'},
				AggregationBits:   b1,
			},
			want: false,
		},
		{
			name: "slot 1, same root, bit doesn't overlaps, has = true",
			input: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 1,
				AggregationBits:   b2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.HasSyncCommitteeContribution(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
