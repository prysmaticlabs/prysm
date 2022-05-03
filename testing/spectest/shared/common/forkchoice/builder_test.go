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

func TestBuilderBlock(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	blk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	builder := NewBuilder(t, st, blk)
	builder.Block(t, blk, true /*invalid*/)
}

func TestPoWBlock(t *testing.T) {
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	blk, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	builder := NewBuilder(t, st, blk)
	builder.PoWBlock(t, &ethpb.PowBlock{BlockHash: []byte{1, 2, 3}})

	require.Equal(t, 1, len(builder.execMock.powBlocks))
}
