package kv

import (
	"bytes"
	"context"
	"sort"
	"testing"

	c "github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestKV_Unaggregated_SaveUnaggregatedAttestation(t *testing.T) {
	tests := []struct {
		name          string
		att           ethpb.Att
		count         int
		wantErrString string
	}{
		{
			name: "nil attestation",
			att:  nil,
		},
		{
			name:          "already aggregated",
			att:           &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10101}, Data: &ethpb.AttestationData{Slot: 2}},
			wantErrString: "attestation is aggregated",
		},
		{
			name: "invalid hash",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: []byte{0b0},
				},
			},
			wantErrString: "could not create attestation ID",
		},
		{
			name:  "normal save",
			att:   util.HydrateAttestation(&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b0001}}),
			count: 1,
		},
		{
			name: "already seen",
			att: util.HydrateAttestation(&ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 100,
				},
				AggregationBits: bitfield.Bitlist{0b10000001},
			}),
			count: 0,
		},
	}
	id, err := attestation.NewId(util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100}}), attestation.Data)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			cache.seenAtt.Set(id.String(), []bitfield.Bitlist{{0xff}}, c.DefaultExpiration)
			assert.Equal(t, 0, len(cache.unAggregatedAtt), "Invalid start pool, atts: %d", len(cache.unAggregatedAtt))

			if tt.att != nil && tt.att.GetSignature() == nil {
				tt.att.(*ethpb.Attestation).Signature = make([]byte, fieldparams.BLSSignatureLength)
			}

			err := cache.SaveUnaggregatedAttestation(tt.att)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, tt.wantErrString, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(cache.unAggregatedAtt), "Wrong attestation count")
			assert.Equal(t, tt.count, cache.UnaggregatedAttestationCount(), "Wrong attestation count")
		})
	}
}

func TestKV_Unaggregated_SaveUnaggregatedAttestations(t *testing.T) {
	tests := []struct {
		name          string
		atts          []ethpb.Att
		count         int
		wantErrString string
	}{
		{
			name: "unaggregated only",
			atts: []ethpb.Att{
				util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}}),
				util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}}),
				util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}}),
			},
			count: 3,
		},
		{
			name: "has aggregated",
			atts: []ethpb.Att{
				util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}}),
				&ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2}},
				util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}}),
			},
			wantErrString: "attestation is aggregated",
			count:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			assert.Equal(t, 0, len(cache.unAggregatedAtt), "Invalid start pool, atts: %d", len(cache.unAggregatedAtt))

			err := cache.SaveUnaggregatedAttestations(tt.atts)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, tt.wantErrString, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.count, len(cache.unAggregatedAtt), "Wrong attestation count")
			assert.Equal(t, tt.count, cache.UnaggregatedAttestationCount(), "Wrong attestation count")
		})
	}
}

func TestKV_Unaggregated_DeleteUnaggregatedAttestation(t *testing.T) {
	t.Run("nil attestation", func(t *testing.T) {
		cache := NewAttCaches()
		assert.NoError(t, cache.DeleteUnaggregatedAttestation(nil))
	})

	t.Run("aggregated attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2}}
		err := cache.DeleteUnaggregatedAttestation(att)
		assert.ErrorContains(t, "attestation is aggregated", err)
	})

	t.Run("successful deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b101}})
		att2 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110}})
		att3 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b110}})
		atts := []ethpb.Att{att1, att2, att3}
		require.NoError(t, cache.SaveUnaggregatedAttestations(atts))
		for _, att := range atts {
			assert.NoError(t, cache.DeleteUnaggregatedAttestation(att))
		}
		returned, err := cache.UnaggregatedAttestations()
		require.NoError(t, err)
		assert.DeepEqual(t, []ethpb.Att{}, returned)
	})
}

