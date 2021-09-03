package kv

import (
	"sort"
	"testing"

	fssz "github.com/ferranbt/fastssz"
	c "github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestKV_Aggregated_SaveOrphanedAggregatedAttestation(t *testing.T) {
	tests := []struct {
		name          string
		att           *ethpb.Attestation
		count         int
		wantErrString string
	}{
		{
			name:          "nil attestation",
			att:           nil,
			wantErrString: "attestation can't be nil",
		},
		{
			name:          "nil attestation data",
			att:           &ethpb.Attestation{},
			wantErrString: "attestation's data can't be nil",
		},
		{
			name: "not aggregated",
			att: testutil.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{}, AggregationBits: bitfield.Bitlist{0b10100}}),
			wantErrString: "attestation is not aggregated",
		},
		{
			name: "invalid hash",
			att: &ethpb.Attestation{
				Data: testutil.HydrateAttestationData(&ethpb.AttestationData{
					BeaconBlockRoot: []byte{0b0},
				}),
				AggregationBits: bitfield.Bitlist{0b10111},
			},
			wantErrString: "could not tree hash attestation: " + fssz.ErrBytesLength.Error(),
		},
		{
			name: "already seen",
			att: testutil.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 100,
				},
				AggregationBits: bitfield.Bitlist{0b11101001},
			}),
			count: 0,
		},
		{
			name: "normal save",
			att: testutil.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1101},
			}),
			count: 1,
		},
	}
	r, err := hashFn(testutil.HydrateAttestationData(&ethpb.AttestationData{
		Slot: 100,
	}))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			cache.seenAtt.Set(string(r[:]), []bitfield.Bitlist{{0xff}}, c.DefaultExpiration)

			err := cache.SaveOrphanedAggregatedAttestation(tt.att)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, tt.wantErrString, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(cache.orphanedAggregatedAtt), "Wrong attestation count")
		})
	}
}

func TestKV_Aggregated_OrphanedAggregatedAttestations(t *testing.T) {
	cache := NewAttCaches()

	att1 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}})
	att2 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}})
	att3 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}})
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		require.NoError(t, cache.SaveOrphanedAggregatedAttestation(att))
	}

	returned := cache.OrphanedAggregatedAttestations()
	sort.Slice(returned, func(i, j int) bool {
		return returned[i].Data.Slot < returned[j].Data.Slot
	})
	assert.DeepSSZEqual(t, atts, returned)
}

func TestKV_Aggregated_DeleteOrphanedAggregatedAttestation(t *testing.T) {
	t.Run("nil attestation", func(t *testing.T) {
		cache := NewAttCaches()
		assert.ErrorContains(t, "attestation can't be nil", cache.DeleteOrphanedAggregatedAttestation(nil))
		att := testutil.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10101}, Data: &ethpb.AttestationData{Slot: 2}})
		assert.NoError(t, cache.DeleteOrphanedAggregatedAttestation(att))
	})

	t.Run("non aggregated attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := testutil.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}, Data: &ethpb.AttestationData{Slot: 2}})
		err := cache.DeleteOrphanedAggregatedAttestation(att)
		assert.ErrorContains(t, "attestation is not aggregated", err)
	})

	t.Run("invalid hash", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{
			AggregationBits: bitfield.Bitlist{0b1111},
			Data: &ethpb.AttestationData{
				Slot:   2,
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{},
			},
		}
		err := cache.DeleteOrphanedAggregatedAttestation(att)
		wantErr := "could not tree hash attestation data: " + fssz.ErrBytesLength.Error()
		assert.ErrorContains(t, wantErr, err)
	})

	t.Run("nonexistent attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := testutil.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2}})
		assert.NoError(t, cache.DeleteOrphanedAggregatedAttestation(att))
	})

	t.Run("filtered deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b110101}})
		att2 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110111}})
		att3 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110100}})
		att4 := testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110101}})
		atts := []*ethpb.Attestation{att1, att2, att3, att4}
		for _, att := range atts {
			require.NoError(t, cache.SaveOrphanedAggregatedAttestation(att))
		}
		require.NoError(t, cache.DeleteOrphanedAggregatedAttestation(att4))

		returned := cache.OrphanedAggregatedAttestations()
		wanted := []*ethpb.Attestation{att1, att2}
		sort.Slice(returned, func(i, j int) bool {
			return string(returned[i].AggregationBits) < string(returned[j].AggregationBits)
		})
		assert.DeepEqual(t, wanted, returned)
	})
}
