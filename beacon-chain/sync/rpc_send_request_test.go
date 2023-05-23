package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	p2pTypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestSendRequest_SendBeaconBlocksByRangeRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pcl := fmt.Sprintf("%s/ssz_snappy", p2p.RPCBlocksByRangeTopicV1)

	t.Run("stream error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		// Bogus peer doesn't support a given protocol, so stream error is expected.
		bogusPeer := p2ptest.NewTestP2P(t)
		p1.Connect(bogusPeer)

		req := &ethpb.BeaconBlocksByRangeRequest{}
		_, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, bogusPeer.PeerID(), req, nil)
		assert.ErrorContains(t, "protocols not supported", err)
	})

	knownBlocks := make([]*ethpb.SignedBeaconBlock, 0)
	genesisBlk := util.NewBeaconBlock()
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	parentRoot := genesisBlkRoot
	for i := 0; i < 255; i++ {
		blk := util.NewBeaconBlock()
		blk.Block.Slot = primitives.Slot(i)
		blk.Block.ParentRoot = parentRoot[:]
		knownBlocks = append(knownBlocks, blk)
		parentRoot, err = blk.Block.HashTreeRoot()
		require.NoError(t, err)
	}

	knownBlocksProvider := func(p2pProvider p2p.P2P, processor BeaconBlockProcessor) func(stream network.Stream) {
		return func(stream network.Stream) {
			defer func() {
				assert.NoError(t, stream.Close())
			}()

			req := &ethpb.BeaconBlocksByRangeRequest{}
			assert.NoError(t, p2pProvider.Encoding().DecodeWithMaxLength(stream, req))

			for i := req.StartSlot; i < req.StartSlot.Add(req.Count*req.Step); i += primitives.Slot(req.Step) {
				if processor != nil {
					wsb, err := blocks.NewSignedBeaconBlock(knownBlocks[i])
					require.NoError(t, err)
					if processorErr := processor(wsb); processorErr != nil {
						if errors.Is(processorErr, io.EOF) {
							// Close stream, w/o any errors written.
							return
						}
						_, err := stream.Write([]byte{0x01})
						assert.NoError(t, err)
						msg := p2pTypes.ErrorMessage(processorErr.Error())
						_, err = p2pProvider.Encoding().EncodeWithMaxLength(stream, &msg)
						assert.NoError(t, err)
						return
					}
				}
				if uint64(i) >= uint64(len(knownBlocks)) {
					break
				}
				wsb, err := blocks.NewSignedBeaconBlock(knownBlocks[i])
				require.NoError(t, err)
				err = WriteBlockChunk(stream, startup.NewClock(time.Now(), [32]byte{}), p2pProvider.Encoding(), wsb)
				if err != nil && err.Error() != network.ErrReset.Error() {
					require.NoError(t, err)
				}
			}
		}
	}

	t.Run("no block processor", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 128, len(blocks))
	})

	t.Run("has block processor - no errors", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// No error from block processor.
		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		blocksFromProcessor := make([]interfaces.ReadOnlySignedBeaconBlock, 0)
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			blocksFromProcessor = append(blocksFromProcessor, block)
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 128, len(blocks))
		assert.DeepEqual(t, blocks, blocksFromProcessor)
	})

	t.Run("has block processor - throw error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// Send error from block processor.
		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		errFromProcessor := errors.New("processor error")
		_, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
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
		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 128, len(blocks))

		// Cap max returned roots.
		cfg := params.BeaconNetworkConfig().Copy()
		maxRequestBlocks := cfg.MaxRequestBlocks
		defer func() {
			cfg.MaxRequestBlocks = maxRequestBlocks
			params.OverrideBeaconNetworkConfig(cfg)
		}()
		blocks, err = SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			// Since ssz checks the boundaries, and doesn't normally allow to send requests bigger than
			// the max request size, we are updating max request size dynamically. Even when updated dynamically,
			// no more than max request size of blocks is expected on return.
			cfg.MaxRequestBlocks = 3
			params.OverrideBeaconNetworkConfig(cfg)
			return nil
		})
		assert.ErrorContains(t, ErrInvalidFetchedData.Error(), err)
		assert.Equal(t, 0, len(blocks))
	})

	t.Run("process custom error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		blocksProcessed := 0
		expectedErr := errors.New("some error")
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			if blocksProcessed > 2 {
				return expectedErr
			}
			blocksProcessed++
			return nil
		}))

		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.ErrorContains(t, expectedErr.Error(), err)
		assert.Equal(t, 0, len(blocks))
	})

	t.Run("blocks out of order: step 1", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)

		// Switch known blocks, so that slots are out of order.
		knownBlocks[30], knownBlocks[31] = knownBlocks[31], knownBlocks[30]
		defer func() {
			knownBlocks[31], knownBlocks[30] = knownBlocks[30], knownBlocks[31]
		}()

		p2.SetStreamHandler(pcl, func(stream network.Stream) {
			defer func() {
				assert.NoError(t, stream.Close())
			}()

			req := &ethpb.BeaconBlocksByRangeRequest{}
			assert.NoError(t, p2.Encoding().DecodeWithMaxLength(stream, req))

			for i := req.StartSlot; i < req.StartSlot.Add(req.Count*req.Step); i += primitives.Slot(req.Step) {
				if uint64(i) >= uint64(len(knownBlocks)) {
					break
				}
				wsb, err := blocks.NewSignedBeaconBlock(knownBlocks[i])
				err = WriteBlockChunk(stream, startup.NewClock(time.Now(), [32]byte{}), p2.Encoding(), wsb)
				if err != nil && err.Error() != network.ErrReset.Error() {
					require.NoError(t, err)
				}
			}
		})

		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      1,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.ErrorContains(t, ErrInvalidFetchedData.Error(), err)
		assert.Equal(t, 0, len(blocks))

	})

	t.Run("blocks out of order: step 10", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)

		// Switch known blocks, so that slots are out of order.
		knownBlocks[30], knownBlocks[31] = knownBlocks[31], knownBlocks[30]
		defer func() {
			knownBlocks[31], knownBlocks[30] = knownBlocks[30], knownBlocks[31]
		}()

		p2.SetStreamHandler(pcl, func(stream network.Stream) {
			defer func() {
				assert.NoError(t, stream.Close())
			}()

			req := &ethpb.BeaconBlocksByRangeRequest{}
			assert.NoError(t, p2.Encoding().DecodeWithMaxLength(stream, req))

			for i := req.StartSlot; i < req.StartSlot.Add(req.Count*req.Step); i += primitives.Slot(req.Step) {
				if uint64(i) >= uint64(len(knownBlocks)) {
					break
				}
				wsb, err := blocks.NewSignedBeaconBlock(knownBlocks[i])
				require.NoError(t, err)
				err = WriteBlockChunk(stream, startup.NewClock(time.Now(), [32]byte{}), p2.Encoding(), wsb)
				if err != nil && err.Error() != network.ErrReset.Error() {
					require.NoError(t, err)
				}
			}
		})

		req := &ethpb.BeaconBlocksByRangeRequest{
			StartSlot: 20,
			Count:     128,
			Step:      10,
		}
		blocks, err := SendBeaconBlocksByRangeRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.ErrorContains(t, ErrInvalidFetchedData.Error(), err)
		assert.Equal(t, 0, len(blocks))

	})
}

