package state_native

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestCorrectness_FixedSize(t *testing.T) {
	m1_0 := bytesutil.ToBytes32([]byte("m1_0"))
	m2_0 := bytesutil.ToBytes32([]byte("m2_0"))
	m1_1 := bytesutil.ToBytes32([]byte("m1_1"))

	var bRootsArray [fieldparams.BlockRootsLength][32]byte
	bRootsSlice := make([][]byte, len(bRootsArray))
	for i := range bRootsArray {
		bRootsSlice[i] = bRootsArray[i][:]
	}
	var sRootsArray [fieldparams.StateRootsLength][32]byte
	sRootsSlice := make([][]byte, len(sRootsArray))
	for i := range sRootsArray {
		sRootsSlice[i] = sRootsArray[i][:]
	}
	var mixesArray [fieldparams.RandaoMixesLength][32]byte
	mixesArray[0] = m1_0
	mixesArray[1] = m1_1
	mixesSlice := make([][]byte, len(mixesArray))
	for i := range mixesArray {
		mixesSlice[i] = mixesArray[i][:]
	}
	st1, err := InitializeFromProtoUnsafePhase0(&eth.BeaconState{BlockRoots: bRootsSlice, StateRoots: sRootsSlice, RandaoMixes: mixesSlice, Balances: []uint64{}})
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
	m, err = st2.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m2_0[:], m)
	v = st1.RandaoMixes()
	assert.DeepEqual(t, m1_0[:], v[0])
	v = st2.RandaoMixes()
	assert.DeepEqual(t, m2_0[:], v[0])
	m, err = st1.RandaoMixAtIndex(1)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_1[:], m)
	m, err = st2.RandaoMixAtIndex(1)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_1[:], m)

	st3 := st2.Copy()
	m, err = st1.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_0[:], m)
	m, err = st2.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m2_0[:], m)
	m, err = st3.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m2_0[:], m)
	v = st1.RandaoMixes()
	assert.DeepEqual(t, m1_0[:], v[0])
	v = st2.RandaoMixes()
	assert.DeepEqual(t, m2_0[:], v[0])
	v = st3.RandaoMixes()
	assert.DeepEqual(t, m2_0[:], v[0])
	m, err = st1.RandaoMixAtIndex(1)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_1[:], m)
	m, err = st2.RandaoMixAtIndex(1)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_1[:], m)
	m, err = st3.RandaoMixAtIndex(1)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_1[:], m)

	require.NoError(t, st3.UpdateRandaoMixesAtIndex(0, m1_0))
	m, err = st1.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_0[:], m)
	m, err = st2.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m2_0[:], m)
	m, err = st3.RandaoMixAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, m1_0[:], m)
}

