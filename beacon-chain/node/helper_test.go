package node

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestConvertWspInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		bRoot   []byte
		epoch   uint64
		wantErr bool
		errStr  string
	}{
		{
			name:    "No column in string",
			input:   "0x111111;123",
			wantErr: true,
			errStr:  "did not contain column",
		},
		{
			name:    "Correct input #1",
			input:   "0x010203:987",
			bRoot:   []byte{1, 2, 3},
			epoch:   987,
			wantErr: false,
		},
		{
			name:    "Correct input #2",
			input:   "FFFFFFFFFFFFFFFFFF:123456789",
			bRoot:   []byte{255, 255, 255, 255, 255, 255, 255, 255, 255},
			epoch:   123456789,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bRoot, epoch, err := convertWspInput(tt.input)
			if (err != nil) != tt.wantErr {
				require.ErrorContains(t, tt.errStr, err)
				return
			}
			if !reflect.DeepEqual(bRoot, tt.bRoot) {
				t.Errorf("convertWspInput() block root = %v, want %v", bRoot, tt.bRoot)
			}
			if epoch != tt.epoch {
				t.Errorf("convertWspInput() epoch = %v, want %v", epoch, tt.epoch)
			}
		})
	}
}
