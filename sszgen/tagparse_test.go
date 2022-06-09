package sszgen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestTokens(t *testing.T) {
	testTag := "`protobuf:\"bytes,2004,rep,name=historical_roots,json=historicalRoots,proto3\" json:\"historical_roots,omitempty\" ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	tp := &TagParser{}
	tp.Init(testTag)
	tags := tp.GetSSZTags()
	sszSize, ok := tags["ssz-size"]
	require.Equal(t, true, ok)
	require.Equal(t, "?,32", sszSize)
	sszMax, ok := tags["ssz-max"]
	require.Equal(t, true, ok)
	require.Equal(t, "16777216", sszMax)
}

func TestFullTag(t *testing.T) {
	tag := "`protobuf:\"bytes,1002,opt,name=genesis_validators_root,json=genesisValidatorsRoot,proto3\" json:\"genesis_validators_root,omitempty\" ssz-size:\"32\"`"
	_, err := extractSSZDimensions(tag)
	require.NoError(t, err)
}

func TestListOfVector(t *testing.T) {
	tag := "`protobuf:\"bytes,2004,rep,name=historical_roots,json=historicalRoots,proto3\" json:\"historical_roots,omitempty\" ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	_, err := extractSSZDimensions(tag)
	require.NoError(t, err)
}

func TestWildcardSSZSize(t *testing.T)  {
	tag := "`ssz-max:\"16777216\" ssz-size:\"?,32\"`"
	bounds, err := extractSSZDimensions(tag)
	require.NoError(t, err)
	require.Equal(t, 2, len(bounds))
	require.Equal(t, true, bounds[0].IsList())
	require.Equal(t, false, bounds[0].IsVector())
	require.Equal(t, 16777216, bounds[0].ListLen())
	require.Equal(t, false, bounds[1].IsList())
	require.Equal(t, true, bounds[1].IsVector())
	require.Equal(t, 32, bounds[1].VectorLen())
}
