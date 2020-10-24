package protoarray

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestStore_NodesIndices(t *testing.T) {
	type fields struct {
		nodesIndices       map[[32]byte]uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]uint64
	}{
		{
			name: "converts 32 byte key to hex string",
			fields: fields{
				nodesIndices: map[[32]byte]uint64{
					bytesutil.ToBytes32([]byte{0x01}): 1,
					bytesutil.ToBytes32([]byte{0x02}): 2,
				},
			},
			want: map[string]uint64{
				"0x0100000000000000000000000000000000000000000000000000000000000000": 1,
				"0x0200000000000000000000000000000000000000000000000000000000000000": 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				nodesIndices:       tt.fields.nodesIndices,
			}
			if got := s.NodesIndices(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NodesIndices() = %v, want %v", got, tt.want)
			}
		})
	}
}