func TestKV_Unaggregated_DeleteSeenUnaggregatedAttestations(t *testing.T) {
	d := util.HydrateAttestationData(&ethpb.AttestationData{})

	t.Run("no attestations", func(t *testing.T) {
		cache := NewAttCaches()
		count, err := cache.DeleteSeenUnaggregatedAttestations()
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("none seen", func(t *testing.T) {
		cache := NewAttCaches()
		atts := []ethpb.Att{
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1001}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1010}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1100}}),
		}
		require.NoError(t, cache.SaveUnaggregatedAttestations(atts))
		assert.Equal(t, 3, cache.UnaggregatedAttestationCount())

		// As none of attestations have been marked seen, nothing should be deleted.
		count, err := cache.DeleteSeenUnaggregatedAttestations()
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, 3, cache.UnaggregatedAttestationCount())
	})

	t.Run("some seen", func(t *testing.T) {
		cache := NewAttCaches()
		atts := []ethpb.Att{
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1001}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1010}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1100}}),
		}
		require.NoError(t, cache.SaveUnaggregatedAttestations(atts))
		assert.Equal(t, 3, cache.UnaggregatedAttestationCount())

		require.NoError(t, cache.insertSeenBit(atts[1]))

		// Only seen attestations must be deleted.
		count, err := cache.DeleteSeenUnaggregatedAttestations()
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, 2, cache.UnaggregatedAttestationCount())
		returned, err := cache.UnaggregatedAttestations()
		sort.Slice(returned, func(i, j int) bool {
			return bytes.Compare(returned[i].GetAggregationBits(), returned[j].GetAggregationBits()) < 0
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []ethpb.Att{atts[0], atts[2]}, returned)
	})

	t.Run("all seen", func(t *testing.T) {
		cache := NewAttCaches()
		atts := []ethpb.Att{
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1001}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1010}}),
			util.HydrateAttestation(&ethpb.Attestation{Data: d, AggregationBits: bitfield.Bitlist{0b1100}}),
		}
		require.NoError(t, cache.SaveUnaggregatedAttestations(atts))
		assert.Equal(t, 3, cache.UnaggregatedAttestationCount())

		require.NoError(t, cache.insertSeenBit(atts[0]))
		require.NoError(t, cache.insertSeenBit(atts[1]))
		require.NoError(t, cache.insertSeenBit(atts[2]))

		// All attestations have been processed -- all should be removed.
		count, err := cache.DeleteSeenUnaggregatedAttestations()
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Equal(t, 0, cache.UnaggregatedAttestationCount())
		returned, err := cache.UnaggregatedAttestations()
		require.NoError(t, err)
		assert.DeepEqual(t, []ethpb.Att{}, returned)
	})
}

func TestKV_Unaggregated_UnaggregatedAttestationsBySlotIndex(t *testing.T) {
	cache := NewAttCaches()

	att1 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b101}})
	att2 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2}, AggregationBits: bitfield.Bitlist{0b110}})
	att3 := util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1}, AggregationBits: bitfield.Bitlist{0b110}})
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		require.NoError(t, cache.SaveUnaggregatedAttestation(att))
	}
	ctx := context.Background()
	returned := cache.UnaggregatedAttestationsBySlotIndex(ctx, 1, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att1}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(ctx, 1, 2)
	assert.DeepEqual(t, []*ethpb.Attestation{att2}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(ctx, 2, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att3}, returned)
}

func TestKV_Unaggregated_UnaggregatedAttestationsBySlotIndexElectra(t *testing.T) {
	cache := NewAttCaches()

	committeeBits := primitives.NewAttestationCommitteeBits()
	committeeBits.SetBitAt(1, true)
	att1 := util.HydrateAttestationElectra(&ethpb.AttestationElectra{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b101}, CommitteeBits: committeeBits})
	committeeBits = primitives.NewAttestationCommitteeBits()
	committeeBits.SetBitAt(2, true)
	att2 := util.HydrateAttestationElectra(&ethpb.AttestationElectra{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b110}, CommitteeBits: committeeBits})
	committeeBits = primitives.NewAttestationCommitteeBits()
	committeeBits.SetBitAt(1, true)
	att3 := util.HydrateAttestationElectra(&ethpb.AttestationElectra{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110}, CommitteeBits: committeeBits})
	atts := []*ethpb.AttestationElectra{att1, att2, att3}

	for _, att := range atts {
		require.NoError(t, cache.SaveUnaggregatedAttestation(att))
	}
	ctx := context.Background()
	returned := cache.UnaggregatedAttestationsBySlotIndexElectra(ctx, 1, 1)
	assert.DeepEqual(t, []*ethpb.AttestationElectra{att1}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndexElectra(ctx, 1, 2)
	assert.DeepEqual(t, []*ethpb.AttestationElectra{att2}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndexElectra(ctx, 2, 1)
	assert.DeepEqual(t, []*ethpb.AttestationElectra{att3}, returned)
}
