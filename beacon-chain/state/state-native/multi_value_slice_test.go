package state_native

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestCorrectness(t *testing.T) {
	var mixesArray [fieldparams.RandaoMixesLength][32]byte
	m1_0 := bytesutil.ToBytes32([]byte("m1_0"))
	m2_0 := bytesutil.ToBytes32([]byte("m2_0"))
	mixesArray[0] = m1_0
	mixesSlice := make([][]byte, len(mixesArray))
	for i := range mixesArray {
		mixesSlice[i] = mixesArray[i][:]
	}
	st1, err := InitializeFromProtoUnsafePhase0(&eth.BeaconState{RandaoMixes: mixesSlice})
	m, err := st1.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_0[:], m)
	v := st1.RandaoMixes()
	assert.DeepEqual(t, m1_0[:], v[0])

	st2 := st1.Copy()
	require.NoError(t, st2.UpdateRandaoMixesAtIndex(0, m2_0))
	m, err = st1.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_0[:], m)
	v = st1.RandaoMixes()
	assert.DeepEqual(t, m1_0[:], v[0])
	m, err = st2.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m2_0[:], m)
	v = st2.RandaoMixes()
	assert.DeepEqual(t, m2_0[:], v[0])
}
