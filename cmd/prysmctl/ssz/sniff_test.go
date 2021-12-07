package ssz

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestPathHead(t *testing.T) {
	p := path("a.b.c")
	head := p.head()
	tail := p.tail()

	require.Equal(t, path("a"), head)
	require.Equal(t, path("b.c"), tail)
}

func TestPathLeaf(t *testing.T) {
	p := path("a.b")
	l := path("a")
	require.Equal(t, false, p.leaf())
	require.Equal(t, true, l.leaf())
}

func fixtureField() *field {
	return &field{fieldType: TypeContainer, name: "BeaconState", nested: []*field{
		{size: 8, fieldType: TypeUint64, name: "genesis_time"},
		{size: 32, fieldType: TypeRoot, name: "genesis_validators_root"},
		{size: 8, fieldType: TypeUint64, name: "slot"},
		{fieldType: TypeContainer, name: "fork", nested: []*field{
			{size: 4, fieldType: TypeByteSlice, name: "previous_version"},
			{size: 4, fieldType: TypeByteSlice, name: "current_version"},
			{size: 8, fieldType: TypeUint64, name: "epoch"},
		}},
	}}
}

func TestFieldGet(t *testing.T) {
	testField := fixtureField()
	f, err := testField.get("BeaconState.fork.current_version")
	require.NoError(t, err)
	require.Equal(t, 4, f.size)
}

func TestFieldGetContainer(t *testing.T) {
	testField := fixtureField()
	f, err := testField.get("BeaconState.fork")
	require.NoError(t, err)
	require.Equal(t, 3, len(f.nested))
	require.Equal(t, "fork", f.name)
	epoch, err := f.get("fork.epoch")
	require.NoError(t, err)
	require.Equal(t, 8, epoch.size)
}
