package validator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	bt "github.com/prysmaticlabs/prysm/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestServer_getBuilderBlock(t *testing.T) {
	tests := []struct {
		name        string
		blk         interfaces.SignedBeaconBlock
		mock        *bt.MockBuilderService
		err         string
		returnedBlk interfaces.SignedBeaconBlock
	}{
		{
			name: "old block version",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := wrapper.WrappedSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "not configured",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := wrapper.WrappedSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
			mock: &bt.MockBuilderService{
				HasConfigured: false,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := wrapper.WrappedSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "builder is not ready",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := wrapper.WrappedSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
			mock: &bt.MockBuilderService{
				HasConfigured: true,
				ErrStatus:     errors.New("builder is not ready"),
			},
			err: "builder is not ready",
		},
		{
			name: "submit blind block error",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := wrapper.WrappedSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &bt.MockBuilderService{
				HasConfigured:         true,
				ErrSubmitBlindedBlock: errors.New("can't submit"),
			},
			err: "can't submit",
		},
		{
			name: "can submit block",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := wrapper.WrappedSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &bt.MockBuilderService{
				HasConfigured: true,
				Payload:       &v1.ExecutionPayload{GasLimit: 123},
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = &v1.ExecutionPayload{GasLimit: 123}
				wb, err := wrapper.WrappedSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vs := &Server{BlockBuilder: tc.mock}
			gotBlk, err := vs.getBuilderBlock(context.Background(), tc.blk)
			if err != nil {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.DeepEqual(t, tc.returnedBlk, gotBlk)
			}
		})
	}
}
