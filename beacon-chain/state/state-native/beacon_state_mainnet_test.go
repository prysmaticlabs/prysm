//go:build !minimal

package state_native

import (
	"reflect"
	"strconv"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestMainnetSszValuesAgainstFieldParams(t *testing.T) {
	// Casting needed to avoid lock copy analyzer issue.
	bs := (interface{})(BeaconState{})
	bsType := reflect.TypeOf(bs)

	f, ok := bsType.FieldByName("genesisValidatorsRoot")
	require.Equal(t, true, ok, "Required field not found")
	v := f.Tag.Get("ssz-size")
	assert.Equal(t, strconv.Itoa(fieldparams.RootLength), v)

	f, ok = bsType.FieldByName("blockRoots")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, strconv.Itoa(fieldparams.BlockRootsLength)+","+strconv.Itoa(fieldparams.RootLength), v)

	f, ok = bsType.FieldByName("stateRoots")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, strconv.Itoa(fieldparams.StateRootsLength)+","+strconv.Itoa(fieldparams.RootLength), v)

	f, ok = bsType.FieldByName("historicalRoots")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, "?,"+strconv.Itoa(fieldparams.RootLength), v)
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.HistoricalRootsLength), v)

	f, ok = bsType.FieldByName("eth1DataVotes")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.Eth1DataVotesLength), v)

	f, ok = bsType.FieldByName("validators")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.ValidatorRegistryLimit), v)

	f, ok = bsType.FieldByName("balances")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.ValidatorRegistryLimit), v)

	f, ok = bsType.FieldByName("randaoMixes")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, strconv.Itoa(fieldparams.RandaoMixesLength)+","+strconv.Itoa(fieldparams.RootLength), v)

	f, ok = bsType.FieldByName("slashings")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, strconv.Itoa(fieldparams.SlashingsLength), v)

	f, ok = bsType.FieldByName("previousEpochAttestations")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.PreviousEpochAttestationsLength), v)

	f, ok = bsType.FieldByName("currentEpochAttestations")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.CurrentEpochAttestationsLength), v)

	f, ok = bsType.FieldByName("justificationBits")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-size")
	assert.Equal(t, "1", v)

	f, ok = bsType.FieldByName("inactivityScores")
	require.Equal(t, true, ok, "Required field not found")
	v = f.Tag.Get("ssz-max")
	assert.Equal(t, strconv.Itoa(fieldparams.ValidatorRegistryLimit), v)
}
