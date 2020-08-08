package kv

import (
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestKV_Aggregated_AggregateUnaggregatedAttestations(t *testing.T) {
	cache := NewAttCaches()
	priv := bls.RandKey()
	sig1 := priv.Sign([]byte{'a'})
	sig2 := priv.Sign([]byte{'b'})
	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig1.Marshal()}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1010}, Signature: sig1.Marshal()}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig1.Marshal()}
	att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig2.Marshal()}
	att5 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig1.Marshal()}
	att6 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1010}, Signature: sig1.Marshal()}
	att7 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1100}, Signature: sig1.Marshal()}
	att8 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1001}, Signature: sig2.Marshal()}
	atts := []*ethpb.Attestation{att1, att2, att3, att4, att5, att6, att7, att8}
	if err := cache.SaveUnaggregatedAttestations(atts); err != nil {
		t.Fatal(err)
	}
	if err := cache.AggregateUnaggregatedAttestations(); err != nil {
		t.Fatal(err)
	}

	if len(cache.AggregatedAttestationsBySlotIndex(1, 0)) != 1 {
		t.Fatal("Did not aggregate correctly")
	}
	if len(cache.AggregatedAttestationsBySlotIndex(2, 0)) != 1 {
		t.Fatal("Did not aggregate correctly")
	}
}

func TestKV_Aggregated_SaveAggregatedAttestation(t *testing.T) {
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
			name: "nil attestation data",
			att:  &ethpb.Attestation{},
		},
		{
			name: "not aggregated",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: make([]byte, 32),
					Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
					Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				}, AggregationBits: bitfield.Bitlist{0b10100}},
			wantErrString: "attestation is not aggregated",
		},
		{
			name: "invalid hash",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: []byte{0b0},
				},
				AggregationBits: bitfield.Bitlist{0b10111},
			},
			wantErrString: "could not tree hash attestation: incorrect fixed bytes marshalling",
		},
		{
			name: "normal save",
			att: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1101},
			},
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			if len(cache.unAggregatedAtt) != 0 {
				t.Errorf("Invalid start pool, atts: %d", len(cache.unAggregatedAtt))
			}

			err := cache.SaveAggregatedAttestation(tt.att)
			if tt.wantErrString != "" && (err == nil || err.Error() != tt.wantErrString) {
				t.Errorf("Did not receive wanted error, want: %q, got: %v", tt.wantErrString, err)
				return
			}
			if tt.wantErrString == "" && err != nil {
				t.Error(err)
				return
			}
			if len(cache.aggregatedAtt) != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, len(cache.aggregatedAtt))
			}
			if cache.AggregatedAttestationCount() != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, cache.AggregatedAttestationCount())
			}
		})
	}
}

func TestKV_Aggregated_SaveAggregatedAttestations(t *testing.T) {
	tests := []struct {
		name          string
		atts          []*ethpb.Attestation
		count         int
		wantErrString string
	}{
		{
			name: "no duplicates",
			atts: []*ethpb.Attestation{
				{Data: &ethpb.AttestationData{Slot: 1},
					AggregationBits: bitfield.Bitlist{0b1101}},
				{Data: &ethpb.AttestationData{Slot: 1},
					AggregationBits: bitfield.Bitlist{0b1101}},
			},
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			if len(cache.aggregatedAtt) != 0 {
				t.Errorf("Invalid start pool, atts: %d", len(cache.unAggregatedAtt))
			}
			err := cache.SaveAggregatedAttestations(tt.atts)
			if tt.wantErrString == "" && err != nil {
				t.Error(err)
				return
			}
			if len(cache.aggregatedAtt) != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, len(cache.aggregatedAtt))
			}
			if cache.AggregatedAttestationCount() != tt.count {
				t.Errorf("Wrong attestation count, want: %d, got: %d", tt.count, cache.AggregatedAttestationCount())
			}
		})
	}
}

