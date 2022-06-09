package backend

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

func TestGenerateSizeSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateSizeSSZ.expected")
	require.NoError(t, err)
	expected := string(b)

	ty, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := GenerateSizeSSZ(&generateContainer{ty, ""})
	require.Equal(t, 4, len(gc.imports))
	actual, err := normalizeFixtureString(gc.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

