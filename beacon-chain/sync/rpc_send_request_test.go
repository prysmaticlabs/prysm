package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/network"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	p2pTypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSendRequest_SendBeaconBlocksByRangeRequest(t *testing.T) {
}

func TestSendRequest_SendBeaconBlocksByRootRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pcl := fmt.Sprintf("%s/ssz_snappy", p2p.RPCBlocksByRootTopic)

	knownBlocks := make(map[[32]byte]*eth.SignedBeaconBlock, 0)
	knownRoots := make([][32]byte, 0)
	for i := 0; i < 5; i++ {
		blk := testutil.NewBeaconBlock()
		blkRoot, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		knownRoots = append(knownRoots, blkRoot)
		knownBlocks[knownRoots[len(knownRoots)-1]] = blk
	}

	t.Run("stream error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		// Bogus peer doesn't support a given protocol, so stream error is expected.
		bogusPeer := p2ptest.NewTestP2P(t)
		p1.Connect(bogusPeer)

		req := &p2pTypes.BeaconBlockByRootsReq{}
		_, err := SendBeaconBlocksByRootRequest(ctx, p1, bogusPeer.PeerID(), req, nil)
		assert.ErrorContains(t, "protocol not supported", err)
	})

	knownBlocksProvider := func(p2pProvider p2p.P2P, processor BeaconBlockProcessor) func(stream network.Stream) {
		return func(stream network.Stream) {
			defer func() {
				assert.NoError(t, stream.Close())
			}()

			req := new(p2pTypes.BeaconBlockByRootsReq)
			assert.NoError(t, p2pProvider.Encoding().DecodeWithMaxLength(stream, req))
			if len(*req) == 0 {
				return
			}
			for _, root := range *req {
				if blk, ok := knownBlocks[root]; ok {
					if processor != nil {
						if processorErr := processor(blk); processorErr != nil {
							_, err := stream.Write([]byte{0x01})
							assert.NoError(t, err)
							msg := p2pTypes.ErrorMessage(processorErr.Error())
							_, err = p2pProvider.Encoding().EncodeWithMaxLength(stream, &msg)
							assert.NoError(t, err)
							return
						}
					}
					_, err := stream.Write([]byte{0x00})
					assert.NoError(t, err, "Failed to write to stream")
					_, err = p2pProvider.Encoding().EncodeWithMaxLength(stream, blk)
					assert.NoError(t, err, "Could not send response back")
				}
			}
		}
	}

	t.Run("no block processor", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1]}
		blocks, err := SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(blocks))
	})

	t.Run("has block processor", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// No error from block processor.
		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1]}
		blocksFromProcessor := make([]*eth.SignedBeaconBlock, 0)
		blocks, err := SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, func(block *eth.SignedBeaconBlock) error {
			blocksFromProcessor = append(blocksFromProcessor, block)
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(blocks))
		assert.DeepEqual(t, blocks, blocksFromProcessor)

		// Send error from block processor.
		errFromProcessor := errors.New("processor error")
		_, err = SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, func(block *eth.SignedBeaconBlock) error {
			return errFromProcessor
		})
		assert.ErrorContains(t, errFromProcessor.Error(), err)
	})

	t.Run("max request blocks", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// No cap on max roots.
		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1], knownRoots[2], knownRoots[3]}
		blocks, err := SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(blocks))

		// Cap max returned roots.
		cfg := params.BeaconNetworkConfig().Copy()
		maxRequestBlocks := cfg.MaxRequestBlocks
		defer func() {
			cfg.MaxRequestBlocks = maxRequestBlocks
			params.OverrideBeaconNetworkConfig(cfg)
		}()
		blocks, err = SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, func(block *eth.SignedBeaconBlock) error {
			// Since ssz checks the boundaries, and doesn't normally allow to send requests bigger than
			// the max request size, we are updating max request size dynamically. Even when updated dynamically,
			// no more than max request size of blocks is expected on return.
			cfg.MaxRequestBlocks = 3
			params.OverrideBeaconNetworkConfig(cfg)
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(blocks))
	})

	t.Run("process custom error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		blocksProcessed := 0
		expectedErr := errors.New("some error")
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, func(block *eth.SignedBeaconBlock) error {
			if blocksProcessed > 2 {
				return expectedErr
			}
			blocksProcessed++
			return nil
		}))

		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1], knownRoots[2], knownRoots[3]}
		blocks, err := SendBeaconBlocksByRootRequest(ctx, p1, p2.PeerID(), req, nil)
		assert.ErrorContains(t, expectedErr.Error(), err)
		assert.Equal(t, 0, len(blocks))
	})
}
