package das

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func Test_commitmentsToCheck(t *testing.T) {
	windowSlots, err := slots.EpochEnd(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
	require.NoError(t, err)
	commits := [][]byte{
		bytesutil.PadTo([]byte("a"), 48),
		bytesutil.PadTo([]byte("b"), 48),
		bytesutil.PadTo([]byte("c"), 48),
		bytesutil.PadTo([]byte("d"), 48),
	}
	cases := []struct {
		name    string
		commits [][]byte
		block   func(*testing.T) blocks.ROBlock
		slot    primitives.Slot
	}{
		{
			name: "pre deneb",
			block: func(t *testing.T) blocks.ROBlock {
				bb := util.NewBeaconBlockBellatrix()
				sb, err := blocks.NewSignedBeaconBlock(bb)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
		},
		{
			name: "commitments within da",
			block: func(t *testing.T) blocks.ROBlock {
				d := util.NewBeaconBlockDeneb()
				d.Block.Body.BlobKzgCommitments = commits
				d.Block.Slot = 100
				sb, err := blocks.NewSignedBeaconBlock(d)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
			commits: commits,
			slot:    100,
		},
		{
			name: "commitments outside da",
			block: func(t *testing.T) blocks.ROBlock {
				d := util.NewBeaconBlockDeneb()
				// block is from slot 0, "current slot" is window size +1 (so outside the window)
				d.Block.Body.BlobKzgCommitments = commits
				sb, err := blocks.NewSignedBeaconBlock(d)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
			slot: windowSlots + 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.block(t)
			co := commitmentsToCheck(b, c.slot)
			require.Equal(t, len(c.commits), len(co))
			for i := 0; i < len(c.commits); i++ {
				require.Equal(t, true, bytes.Equal(c.commits[i], co[i]))
			}
		})
	}
}
