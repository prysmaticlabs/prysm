package validator

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/stretchr/testify/require"
)

func TestConstructGenericBeaconBlock(t *testing.T) {
	vs := &Server{}

	// Test when sBlk or sBlk.Block() is nil
	t.Run("NilBlock", func(t *testing.T) {
		_, err := vs.constructGenericBeaconBlock(nil, nil, nil)
		require.ErrorContains(t, err, "block cannot be nil")
	})

	// Test for Deneb version
	t.Run("deneb block", func(t *testing.T) {
		eb := util.NewBeaconBlockDeneb()
		b, err := blocks.NewSignedBeaconBlock(eb)
		require.NoError(t, err)
		r1, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		scs := []*ethpb.DeprecatedBlobSidecar{
			util.GenerateTestDenebBlobSidecar(r1, eb, 0, []byte{}),
			util.GenerateTestDenebBlobSidecar(r1, eb, 1, []byte{}),
			util.GenerateTestDenebBlobSidecar(r1, eb, 2, []byte{}),
			util.GenerateTestDenebBlobSidecar(r1, eb, 3, []byte{}),
			util.GenerateTestDenebBlobSidecar(r1, eb, 4, []byte{}),
			util.GenerateTestDenebBlobSidecar(r1, eb, 5, []byte{}),
		}
		result, err := vs.constructGenericBeaconBlock(b, nil, scs)
		require.NoError(t, err)
		r2, err := result.GetDeneb().Block.HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, len(result.GetDeneb().Blobs), len(scs))
		require.Equal(t, result.IsBlinded, false)
	})

	// Test for blind Deneb version
	t.Run("blind deneb block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockDeneb())
		require.NoError(t, err)
		r1, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		scs := []*ethpb.BlindedBlobSidecar{{}, {}, {}, {}, {}, {}}
		result, err := vs.constructGenericBeaconBlock(b, scs, nil)
		require.NoError(t, err)
		r2, err := result.GetBlindedDeneb().Block.HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, len(result.GetBlindedDeneb().Blobs), len(scs))
		require.Equal(t, result.IsBlinded, true)
	})

	// Test for Capella version
	t.Run("capella block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		result, err := vs.constructGenericBeaconBlock(b, nil, nil)
		require.NoError(t, err)
		r1, err := result.GetCapella().HashTreeRoot()
		require.NoError(t, err)
		r2, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, result.IsBlinded, false)
	})

	// Test for blind Capella version
	t.Run("blind capella block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockCapella())
		require.NoError(t, err)
		result, err := vs.constructGenericBeaconBlock(b, nil, nil)
		require.NoError(t, err)
		r1, err := result.GetBlindedCapella().HashTreeRoot()
		require.NoError(t, err)
		r2, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, result.IsBlinded, true)
	})

	// Test for Bellatrix version
	t.Run("bellatrix block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
		require.NoError(t, err)
		result, err := vs.constructGenericBeaconBlock(b, nil, nil)
		require.NoError(t, err)
		r1, err := result.GetBellatrix().HashTreeRoot()
		require.NoError(t, err)
		r2, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, result.IsBlinded, false)
	})

	// Test for Altair version
	t.Run("altair block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockAltair())
		require.NoError(t, err)
		result, err := vs.constructGenericBeaconBlock(b, nil, nil)
		require.NoError(t, err)
		r1, err := result.GetAltair().HashTreeRoot()
		require.NoError(t, err)
		r2, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, result.IsBlinded, false)
	})

	// Test for phase0 version
	t.Run("phase0 block", func(t *testing.T) {
		b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
		require.NoError(t, err)
		result, err := vs.constructGenericBeaconBlock(b, nil, nil)
		require.NoError(t, err)
		r1, err := result.GetPhase0().HashTreeRoot()
		require.NoError(t, err)
		r2, err := b.Block().HashTreeRoot()
		require.NoError(t, err)
		require.Equal(t, r1, r2)
		require.Equal(t, result.IsBlinded, false)
	})
}
