package stateutils_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/stateutils"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state-native/v1"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
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
		key [fieldparams.BLSPubkeyLength]byte
		val types.ValidatorIndex
		ok  bool
	}{
		{
			key: bytesutil.ToBytes48([]byte("zero")),
			val: 0,
			ok:  true,
		}, {
			key: bytesutil.ToBytes48([]byte("one")),
			val: 1,
			ok:  true,
		}, {
			key: bytesutil.ToBytes48([]byte("no")),
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
