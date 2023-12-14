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
	pDeneb := emptyPayloadDeneb()
	pDeneb.GasLimit = 123
	pDeneb.ExcessBlobGas = 456
	pDeneb.BlobGasUsed = 789

	denebblk, denebsidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 0, fieldparams.MaxBlobsPerBlock)
	denebCommitments, err := denebblk.Block().Body().BlobKzgCommitments()
	require.NoError(t, err)
	execution, err := denebblk.Block().Body().Execution()
	require.NoError(t, err)
	denebPayload, err := execution.PbDeneb()
	require.NoError(t, err)

	blobs := make([][]byte, len(denebsidecars))
	for i, sidecar := range denebsidecars {
		blobs[i] = sidecar.BlobSidecar.Blob
	}

	tests := []struct {
		name                 string
		blk                  interfaces.SignedBeaconBlock
		mock                 *builderTest.MockBuilderService
		err                  string
		returnedBlk          interfaces.SignedBeaconBlock
		returnedBlobSidecars []blocks.ROBlob
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
		{
			name: "can get payload and blobs Deneb",
			blk: func() interfaces.SignedBeaconBlock {
				blindedBlock, err := denebblk.ToBlinded()
				require.NoError(t, err)
				b, err := blindedBlock.PbBlindedDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				PayloadDeneb:  denebPayload,
				BlobBundle: &v1.BlobsBundle{
					KzgCommitments: denebCommitments,
					Proofs:         [][]byte{{'d', 0}, {'d', 1}, {'d', 2}, {'d', 3}, {'d', 4}, {'d', 5}},
					Blobs:          blobs,
				},
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				b, err := denebblk.PbDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			returnedBlobSidecars: denebsidecars,
		},
		{
			name: "deneb mismatch commitments count",
			blk: func() interfaces.SignedBeaconBlock {
				blindedBlock, err := denebblk.ToBlinded()
				require.NoError(t, err)
				b, err := blindedBlock.PbBlindedDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				PayloadDeneb:  denebPayload,
				BlobBundle: &v1.BlobsBundle{
					KzgCommitments: [][]byte{{'c', 0}, {'c', 1}, {'c', 2}, {'c', 3}, {'c', 4}},
					Proofs:         [][]byte{{'d', 0}, {'d', 1}, {'d', 2}, {'d', 3}, {'d', 4}, {'d', 5}},
					Blobs:          blobs,
				},
			},
			err: "mismatch commitments count",
		},
		{
			name: "deneb mismatch proofs count",
			blk: func() interfaces.SignedBeaconBlock {
				blindedBlock, err := denebblk.ToBlinded()
				require.NoError(t, err)
				b, err := blindedBlock.PbBlindedDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				PayloadDeneb:  denebPayload,
				BlobBundle: &v1.BlobsBundle{
					KzgCommitments: [][]byte{{'c', 0}, {'c', 1}, {'c', 2}, {'c', 3}, {'c', 4}, {'c', 5}},
					Proofs:         [][]byte{{'d', 0}, {'d', 1}, {'d', 2}, {'d', 3}, {'d', 4}},
					Blobs:          blobs,
				},
			},
			err: "mismatch proofs count",
		},
		{
			name: "deneb different count commitments bundle vs block",
			blk: func() interfaces.SignedBeaconBlock {
				blindedBlock, err := denebblk.ToBlinded()
				require.NoError(t, err)
				b, err := blindedBlock.PbBlindedDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				PayloadDeneb:  denebPayload,
				BlobBundle: &v1.BlobsBundle{
					KzgCommitments: [][]byte{{'c', 0}, {'c', 1}, {'c', 2}, {'c', 3}, {'c', 4}},
					Proofs:         [][]byte{{'d', 0}, {'d', 1}, {'d', 2}, {'d', 3}, {'d', 4}},
					Blobs:          blobs[:5],
				},
			},
			err: "commitment count doesn't match block",
		},
		{
			name: "deneb different value commitments bundle vs block",
			blk: func() interfaces.SignedBeaconBlock {
				blindedBlock, err := denebblk.ToBlinded()
				require.NoError(t, err)
				b, err := blindedBlock.PbBlindedDenebBlock()
				require.NoError(t, err)
				wb, err := blocks.NewSignedBeaconBlock(b)
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: true,
				PayloadDeneb:  denebPayload,
				BlobBundle: &v1.BlobsBundle{
					KzgCommitments: [][]byte{{'c', 0}, {'c', 1}, {'c', 2}, {'c', 3}, {'c', 4}, {'c', 5}},
					Proofs:         [][]byte{{'d', 0}, {'d', 1}, {'d', 2}, {'d', 3}, {'d', 4}, {'d', 5}},
					Blobs:          blobs,
				},
			},
			err: "commitment value doesn't match block",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unblinder, err := newUnblinder(tc.blk, tc.mock)
			require.NoError(t, err)
			gotBlk, gotBlobs, err := unblinder.unblindBuilderBlock(context.Background())
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tc.returnedBlk, gotBlk)
				if tc.returnedBlobSidecars != nil {
					blobs := make([]blocks.ROBlob, len(gotBlobs))
					for i := range gotBlobs {
						blobs[i], err = blocks.NewROBlob(gotBlobs[i])
						require.NoError(t, err)
						// TODO: update this check when generate function is updated for inclusion proofs require.DeepEqual(t, tc.returnedBlobSidecars[i].CommitmentInclusionProof, blobs[i].CommitmentInclusionProof)
						require.DeepEqual(t, tc.returnedBlobSidecars[i].SignedBlockHeader, blobs[i].SignedBlockHeader)
						require.Equal(t, len(tc.returnedBlobSidecars[i].Blob), len(blobs[i].Blob))
					}
				}
			}
		})
	}
}
