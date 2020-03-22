package attestationutil_test

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AttestingIndices() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkAttestingIndices_PartialCommittee(b *testing.B) {
	bf := bitfield.Bitlist{0b11111111, 0b11111111, 0b11111111, 0b11111111, 0b100}
	committee := []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34}

	for i := 0; i < b.N; i++ {
		_ = attestationutil.AttestingIndices(bf, committee)
	}
}