func TestCorrectness_VariableSize(t *testing.T) {
	var bRootsArray [fieldparams.BlockRootsLength][32]byte
	bRootsSlice := make([][]byte, len(bRootsArray))
	for i := range bRootsArray {
		bRootsSlice[i] = bRootsArray[i][:]
	}
	var sRootsArray [fieldparams.StateRootsLength][32]byte
	sRootsSlice := make([][]byte, len(sRootsArray))
	for i := range sRootsArray {
		sRootsSlice[i] = sRootsArray[i][:]
	}
	var mixesArray [fieldparams.RandaoMixesLength][32]byte
	mixesSlice := make([][]byte, len(mixesArray))
	for i := range mixesArray {
		mixesSlice[i] = mixesArray[i][:]
	}
	balances := []uint64{100, 101, 102}
	st1, err := InitializeFromProtoUnsafePhase0(&eth.BeaconState{BlockRoots: bRootsSlice, StateRoots: sRootsSlice, RandaoMixes: mixesSlice, Balances: balances})
	b, err := st1.BalanceAtIndex(0)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), b)
	assert.DeepEqual(t, []uint64{100, 101, 102}, st1.Balances())

	st2 := st1.Copy()
	require.NoError(t, st2.UpdateBalancesAtIndex(0, 200))
	b, err = st1.BalanceAtIndex(0)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), b)
	b, err = st2.BalanceAtIndex(0)
	require.NoError(t, err)
	assert.DeepEqual(t, uint64(200), b)
	b, err = st1.BalanceAtIndex(1)
	require.NoError(t, err)
	assert.Equal(t, uint64(101), b)
	b, err = st2.BalanceAtIndex(1)
	require.NoError(t, err)
	assert.Equal(t, uint64(101), b)
	assert.DeepEqual(t, []uint64{100, 101, 102}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102}, st2.Balances())
	assert.Equal(t, 3, st1.BalancesLength())
	assert.Equal(t, 3, st2.BalancesLength())

	// Test cases:
	// - append to st1
	// - append to st2
	// - append to st1
	// - update st1[3]
	// - update st1[4]
	// - update st2[4] (error)
	// - copy st1 to st3
	// - append to st3

	require.NoError(t, st1.AppendBalance(103))
	b, err = st1.BalanceAtIndex(3)
	require.NoError(t, err)
	assert.Equal(t, uint64(103), b)
	_, err = st2.BalanceAtIndex(3)
	assert.ErrorContains(t, "out of bounds", err)
	assert.DeepEqual(t, []uint64{100, 101, 102, 103}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102}, st2.Balances())
	assert.Equal(t, 4, st1.BalancesLength())
	assert.Equal(t, 3, st2.BalancesLength())

	require.NoError(t, st2.AppendBalance(203))
	b, err = st1.BalanceAtIndex(3)
	require.NoError(t, err)
	assert.Equal(t, uint64(103), b)
	b, err = st2.BalanceAtIndex(3)
	require.NoError(t, err)
	assert.Equal(t, uint64(203), b)
	assert.DeepEqual(t, []uint64{100, 101, 102, 103}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102, 203}, st2.Balances())
	assert.Equal(t, 4, st1.BalancesLength())
	assert.Equal(t, 4, st2.BalancesLength())

	require.NoError(t, st1.AppendBalance(104))
	b, err = st1.BalanceAtIndex(4)
	require.NoError(t, err)
	assert.Equal(t, uint64(104), b)
	_, err = st2.BalanceAtIndex(4)
	assert.ErrorContains(t, "out of bounds", err)
	assert.DeepEqual(t, []uint64{100, 101, 102, 103, 104}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102, 203}, st2.Balances())
	assert.Equal(t, 5, st1.BalancesLength())
	assert.Equal(t, 4, st2.BalancesLength())

	require.NoError(t, st1.UpdateBalancesAtIndex(3, 1103))
	b, err = st1.BalanceAtIndex(3)
	require.NoError(t, err)
	assert.Equal(t, uint64(1103), b)
	b, err = st2.BalanceAtIndex(3)
	require.NoError(t, err)
	assert.Equal(t, uint64(203), b)
	assert.DeepEqual(t, []uint64{100, 101, 102, 1103, 104}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102, 203}, st2.Balances())
	assert.Equal(t, 5, st1.BalancesLength())
	assert.Equal(t, 4, st2.BalancesLength())

	require.NoError(t, st1.UpdateBalancesAtIndex(4, 1104))
	b, err = st1.BalanceAtIndex(4)
	require.NoError(t, err)
	assert.Equal(t, uint64(1104), b)
	_, err = st2.BalanceAtIndex(4)
	assert.ErrorContains(t, "out of bounds", err)
	assert.DeepEqual(t, []uint64{100, 101, 102, 1103, 1104}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102, 203}, st2.Balances())
	assert.Equal(t, 5, st1.BalancesLength())
	assert.Equal(t, 4, st2.BalancesLength())

	assert.ErrorContains(t, "out of bounds", st2.UpdateBalancesAtIndex(4, 9999))

	st3 := st1.Copy()
	assert.DeepEqual(t, st1.Balances(), st3.Balances())

	require.NoError(t, st3.AppendBalance(305))
	_, err = st1.BalanceAtIndex(5)
	assert.ErrorContains(t, "out of bounds", err)
	_, err = st2.BalanceAtIndex(5)
	assert.ErrorContains(t, "out of bounds", err)
	b, err = st3.BalanceAtIndex(5)
	assert.Equal(t, uint64(305), b)
	assert.DeepEqual(t, []uint64{100, 101, 102, 1103, 1104}, st1.Balances())
	assert.DeepEqual(t, []uint64{200, 101, 102, 203}, st2.Balances())
	assert.DeepEqual(t, []uint64{100, 101, 102, 1103, 1104, 305}, st3.Balances())
	assert.Equal(t, 5, st1.BalancesLength())
	assert.Equal(t, 4, st2.BalancesLength())
	assert.Equal(t, 6, st3.BalancesLength())
}
