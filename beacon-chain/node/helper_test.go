package node

import (
	"reflect"
	"testing"

	"github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestConvertWspInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		bRoot   []byte
		epoch   types.Epoch
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
			name:    "Too many columns in string",
			input:   "0x010203:123:456",
			wantErr: false,
			errStr:  "weak subjectivity checkpoint input should be in `block_root:epoch_number` format",
		},
		{
			name:    "Incorrect block root length",
			input:   "0x010203:987",
			wantErr: false,
			errStr:  "block root is not length of 32",
		},
		{
			name:    "Correct input",
			input:   "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			bRoot:   []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			epoch:   123456789,
			wantErr: false,
		},
		{
			name:    "Correct input without 0x",
			input:   "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF:123456789",
			bRoot:   []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
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
