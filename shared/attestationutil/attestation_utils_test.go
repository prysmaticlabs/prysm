package attestationutil_test

import (
	"context"
	"strings"
	"testing"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestAttestingIndices(t *testing.T) {
	type args struct {
		bf        bitfield.Bitfield
		committee []uint64
	}
	tests := []struct {
		name string
		args args
		want []uint64
	}{
		{
			name: "Full committee attested",
			args: args{
				bf:        bitfield.Bitlist{0b1111},
				committee: []uint64{0, 1, 2},
			},
			want: []uint64{0, 1, 2},
		},
		{
			name: "Partial committee attested",
			args: args{
				bf:        bitfield.Bitlist{0b1101},
				committee: []uint64{0, 1, 2},
			},
			want: []uint64{0, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attestationutil.AttestingIndices(tt.args.bf, tt.args.committee)
			assert.DeepEqual(t, tt.want, got)
		})
	}
}

func TestIsValidAttestationIndices(t *testing.T) {
	tests := []struct {
		name string
		att  *eth.IndexedAttestation
		want string
	}{
		{
			name: "Indices should be non-empty",
			att: &eth.IndexedAttestation{
				AttestingIndices: []uint64{},
				Data: &eth.AttestationData{
					Target: &eth.Checkpoint{},
				},
				Signature: nil,
			},
			want: "expected non-empty",
		},
		{
			name: "Greater than max validators per committee",
			att: &eth.IndexedAttestation{
				AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
				Data: &eth.AttestationData{
					Target: &eth.Checkpoint{},
				},
				Signature: nil,
			},
			want: "indices count exceeds",
		},
		{
			name: "Needs to be sorted",
			att: &eth.IndexedAttestation{
				AttestingIndices: []uint64{3, 2, 1},
				Data: &eth.AttestationData{
					Target: &eth.Checkpoint{},
				},
				Signature: nil,
			},
			want: "not uniquely sorted",
		},
		{
			name: "Valid indices",
			att: &eth.IndexedAttestation{
				AttestingIndices: []uint64{1, 2, 3},
				Data: &eth.AttestationData{
					Target: &eth.Checkpoint{},
				},
				Signature: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := attestationutil.IsValidAttestationIndices(context.Background(), tt.att)
			if tt.want == "" && err != nil {
				t.Fatal(err)
			}
			if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
				t.Errorf("IsValidAttestationIndices() got = %v, want %v", err, tt.want)
			}
		})
	}
}

func BenchmarkAttestingIndices_PartialCommittee(b *testing.B) {
	bf := bitfield.Bitlist{0b11111111, 0b11111111, 0b10000111, 0b11111111, 0b100}
	committee := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = attestationutil.AttestingIndices(bf, committee)
	}
}

func TestAttDataIsEqual(t *testing.T) {
	type test struct {
		name     string
		attData1 *eth.AttestationData
		attData2 *eth.AttestationData
		equal    bool
	}
	tests := []test{
		{
			name: "same",
			attData1: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
			attData2: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
			equal: true,
		},
		{
			name: "diff slot",
			attData1: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
			attData2: &eth.AttestationData{
				Slot:            4,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
		},
		{
			name: "diff block",
			attData1: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("good block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
			attData2: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
		},
		{
			name: "diff source root",
			attData1: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("good source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
			attData2: &eth.AttestationData{
				Slot:            5,
				CommitteeIndex:  2,
				BeaconBlockRoot: []byte("great block"),
				Source: &eth.Checkpoint{
					Epoch: 4,
					Root:  []byte("bad source"),
				},
				Target: &eth.Checkpoint{
					Epoch: 10,
					Root:  []byte("good target"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, attestationutil.AttDataIsEqual(tt.attData1, tt.attData2))
		})
	}
}

func TestCheckPtIsEqual(t *testing.T) {
	type test struct {
		name     string
		checkPt1 *eth.Checkpoint
		checkPt2 *eth.Checkpoint
		equal    bool
	}
	tests := []test{
		{
			name: "same",
			checkPt1: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			checkPt2: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			equal: true,
		},
		{
			name: "diff epoch",
			checkPt1: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			checkPt2: &eth.Checkpoint{
				Epoch: 5,
				Root:  []byte("good source"),
			},
			equal: false,
		},
		{
			name: "diff root",
			checkPt1: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			checkPt2: &eth.Checkpoint{
				Epoch: 4,
				Root:  []byte("bad source"),
			},
			equal: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, attestationutil.CheckPointIsEqual(tt.checkPt1, tt.checkPt2))
		})
	}
}

func BenchmarkAttDataIsEqual(b *testing.B) {
	attData1 := &eth.AttestationData{
		Slot:            5,
		CommitteeIndex:  2,
		BeaconBlockRoot: []byte("great block"),
		Source: &eth.Checkpoint{
			Epoch: 4,
			Root:  []byte("good source"),
		},
		Target: &eth.Checkpoint{
			Epoch: 10,
			Root:  []byte("good target"),
		},
	}
	attData2 := &eth.AttestationData{
		Slot:            5,
		CommitteeIndex:  2,
		BeaconBlockRoot: []byte("great block"),
		Source: &eth.Checkpoint{
			Epoch: 4,
			Root:  []byte("good source"),
		},
		Target: &eth.Checkpoint{
			Epoch: 10,
			Root:  []byte("good target"),
		},
	}

	b.Run("fast", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			assert.Equal(b, true, attestationutil.AttDataIsEqual(attData1, attData2))
		}
	})

	b.Run("proto.Equal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			assert.Equal(b, true, attestationutil.AttDataIsEqual(attData1, attData2))
		}
	})
}
