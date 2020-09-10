package kv

import (
	"bytes"
	"sort"
	"testing"

	fssz "github.com/ferranbt/fastssz"
	c "github.com/patrickmn/go-cache"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestKV_Unaggregated_SaveUnaggregatedAttestation(t *testing.T) {
	tests := []struct {
		name          string
		att           *ethpb.Attestation
		count         int
		wantErrString string
	}{
		{
			name: "nil attestation",
			att:  nil,
		},
		{
			name:          "already aggregated",
			att:           &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10101}, Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}},
			wantErrString: "attestation is aggregated",
		},
		{
			name: "invalid hash",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: []byte{0b0},
				},
			},
			wantErrString: fssz.ErrBytesLength.Error(),
		},
		{
			name:  "normal save",
			att:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b0001}, Data: &ethpb.AttestationData{BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}},
			count: 1,
		},
		{
			name: "already seen",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot:            100,
					BeaconBlockRoot: make([]byte, 32),
					Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				},
				AggregationBits: bitfield.Bitlist{0b10000001},
				Signature:       make([]byte, 96),
			},
			count: 0,
		},
	}
	r, err := hashFn(&ethpb.AttestationData{
		Slot:            100,
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		BeaconBlockRoot: make([]byte, 32),
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			cache.seenAtt.Set(string(r[:]), []bitfield.Bitlist{{0xff}}, c.DefaultExpiration)
			assert.Equal(t, 0, len(cache.unAggregatedAtt), "Invalid start pool, atts: %d", len(cache.unAggregatedAtt))

			if tt.att != nil && tt.att.Signature == nil {
				tt.att.Signature = make([]byte, 96)
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
		atts          []*ethpb.Attestation
		count         int
		wantErrString string
	}{
		{
			name: "unaggregated only",
			atts: []*ethpb.Attestation{
				{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
				{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
				{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
			},
			count: 3,
		},
		{
			name: "has aggregated",
			atts: []*ethpb.Attestation{
				{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
				{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
				{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)},
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
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)}
		err := cache.DeleteUnaggregatedAttestation(att)
		assert.ErrorContains(t, "attestation is aggregated", err)
	})

	t.Run("successful deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b101}, Signature: make([]byte, 96)}
		att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
		att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
		atts := []*ethpb.Attestation{att1, att2, att3}
		require.NoError(t, cache.SaveUnaggregatedAttestations(atts))
		for _, att := range atts {
			assert.NoError(t, cache.DeleteUnaggregatedAttestation(att))
		}
		returned, err := cache.UnaggregatedAttestations()
		require.NoError(t, err)
		assert.DeepEqual(t, []*ethpb.Attestation{}, returned)
	})
}

func TestKV_Unaggregated_DeleteSeenUnaggregatedAttestations(t *testing.T) {
	d := &ethpb.AttestationData{
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		BeaconBlockRoot: make([]byte, 32),
	}

	t.Run("no attestations", func(t *testing.T) {
		cache := NewAttCaches()
		count, err := cache.DeleteSeenUnaggregatedAttestations()
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("none seen", func(t *testing.T) {
		cache := NewAttCaches()
		atts := []*ethpb.Attestation{
			{Data: d, AggregationBits: bitfield.Bitlist{0b1001}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1010}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1100}, Signature: make([]byte, 96)},
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
		atts := []*ethpb.Attestation{
			{Data: d, AggregationBits: bitfield.Bitlist{0b1001}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1010}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1100}, Signature: make([]byte, 96)},
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
			return bytes.Compare(returned[i].AggregationBits, returned[j].AggregationBits) < 0
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []*ethpb.Attestation{atts[0], atts[2]}, returned)
	})

	t.Run("all seen", func(t *testing.T) {
		cache := NewAttCaches()
		atts := []*ethpb.Attestation{
			{Data: d, AggregationBits: bitfield.Bitlist{0b1001}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1010}, Signature: make([]byte, 96)},
			{Data: d, AggregationBits: bitfield.Bitlist{0b1100}, Signature: make([]byte, 96)},
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
		assert.DeepEqual(t, []*ethpb.Attestation{}, returned)
	})
}

func TestKV_Unaggregated_UnaggregatedAttestationsBySlotIndex(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b101}, Signature: make([]byte, 96)}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, CommitteeIndex: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, CommitteeIndex: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		require.NoError(t, cache.SaveUnaggregatedAttestation(att))
	}

	returned := cache.UnaggregatedAttestationsBySlotIndex(1, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att1}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(1, 2)
	assert.DeepEqual(t, []*ethpb.Attestation{att2}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(2, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att3}, returned)
}
