package initialsync

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/network"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	beaconsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
)

func TestConstants(t *testing.T) {
	if params.BeaconConfig().MaxPeersToSync*blockBatchSize > 1000 {
		t.Fatal("rpc rejects requests over 1000 range slots")
	}
}

func TestRoundRobinSync(t *testing.T) {

	tests := []struct {
		name               string
		currentSlot        uint64
		expectedBlockSlots []uint64
		peers              []*peerData
	}{
		{
			name:               "Single peer with all blocks",
			currentSlot:        131,
			expectedBlockSlots: makeSequence(1, 131),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
			},
		},
		{
			name:               "Multiple peers with all blocks",
			currentSlot:        131,
			expectedBlockSlots: makeSequence(1, 131),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
				{
					blocks:         makeSequence(1, 131),
					finalizedEpoch: 1,
					headSlot:       131,
				},
			},
		},
		{
			name:               "Multiple peers with failures",
			currentSlot:        320, // 10 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
					failureSlots:   makeSequence(1, 32), // first epoch
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 8,
					headSlot:       320,
				},
			},
		},
		{
			name:               "Multiple peers with many skipped slots",
			currentSlot:        640, // 10 epochs
			expectedBlockSlots: append(makeSequence(1, 64), makeSequence(500, 640)...),
			peers: []*peerData{
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 18,
					headSlot:       640,
				},
			},
		},

		// TODO(3147): Handle multiple failures.
		//{
		//	name:               "Multiple peers with multiple failures",
		//	currentSlot:        320, // 10 epochs
		//	expectedBlockSlots: makeSequence(1, 320),
		//	peers: []*peerData{
		//		{
		//			blocks:         makeSequence(1, 320),
		//			finalizedEpoch: 4,
		//			headSlot:       320,
		//		},
		//		{
		//			blocks:         makeSequence(1, 320),
		//			finalizedEpoch: 4,
		//			headSlot:       320,
		//			failureSlots:   makeSequence(1, 320),
		//		},
		//		{
		//			blocks:         makeSequence(1, 320),
		//			finalizedEpoch: 4,
		//			headSlot:       320,
		//			failureSlots:   makeSequence(1, 320),
		//		},
		//		{
		//			blocks:         makeSequence(1, 320),
		//			finalizedEpoch: 4,
		//			headSlot:       320,
		//			failureSlots:   makeSequence(1, 320),
		//		},
		//	},
		//},
		{
			name:               "Multiple peers with different finalized epoch",
			currentSlot:        320, // 10 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 256),
					finalizedEpoch: 3,
					headSlot:       256,
				},
				{
					blocks:         makeSequence(1, 256),
					finalizedEpoch: 3,
					headSlot:       256,
				},
				{
					blocks:         makeSequence(1, 192),
					finalizedEpoch: 2,
					headSlot:       192,
				},
			},
		},
		{
			name:               "Multiple peers with missing parent blocks",
			currentSlot:        160, // 5 epochs
			expectedBlockSlots: makeSequence(1, 160),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         append(makeSequence(1, 6), makeSequence(161, 165)...),
					finalizedEpoch: 4,
					headSlot:       160,
					forkedPeer:     true,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
				{
					blocks:         makeSequence(1, 160),
					finalizedEpoch: 4,
					headSlot:       160,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.initializeRootCache(tt.expectedBlockSlots, t)

			p := p2pt.NewTestP2P(t)
			beaconDB := dbtest.SetupDB(t)

			connectPeers(t, p, tt.peers, p.Peers())
			cache.RLock()
			genesisRoot := cache.rootCache[0]
			cache.RUnlock()

			err := beaconDB.SaveBlock(context.Background(), &eth.SignedBeaconBlock{
				Block: &eth.BeaconBlock{
					Slot: 0,
				}})
			if err != nil {
				t.Fatal(err)
			}

			st, err := stateTrie.InitializeFromProto(&p2ppb.BeaconState{})
			if err != nil {
				t.Fatal(err)
			}
			mc := &mock.ChainService{
				State: st,
				Root:  genesisRoot[:],
				DB:    beaconDB,
			} // no-op mock
			s := &Service{
				chain:             mc,
				blockNotifier:     mc.BlockNotifier(),
				p2p:               p,
				db:                beaconDB,
				synced:            false,
				chainStarted:      true,
				blocksRateLimiter: leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksPerSecond, false /* deleteEmptyBuckets */),
			}
			if err := s.roundRobinSync(makeGenesisTime(tt.currentSlot)); err != nil {
				t.Error(err)
			}
			if s.chain.HeadSlot() != tt.currentSlot {
				t.Errorf("Head slot (%d) is not currentSlot (%d)", s.chain.HeadSlot(), tt.currentSlot)
			}
			if len(mc.BlocksReceived) != len(tt.expectedBlockSlots) {
				t.Errorf("Processes wrong number of blocks. Wanted %d got %d", len(tt.expectedBlockSlots), len(mc.BlocksReceived))
			}
			var receivedBlockSlots []uint64
			for _, blk := range mc.BlocksReceived {
				receivedBlockSlots = append(receivedBlockSlots, blk.Block.Slot)
			}
			if missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots); len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

