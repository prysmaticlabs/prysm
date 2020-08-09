package kv

import (
	"testing"

	fssz "github.com/ferranbt/fastssz"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
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
			wantErrString: "could not tree hash attestation: " + fssz.ErrBytesLength.Error(),
		},
		{
			name:  "normal save",
			att:   &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b0001}, Data: &ethpb.AttestationData{BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}},
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			if len(cache.unAggregatedAtt) != 0 {
				t.Errorf("Invalid start pool, atts: %d", len(cache.unAggregatedAtt))
			}

			if tt.att != nil && tt.att.Signature == nil {
				tt.att.Signature = make([]byte, 96)
			}

			err := cache.SaveUnaggregatedAttestation(tt.att)
			if tt.wantErrString != "" && (err == nil || err.Error() != tt.wantErrString) {
				t.Errorf("Did not receive wanted error, want: %q, got: %v", tt.wantErrString, err)
				return
			}
			if tt.wantErrString == "" && err != nil {
				t.Error(err)
				return
			}
			if len(cache.unAggregatedAtt) != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, len(cache.unAggregatedAtt))
			}
			if cache.UnaggregatedAttestationCount() != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, cache.UnaggregatedAttestationCount())
			}
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
			if len(cache.unAggregatedAtt) != 0 {
				t.Errorf("Invalid start pool, atts: %d", len(cache.unAggregatedAtt))
			}

			err := cache.SaveUnaggregatedAttestations(tt.atts)
			if tt.wantErrString != "" && (err == nil || err.Error() != tt.wantErrString) {
				t.Errorf("Did not receive wanted error, want: %q, got: %v", tt.wantErrString, err)
			}
			if tt.wantErrString == "" && err != nil {
				t.Error(err)
			}
			if len(cache.unAggregatedAtt) != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, len(cache.unAggregatedAtt))
			}
			if cache.UnaggregatedAttestationCount() != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, cache.UnaggregatedAttestationCount())
			}
		})
	}
}

func TestKV_Unaggregated_DeleteUnaggregatedAttestation(t *testing.T) {
	t.Run("nil attestation", func(t *testing.T) {
		cache := NewAttCaches()
		if err := cache.DeleteUnaggregatedAttestation(nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("aggregated attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, Signature: make([]byte, 96)}
		err := cache.DeleteUnaggregatedAttestation(att)
		wantErr := "attestation is aggregated"
		if err == nil || err.Error() != wantErr {
			t.Errorf("Did not receive wanted error, want: %q, got: %v", wantErr, err)
		}
	})

	t.Run("successful deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b101}, Signature: make([]byte, 96)}
		att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
		att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b110}, Signature: make([]byte, 96)}
		atts := []*ethpb.Attestation{att1, att2, att3}
		if err := cache.SaveUnaggregatedAttestations(atts); err != nil {
			t.Fatal(err)
		}
		for _, att := range atts {
			if err := cache.DeleteUnaggregatedAttestation(att); err != nil {
				t.Error(err)
			}
		}
		returned := cache.UnaggregatedAttestations()
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
		if err := cache.SaveUnaggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.UnaggregatedAttestationsBySlotIndex(1, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att1}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(1, 2)
	assert.DeepEqual(t, []*ethpb.Attestation{att2}, returned)
	returned = cache.UnaggregatedAttestationsBySlotIndex(2, 1)
	assert.DeepEqual(t, []*ethpb.Attestation{att3}, returned)
}
