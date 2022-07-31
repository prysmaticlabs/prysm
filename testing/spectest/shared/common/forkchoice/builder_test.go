package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestBuilderTick(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	blk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	builder := NewBuilder(t, st, blk)
	builder.Tick(t, 10)

	require.Equal(t, int64(10), builder.lastTick)
}

func TestBuilderInvalidBlock(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	blk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	builder := NewBuilder(t, st, blk)
	builder.InvalidBlock(t, blk)
}

func TestPoWBlock(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	blk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	builder := NewBuilder(t, st, blk)
	builder.PoWBlock(&ethpb.PowBlock{BlockHash: []byte{1, 2, 3}})

	require.Equal(t, 1, len(builder.execMock.powBlocks))
}