func TestKV_Aggregated_AggregatedAttestations(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
	att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
	atts := []*ethpb.Attestation{att1, att2, att3}

	for _, att := range atts {
		if err := cache.SaveAggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.AggregatedAttestations()
	sort.Slice(returned, func(i, j int) bool {
		return returned[i].Data.Slot < returned[j].Data.Slot
	})
	assert.DeepEqual(t, atts, returned)
}

func TestKV_Aggregated_DeleteAggregatedAttestation(t *testing.T) {
	t.Run("nil attestation", func(t *testing.T) {
		cache := NewAttCaches()
		if err := cache.DeleteAggregatedAttestation(nil); err != nil {
			t.Error(err)
		}
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b10101}}
		if err := cache.DeleteAggregatedAttestation(att); err != nil {
			t.Error(err)
		}
	})

	t.Run("non aggregated attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}, Data: &ethpb.AttestationData{Slot: 2}}
		err := cache.DeleteAggregatedAttestation(att)
		wantErr := "attestation is not aggregated"
		if err == nil || err.Error() != wantErr {
			t.Errorf("Did not receive wanted error, want: %q, got: %v", wantErr, err)
		}
	})

	t.Run("invalid hash", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{
			AggregationBits: bitfield.Bitlist{0b1111},
			Data: &ethpb.AttestationData{
				Slot:            2,
				BeaconBlockRoot: []byte{0b0},
			},
		}
		err := cache.DeleteAggregatedAttestation(att)
		wantErr := "could not tree hash attestation data: incorrect fixed bytes marshalling"
		if err == nil || err.Error() != wantErr {
			t.Errorf("Did not receive wanted error, want: %q, got: %v", wantErr, err)
		}
	})

	t.Run("nonexistent attestation", func(t *testing.T) {
		cache := NewAttCaches()
		att := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1111}, Data: &ethpb.AttestationData{Slot: 2}}
		if err := cache.DeleteAggregatedAttestation(att); err != nil {
			t.Error(err)
		}
	})

	t.Run("non-filtered deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
		att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b1101}}
		att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b1101}}
		att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 3}, AggregationBits: bitfield.Bitlist{0b10101}}
		atts := []*ethpb.Attestation{att1, att2, att3, att4}
		if err := cache.SaveAggregatedAttestations(atts); err != nil {
			t.Fatal(err)
		}

		if err := cache.DeleteAggregatedAttestation(att1); err != nil {
			t.Fatal(err)
		}
		if err := cache.DeleteAggregatedAttestation(att3); err != nil {
			t.Fatal(err)
		}

		returned := cache.AggregatedAttestations()
		wanted := []*ethpb.Attestation{att2}
		assert.DeepEqual(t, wanted, returned)
	})

	t.Run("filtered deletion", func(t *testing.T) {
		cache := NewAttCaches()
		att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b110101}}
		att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110111}}
		att3 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110100}}
		att4 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 2}, AggregationBits: bitfield.Bitlist{0b110101}}
		atts := []*ethpb.Attestation{att1, att2, att3, att4}
		if err := cache.SaveAggregatedAttestations(atts); err != nil {
			t.Fatal(err)
		}

		if cache.AggregatedAttestationCount() != 2 {
			t.Error("Unexpected number of atts")
		}
		if err := cache.DeleteAggregatedAttestation(att4); err != nil {
			t.Fatal(err)
		}

		returned := cache.AggregatedAttestations()
		wanted := []*ethpb.Attestation{att1, att2}
		sort.Slice(returned, func(i, j int) bool {
			return string(returned[i].AggregationBits) < string(returned[j].AggregationBits)
		})
		assert.DeepEqual(t, wanted, returned)
	})
}

func TestKV_Aggregated_HasAggregatedAttestation(t *testing.T) {
	tests := []struct {
		name     string
		existing []*ethpb.Attestation
		input    *ethpb.Attestation
		want     bool
	}{
		{
			name:  "nil attestation",
			input: nil,
			want:  false,
		},
		{
			name: "nil attestation data",
			input: &ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: false,
		},
		{
			name: "empty cache aggregated",
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: false,
		},
		{
			name: "empty cache unaggregated",
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1001}},
			want: false,
		},
		{
			name: "single attestation in cache with exact match",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: true,
		},
		{
			name: "single attestation in cache with subset aggregation",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1110}},
			want: true,
		},
		{
			name: "single attestation in cache with superset aggregation",
			existing: []*ethpb.Attestation{{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1110}},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111}},
			want: false,
		},
		{
			name: "multiple attestations with same data in cache with overlapping aggregation, input is subset",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1100000}},
			want: true,
		},
		{
			name: "multiple attestations with same data in cache with overlapping aggregation and input is superset",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 1,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111111}},
			want: false,
		},
		{
			name: "multiple attestations with different data in cache",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 2,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
				{
					Data: &ethpb.AttestationData{
						Slot: 3,
					},
					AggregationBits: bitfield.Bitlist{0b1100111},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 1,
				},
				AggregationBits: bitfield.Bitlist{0b1111111}},
			want: false,
		},
		{
			name: "attestations with different bitlist lengths",
			existing: []*ethpb.Attestation{
				{
					Data: &ethpb.AttestationData{
						Slot: 2,
					},
					AggregationBits: bitfield.Bitlist{0b1111000},
				},
			},
			input: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Slot: 2,
				},
				AggregationBits: bitfield.Bitlist{0b1111},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewAttCaches()
			if err := cache.SaveAggregatedAttestations(tt.existing); err != nil {
				t.Error(err)
			}

			result, err := cache.HasAggregatedAttestation(tt.input)
			require.NoError(t, err)
			if result != tt.want {
				t.Errorf("Result = %v, wanted = %v", result, tt.want)
			}

			// Same test for block attestations
			cache = NewAttCaches()
			if err := cache.SaveBlockAttestations(tt.existing); err != nil {
				t.Error(err)
			}

			result, err = cache.HasAggregatedAttestation(tt.input)
			require.NoError(t, err)
			if result != tt.want {
				t.Errorf("Result = %v, wanted = %v", result, tt.want)
			}
		})
	}
}

func TestKV_Aggregated_DuplicateAggregatedAttestations(t *testing.T) {
	cache := NewAttCaches()

	att1 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1101}}
	att2 := &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 1}, AggregationBits: bitfield.Bitlist{0b1111}}
	atts := []*ethpb.Attestation{att1, att2}

	for _, att := range atts {
		if err := cache.SaveAggregatedAttestation(att); err != nil {
			t.Fatal(err)
		}
	}

	returned := cache.AggregatedAttestations()

	// It should have only returned att2.
	if !reflect.DeepEqual(att2, returned[0]) || len(returned) != 1 {
		t.Error("Did not receive correct aggregated atts")
	}
}
