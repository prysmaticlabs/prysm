package backend

import (
	"os"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

func TestGenerateUnmarshalSSZ(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateUnmarshalSSZ.expected")
	require.NoError(t, err)
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{vc, ""}
	code := GenerateUnmarshalSSZ(gc)
	require.Equal(t, 4, len(code.imports))
	actual, err := normalizeFixtureString(code.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestUnmarshalSteps(t *testing.T) {
	fixturePath := "testdata/TestUnmarshalSteps.expected"
	b, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	expected, err := normalizeFixtureBytes(b)
	require.NoError(t, err)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{vc, "" }
	ums := gc.unmarshalSteps()
	require.Equal(t, 21, len(ums))
	require.Equal(t, ums[15].nextVariable.fieldNumber, ums[16].fieldNumber)

	gotRaw := strings.Join([]string{ums.fixedSlices(), "", ums.variableSlices(gc.fixedOffset())}, "\n")
	actual, err := normalizeFixtureString(gotRaw)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

