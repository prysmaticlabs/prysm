package validator

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func Test_unblindBuilderBlock(t *testing.T) {
	p := emptyPayload()
	p.GasLimit = 123
	pCapella := emptyPayloadCapella()
	pCapella.GasLimit = 123

	tests := []struct {
		name        string
		blk         interfaces.SignedBeaconBlock
		mock        *builderTest.MockBuilderService
		err         string
		returnedBlk interfaces.SignedBeaconBlock
	}{
		{
			name: "old block version",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "blinded without configured builder",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
			err: "builder not configured",
		},
		{
			name: "non-blinded without configured builder",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = &v1.ExecutionPayload{
					ParentHash:    make([]byte, fieldparams.RootLength),
					FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					ReceiptsRoot:  make([]byte, fieldparams.RootLength),
					LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:    make([]byte, fieldparams.RootLength),
					BaseFeePerGas: make([]byte, fieldparams.RootLength),
					BlockHash:     make([]byte, fieldparams.RootLength),
					Transactions:  make([][]byte, 0),
					GasLimit:      123,
				}
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: false,
				Payload:       p,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = p
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "submit blind block error",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				Payload:               &v1.ExecutionPayload{},
				HasConfigured:         true,
				ErrSubmitBlindedBlock: errors.New("can't submit"),
			},
			err: "can't submit",
		},
		{
			name: "head and payload root mismatch",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				Payload:       p,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = p
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			err: "header and payload root do not match",
		},
		{
			name: "can get payload Bellatrix",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				txRoot, err := ssz.TransactionsRoot([][]byte{})
				require.NoError(t, err)
				b.Block.Body.ExecutionPayloadHeader = &v1.ExecutionPayloadHeader{
					ParentHash:       make([]byte, fieldparams.RootLength),
					FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:        make([]byte, fieldparams.RootLength),
					ReceiptsRoot:     make([]byte, fieldparams.RootLength),
					LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:       make([]byte, fieldparams.RootLength),
					BaseFeePerGas:    make([]byte, fieldparams.RootLength),
					BlockHash:        make([]byte, fieldparams.RootLength),
					TransactionsRoot: txRoot[:],
					GasLimit:         123,
				}
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				Payload:       p,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockBellatrix()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.ExecutionPayload = p
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
		},
		{
			name: "can get payload Capella",
			blk: func() interfaces.SignedBeaconBlock {
				b := util.NewBlindedBeaconBlockCapella()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.BlsToExecutionChanges = []*eth.SignedBLSToExecutionChange{
					{
						Message: &eth.BLSToExecutionChange{
							ValidatorIndex:     123,
							FromBlsPubkey:      []byte{'a'},
							ToExecutionAddress: []byte{'a'},
						},
						Signature: []byte("sig123"),
					},
					{
						Message: &eth.BLSToExecutionChange{
							ValidatorIndex:     456,
							FromBlsPubkey:      []byte{'b'},
							ToExecutionAddress: []byte{'b'},
						},
						Signature: []byte("sig456"),
					},
				}
				txRoot, err := ssz.TransactionsRoot([][]byte{})
				require.NoError(t, err)
				withdrawalsRoot, err := ssz.WithdrawalSliceRoot([]*v1.Withdrawal{}, fieldparams.MaxWithdrawalsPerPayload)
				require.NoError(t, err)
				b.Block.Body.ExecutionPayloadHeader = &v1.ExecutionPayloadHeaderCapella{
					ParentHash:       make([]byte, fieldparams.RootLength),
					FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:        make([]byte, fieldparams.RootLength),
					ReceiptsRoot:     make([]byte, fieldparams.RootLength),
					LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:       make([]byte, fieldparams.RootLength),
					BaseFeePerGas:    make([]byte, fieldparams.RootLength),
					BlockHash:        make([]byte, fieldparams.RootLength),
					TransactionsRoot: txRoot[:],
					WithdrawalsRoot:  withdrawalsRoot[:],
					GasLimit:         123,
				}
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured:  true,
				PayloadCapella: pCapella,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b := util.NewBeaconBlockCapella()
				b.Block.Slot = 1
				b.Block.ProposerIndex = 2
				b.Block.Body.BlsToExecutionChanges = []*eth.SignedBLSToExecutionChange{
					{
						Message: &eth.BLSToExecutionChange{
							ValidatorIndex:     123,
							FromBlsPubkey:      []byte{'a'},
							ToExecutionAddress: []byte{'a'},
						},
						Signature: []byte("sig123"),
					},
					{
						Message: &eth.BLSToExecutionChange{
							ValidatorIndex:     456,
							FromBlsPubkey:      []byte{'b'},
							ToExecutionAddress: []byte{'b'},
						},
						Signature: []byte("sig456"),
					},
				}
				b.Block.Body.ExecutionPayload = pCapella
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unblinder, err := newUnblinder(tc.blk, tc.mock)
			require.NoError(t, err)
			gotBlk, err := unblinder.unblindBuilderBlock(context.Background())
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tc.returnedBlk, gotBlk)
			}
		})
	}
}