func TestSendRequest_SendBeaconBlocksByRootRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pcl := fmt.Sprintf("%s/ssz_snappy", p2p.RPCBlocksByRootTopicV1)

	knownBlocks := make(map[[32]byte]*ethpb.SignedBeaconBlock)
	knownRoots := make([][32]byte, 0)
	for i := 0; i < 5; i++ {
		blk := util.NewBeaconBlock()
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
		_, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, bogusPeer.PeerID(), req, nil)
		assert.ErrorContains(t, "protocols not supported", err)
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
						wsb, err := blocks.NewSignedBeaconBlock(blk)
						require.NoError(t, err)
						if processorErr := processor(wsb); processorErr != nil {
							if errors.Is(processorErr, io.EOF) {
								// Close stream, w/o any errors written.
								return
							}
							_, err := stream.Write([]byte{0x01})
							assert.NoError(t, err)
							msg := p2pTypes.ErrorMessage(processorErr.Error())
							_, err = p2pProvider.Encoding().EncodeWithMaxLength(stream, &msg)
							assert.NoError(t, err)
							return
						}
					}
					_, err := stream.Write([]byte{0x00})
					assert.NoError(t, err, "Could not write to stream")
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
		blocks, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(blocks))
	})

	t.Run("has block processor - no errors", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// No error from block processor.
		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1]}
		blocksFromProcessor := make([]interfaces.ReadOnlySignedBeaconBlock, 0)
		blocks, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			blocksFromProcessor = append(blocksFromProcessor, block)
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(blocks))
		assert.DeepEqual(t, blocks, blocksFromProcessor)
	})

	t.Run("has block processor - throw error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, nil))

		// Send error from block processor.
		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1]}
		errFromProcessor := errors.New("processor error")
		_, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
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
		clock := startup.NewClock(time.Now(), [32]byte{})
		blocks, err := SendBeaconBlocksByRootRequest(ctx, clock, p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(blocks))

		// Cap max returned roots.
		cfg := params.BeaconNetworkConfig().Copy()
		maxRequestBlocks := cfg.MaxRequestBlocks
		defer func() {
			cfg.MaxRequestBlocks = maxRequestBlocks
			params.OverrideBeaconNetworkConfig(cfg)
		}()
		blocks, err = SendBeaconBlocksByRootRequest(ctx, clock, p1, p2.PeerID(), req, func(block interfaces.ReadOnlySignedBeaconBlock) error {
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
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			if blocksProcessed > 2 {
				return expectedErr
			}
			blocksProcessed++
			return nil
		}))

		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1], knownRoots[2], knownRoots[3]}
		blocks, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.ErrorContains(t, expectedErr.Error(), err)
		assert.Equal(t, 0, len(blocks))
	})

	t.Run("process io.EOF error", func(t *testing.T) {
		p1 := p2ptest.NewTestP2P(t)
		p2 := p2ptest.NewTestP2P(t)
		p1.Connect(p2)
		blocksProcessed := 0
		expectedErr := io.EOF
		p2.SetStreamHandler(pcl, knownBlocksProvider(p2, func(block interfaces.ReadOnlySignedBeaconBlock) error {
			if blocksProcessed > 2 {
				return expectedErr
			}
			blocksProcessed++
			return nil
		}))

		req := &p2pTypes.BeaconBlockByRootsReq{knownRoots[0], knownRoots[1], knownRoots[2], knownRoots[3]}
		blocks, err := SendBeaconBlocksByRootRequest(ctx, startup.NewClock(time.Now(), [32]byte{}), p1, p2.PeerID(), req, nil)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(blocks))
	})
}

