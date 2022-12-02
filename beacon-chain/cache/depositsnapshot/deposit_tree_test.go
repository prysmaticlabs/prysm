package depositsnapshot

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/status-im/keycard-go/hexutils"
)

func TestEmptyRoot(t *testing.T) {
	empty := NewDepositTree()
	root := empty.GetRoot()
	assert.Equal(t, hexutils.BytesToHex(root[:]), "d70a234731285c6804c2a4f56711ddb8c82c99740f207854891028af34e27e5e")
	//assert.Equal(t, empty.GetRoot(), hexString(t, "d70a234731285c6804c2a4f56711ddb8c82c99740f207854891028af34e27e5e"))
}

func TestDepositTree_AddDeposit(t *testing.T) {
	tests := []struct {
		name    string
		num     int
		expRoot [32]byte
	}{
		{
			name:    "1 deposit",
			num:     1,
			expRoot: [32]byte{245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169, 39, 89, 251, 75},
		},
		{
			name:    "5 deposits",
			num:     5,
			expRoot: [32]byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDepositTree()
			for i := 0; i < tt.num; i++ {
				deposit := hexString(t, fmt.Sprintf("%064d", i))
				err := d.AddDeposit(deposit, uint64(i))
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expRoot, d.tree.GetRoot())
		})
	}
}