// Connect peers with local host. This method sets up peer statuses and the appropriate handlers
// for each test peer.
func connectPeers(t *testing.T, host *p2pt.TestP2P, data []*peerData, peerStatus *peers.Status) {
	const topic = "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz"

	for _, d := range data {
		peer := p2pt.NewTestP2P(t)

		// Copy pointer for callback scope.
		var datum = d

		peer.SetStreamHandler(topic, func(stream network.Stream) {
			defer func() {
				if err := stream.Close(); err != nil {
					t.Log(err)
				}
			}()

			req := &p2ppb.BeaconBlocksByRangeRequest{}
			if err := peer.Encoding().DecodeWithLength(stream, req); err != nil {
				t.Error(err)
			}

			requestedBlocks := makeSequence(req.StartSlot, req.StartSlot+(req.Count*req.Step))

			// Expected failure range
			if len(sliceutil.IntersectionUint64(datum.failureSlots, requestedBlocks)) > 0 {
				if _, err := stream.Write([]byte{0x01}); err != nil {
					t.Error(err)
				}
				if _, err := peer.Encoding().EncodeWithLength(stream, "bad"); err != nil {
					t.Error(err)
				}
				return
			}

			// Determine the correct subset of blocks to return as dictated by the test scenario.
			blocks := sliceutil.IntersectionUint64(datum.blocks, requestedBlocks)

			ret := make([]*eth.SignedBeaconBlock, 0)
			for _, slot := range blocks {
				if (slot-req.StartSlot)%req.Step != 0 {
					continue
				}
				cache.RLock()
				parentRoot := cache.rootCache[cache.parentSlotCache[slot]]
				cache.RUnlock()
				blk := &eth.SignedBeaconBlock{
					Block: &eth.BeaconBlock{
						Slot:       slot,
						ParentRoot: parentRoot[:],
					},
				}
				// If forked peer, give a different parent root.
				if datum.forkedPeer {
					newRoot := hashutil.Hash(parentRoot[:])
					blk.Block.ParentRoot = newRoot[:]
				}
				ret = append(ret, blk)
				currRoot, err := stateutil.BlockRoot(blk.Block)
				if err != nil {
					t.Fatal(err)
				}
				logrus.Infof("block with slot %d , signing root %#x and parent root %#x", slot, currRoot, parentRoot)
			}

			if uint64(len(ret)) > req.Count {
				ret = ret[:req.Count]
			}

			for i := 0; i < len(ret); i++ {
				if err := beaconsync.WriteChunk(stream, peer.Encoding(), ret[i]); err != nil {
					t.Error(err)
				}
			}
		})

		peer.Connect(host)

		peerStatus.Add(new(enr.Record), peer.PeerID(), nil, network.DirOutbound)
		peerStatus.SetConnectionState(peer.PeerID(), peers.PeerConnected)
		peerStatus.SetChainState(peer.PeerID(), &p2ppb.Status{
			ForkDigest:     params.BeaconConfig().GenesisForkVersion,
			FinalizedRoot:  []byte(fmt.Sprintf("finalized_root %d", datum.finalizedEpoch)),
			FinalizedEpoch: datum.finalizedEpoch,
			HeadRoot:       []byte("head_root"),
			HeadSlot:       datum.headSlot,
		})
	}
}
