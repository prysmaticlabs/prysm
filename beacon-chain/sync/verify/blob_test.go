package verify

import (
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestBlobAlignsWithBlock(t *testing.T) {
	tests := []struct {
		name        string
		block       interfaces.ReadOnlySignedBeaconBlock
		blob        *ethpb.BlobSidecar
		expectedErr string
	}{
		{
			name: "Block version less than Deneb",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlock()
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{},
		},
		{
			name: "No commitments in block",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{},
		},
		{
			name: "Blob index exceeds max blobs per block",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, fieldparams.MaxBlobsPerBlock+1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob:        &ethpb.BlobSidecar{Index: fieldparams.MaxBlobsPerBlock},
			expectedErr: "blob index 6 >= max blobs per block 6: incorrect blob index",
		},
		{
			name: "Blob slot does not match block slot",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Slot = 2
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob:        &ethpb.BlobSidecar{Slot: 1},
			expectedErr: "slot 2 != BlobSidecar.Slot 1: BlockSlot in BlobSidecar does not match the expected slot",
		},
		{
			name: "Blob block root does not match block root",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{BlockRoot: []byte{1}},
			expectedErr: "block root 0x0200000000000000000000000000000000000000000000000000000000000000 != " +
				"BlobSidecar.BlockRoot 0x0100000000000000000000000000000000000000000000000000000000000000 at slot 0",
		},
		{
			name: "Blob parent root does not match block parent root",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{BlockRoot: []byte{2}, BlockParentRoot: []byte{1}},
			expectedErr: "block parent root 0x0000000000000000000000000000000000000000000000000000000000000000 != " +
				"BlobSidecar.BlockParentRoot 0x0100000000000000000000000000000000000000000000000000000000000000 at slot 0",
		},
		{
			name: "Blob proposer index does not match block proposer index",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob:        &ethpb.BlobSidecar{BlockRoot: []byte{2}, ProposerIndex: 1},
			expectedErr: "proposer index 0 != BlobSidecar.ProposerIndex 1 at slot ",
		},
		{
			name: "Blob commitment does not match block commitment",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{BlockRoot: []byte{2}, KzgCommitment: []byte{1}},
			expectedErr: "commitment 0x010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 != " +
				"block commitment 0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name: "All fields are correctly aligned",
			block: func() interfaces.ReadOnlySignedBeaconBlock {
				b := util.NewBeaconBlockDeneb()
				b.Block.Body.BlobKzgCommitments = make([][]byte, 1)
				sb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return sb
			}(),
			blob: &ethpb.BlobSidecar{BlockRoot: []byte{2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := blocks.NewROBlockWithRoot(tt.block, [32]byte{2})
			require.NoError(t, err)
			err = BlobAlignsWithBlock(tt.blob, block)
			if tt.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.StringContains(t, tt.expectedErr, err.Error())
			}
		})
	}
}