func TestBlobValidatorFromRootReq(t *testing.T) {
	validRoot := bytesutil.PadTo([]byte("valid"), 32)
	invalidRoot := bytesutil.PadTo([]byte("invalid"), 32)
	cases := []struct {
		name     string
		ids      []*ethpb.BlobIdentifier
		response []*ethpb.BlobSidecar
		err      error
	}{
		{
			name:     "valid",
			ids:      []*ethpb.BlobIdentifier{{BlockRoot: validRoot}},
			response: []*ethpb.BlobSidecar{{BlockRoot: validRoot}},
		},
		{
			name:     "invalid",
			ids:      []*ethpb.BlobIdentifier{{BlockRoot: validRoot}},
			response: []*ethpb.BlobSidecar{{BlockRoot: invalidRoot}},
			err:      ErrUnrequestedRoot,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := p2pTypes.BlobSidecarsByRootReq(c.ids)
			vf := blobValidatorFromRootReq(&r)
			for _, sc := range c.response {
				err := vf(sc)
				if c.err != nil {
					require.ErrorIs(t, err, c.err)
					return
				}
				require.NoError(t, err)
			}
		})
	}
}

func TestBlobValidatorFromRangeReq(t *testing.T) {
	cases := []struct {
		name     string
		req      *ethpb.BlobSidecarsByRangeRequest
		response []*ethpb.BlobSidecar
		err      error
	}{
		{
			name: "valid - count multi",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     10,
			},
			response: []*ethpb.BlobSidecar{{Slot: 14}},
		},
		{
			name: "valid - count 1",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     1,
			},
			response: []*ethpb.BlobSidecar{{Slot: 10}},
		},
		{
			name: "invalid - before",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     1,
			},
			response: []*ethpb.BlobSidecar{{Slot: 9}},
			err:      ErrBlobResponseOutOfBounds,
		},
		{
			name: "invalid - after, count 1",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     1,
			},
			response: []*ethpb.BlobSidecar{{Slot: 11}},
			err:      ErrBlobResponseOutOfBounds,
		},
		{
			name: "invalid - after, multi",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     10,
			},
			response: []*ethpb.BlobSidecar{{Slot: 23}},
			err:      ErrBlobResponseOutOfBounds,
		},
		{
			name: "invalid - after, at boundary, multi",
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: 10,
				Count:     10,
			},
			response: []*ethpb.BlobSidecar{{Slot: 20}},
			err:      ErrBlobResponseOutOfBounds,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vf := blobValidatorFromRangeReq(c.req)
			for _, sc := range c.response {
				err := vf(sc)
				if c.err != nil {
					require.ErrorIs(t, err, c.err)
					return
				}
				require.NoError(t, err)
			}
		})
	}
}
