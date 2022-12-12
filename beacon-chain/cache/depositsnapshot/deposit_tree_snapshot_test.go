package depositsnapshot

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDepositTreeSnapshot_CalculateRoot(t *testing.T) {
	tests := []struct {
		name         string
		finalized    int
		depositCount uint64
		want         [32]byte
	}{
		{
			name:         "empty",
			finalized:    0,
			depositCount: 0,
			want:         [32]byte{89, 154, 19, 135, 81, 239, 248, 61, 237, 225, 98, 110, 43, 130, 145, 21, 218, 236, 72, 223, 115, 71, 51, 92, 190, 80, 146, 24, 214, 95, 89, 175},
		},
		{
			name:         "1 finalized",
			finalized:    1,
			depositCount: 2,
			want:         [32]byte{27, 212, 239, 108, 14, 221, 178, 95, 149, 19, 106, 243, 39, 89, 184, 175, 189, 251, 230, 92, 200, 13, 19, 203, 102, 25, 58, 80, 160, 52, 48, 7},
		},
		{
			name:         "many finalised",
			finalized:    6,
			depositCount: 20,
			want:         [32]byte{94, 45, 50, 112, 88, 170, 119, 249, 185, 170, 83, 1, 45, 97, 27, 244, 170, 160, 27, 238, 110, 200, 60, 144, 193, 135, 74, 173, 253, 141, 24, 224},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var finalized [][32]byte
			for i := 0; i < tt.finalized; i++ {
				finalized = append(finalized, hexString(t, fmt.Sprintf("%064d", i)))
			}
			ds := &DepositTreeSnapshot{
				finalized:    finalized,
				depositCount: tt.depositCount,
			}
			if got := ds.CalculateRoot(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CalculateRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}
