package backend

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/sszgen/types"
)

// cases left to satisfy:
// list-vector-byte
func TestGenerateHashTreeRoot(t *testing.T) {
	b, err := os.ReadFile("testdata/TestGenerateHashTreeRoot.expected")
	require.NoError(t, err)
	expected := string(b)

	vc, ok := testFixBeaconState.(*types.ValueContainer)
	require.Equal(t, true, ok)
	gc := &generateContainer{vc, ""}
	code := GenerateHashTreeRoot(gc)
	require.Equal(t, 4, len(code.imports))
	actual, err := normalizeFixtureString(code.blocks[0])
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestHTROverlayCoerce(t *testing.T) {
	pkg := "derp"
	expected := "hh.PutUint64(uint64(b.Slot))"
	val :=  &types.ValueOverlay{
		Name:       "",
		Package:    pkg,
		Underlying: &types.ValueUint{
			Name:    "uint64",
			Size:    64,
			Package: pkg,
		},
	}
	gv := &generateOverlay{val, pkg}
	actual := gv.generateHTRPutter("b.Slot")
	require.Equal(t, expected, actual)
}

func TestHTRContainer(t *testing.T) {
	pkg := "derp"
	expected := `if err := b.Fork.HashTreeRootWith(hh); err != nil {
		return err
	}`
	val := &types.ValueContainer{}
	gv := &generateContainer{val, pkg}
	actual := gv.generateHTRPutter("b.Fork")
	require.Equal(t, expected, actual)
}

func TestHTRByteVector(t *testing.T) {
	pkg := "derp"
	fieldName := "c.GenesisValidatorsRoot"
	expected := `{
	if len(c.GenesisValidatorsRoot) != 32 {
		return ssz.ErrVectorLength
	}
	hh.PutBytes(c.GenesisValidatorsRoot)
}`
	val := &types.ValueVector{
		ElementValue: &types.ValueByte{},
		Size:         32,
	}
	gv := &generateVector{
		valRep:        val,
		targetPackage: pkg,
	}
	actual := gv.generateHTRPutter(fieldName)
	require.Equal(t, expected, actual)
}