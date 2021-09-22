package stateutils_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/stateutils"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	butil "github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestValidatorIndexMap_OK(t *testing.T) {
	base := &ethpb.BeaconState{
		Validators: []*ethpb.Validator{
			{
				PublicKey: []byte("zero"),
			},
			{
				PublicKey: []byte("one"),
			},
		},
	}
	state, err := v1.InitializeFromProto(base)
	require.NoError(t, err)

	tests := []struct {
		key [48]byte
		val types.ValidatorIndex
		ok  bool
	}{
		{
			key: butil.ToBytes48([]byte("zero")),
			val: 0,
			ok:  true,
		}, {
			key: butil.ToBytes48([]byte("one")),
			val: 1,
			ok:  true,
		}, {
			key: butil.ToBytes48([]byte("no")),
			val: 0,
			ok:  false,
		},
	}

	m := stateutils.ValidatorIndexMap(state.Validators())
	for _, tt := range tests {
		result, ok := m[tt.key]
		assert.Equal(t, tt.val, result)
		assert.Equal(t, tt.ok, ok)
	}
}
