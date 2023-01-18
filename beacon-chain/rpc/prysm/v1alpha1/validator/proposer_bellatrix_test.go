package validator

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	blockchainTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestServer_getPayloadHeader(t *testing.T) {
	emptyRoot, err := ssz.TransactionsRoot([][]byte{})
	require.NoError(t, err)
	ti, err := slots.ToTime(uint64(time.Now().Unix()), 0)
	require.NoError(t, err)

	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: bytesutil.PadTo([]byte{1}, fieldparams.RootLength),
			ParentHash:       params.BeaconConfig().ZeroHash[:],
			Timestamp:        uint64(ti.Unix()),
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}

	require.NoError(t, err)
	tests := []struct {
		name           string
		head           interfaces.SignedBeaconBlock
		mock           *builderTest.MockBuilderService
		fetcher        *blockchainTest.ChainService
		err            string
		returnedHeader *v1.ExecutionPayloadHeader
	}{
		{
			name: "head is not bellatrix ready",
			mock: &builderTest.MockBuilderService{},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
					require.NoError(t, err)
					return wb
				}(),
			},
		},
		{
			name: "get header failed",
			mock: &builderTest.MockBuilderService{
				ErrGetHeader: errors.New("can't get header"),
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "can't get header",
		},
		{
			name: "0 bid",
			mock: &builderTest.MockBuilderService{
				Bid: &ethpb.SignedBuilderBid{
					Message: &ethpb.BuilderBid{
						Header: &v1.ExecutionPayloadHeader{
							BlockNumber: 123,
						},
					},
				},
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "builder returned header with 0 bid amount",
		},
		{
			name: "invalid tx root",
			mock: &builderTest.MockBuilderService{
				Bid: &ethpb.SignedBuilderBid{
					Message: &ethpb.BuilderBid{
						Value: []byte{1},
						Header: &v1.ExecutionPayloadHeader{
							BlockNumber:      123,
							TransactionsRoot: emptyRoot[:],
						},
					},
				},
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "builder returned header with an empty tx root",
		},
		{
			name: "can get header",
			mock: &builderTest.MockBuilderService{
				Bid: sBid,
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.SignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					return wb
				}(),
			},
			returnedHeader: bid.Header,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vs := &Server{BlockBuilder: tc.mock, HeadFetcher: tc.fetcher, TimeFetcher: &blockchainTest.ChainService{
				Genesis: time.Now(),
			}}
			h, err := vs.getPayloadHeaderFromBuilder(context.Background(), 0, 0)
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				if tc.returnedHeader != nil {
					want, err := blocks.WrappedExecutionPayloadHeader(tc.returnedHeader)
					require.NoError(t, err)
					require.DeepEqual(t, want, h)
				}
			}
		})
	}
}

func TestServer_getBuilderBlock(t *testing.T) {
	p := emptyPayload()
	p.GasLimit = 123

	tests := []struct {
		name        string
		blk         interfaces.SignedBeaconBlock
		mock        *builderTest.MockBuilderService
		err         string
		returnedBlk interfaces.SignedBeaconBlock
	}{
		{
			name: "nil block",
			blk:  nil,
			err:  "signed beacon block can't be nil",
		},
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
			name: "not configured",
			blk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
				require.NoError(t, err)
				return wb
			}(),
			mock: &builderTest.MockBuilderService{
				HasConfigured: false,
			},
			returnedBlk: func() interfaces.SignedBeaconBlock {
				wb, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockBellatrix())
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
			name: "can get payload",
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vs := &Server{BlockBuilder: tc.mock}
			gotBlk, err := vs.unblindBuilderBlock(context.Background(), tc.blk)
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tc.returnedBlk, gotBlk)
			}
		})
	}
}

func TestServer_validateBuilderSignature(t *testing.T) {
	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			ParentHash:       make([]byte, fieldparams.RootLength),
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: make([]byte, fieldparams.RootLength),
			BlockNumber:      1,
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	sBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}
	require.NoError(t, validateBuilderSignature(sBid))

	sBid.Message.Value = make([]byte, 32)
	require.ErrorIs(t, validateBuilderSignature(sBid), signing.ErrSigFailedToVerify)
}
