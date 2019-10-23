package initialsync

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
)

type peerData struct {
	blocks         []uint64 // slots that peer has blocks
	finalizedEpoch uint64
	headSlot       uint64
	failureSlots   []uint64 // slots at which the peer will return an error
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestConstants(t *testing.T) {
	if maxPeersToSync*blockBatchSize > 1000 {
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
			currentSlot:        320, // 5 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
					failureSlots:   makeSequence(1, 64), // first epoch
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
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
					finalizedEpoch: 9,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 9,
					headSlot:       640,
				},
				{
					blocks:         append(makeSequence(1, 64), makeSequence(500, 640)...),
					finalizedEpoch: 9,
					headSlot:       640,
				},
			},
		},
		// TODO(3147): Handle multiple failures.
		{
			name:               "Multiple peers with multiple failures",
			currentSlot:        320, // 5 epochs
			expectedBlockSlots: makeSequence(1, 320),
			peers: []*peerData{
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
				{
					blocks:         makeSequence(1, 320),
					finalizedEpoch: 4,
					headSlot:       320,
					failureSlots:   makeSequence(1, 320),
				},
			},
		},
		{
			name:               "Multiple peers with different finalized epoch",
			currentSlot:        320, // 5 epochs
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerstatus.Clear()

			p := p2pt.NewTestP2P(t)
			connectPeers(t, p, tt.peers)

			mc := &mock.ChainService{
				State: &p2ppb.BeaconState{},
			} // no-op mock
			s := &InitialSync{
				chain:        mc,
				p2p:          p,
				synced:       false,
				chainStarted: true,
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
				receivedBlockSlots = append(receivedBlockSlots, blk.Slot)
			}
			if missing := sliceutil.NotUint64(sliceutil.IntersectionUint64(tt.expectedBlockSlots, receivedBlockSlots), tt.expectedBlockSlots); len(missing) > 0 {
				t.Errorf("Missing blocks at slots %v", missing)
			}
		})
	}
}

// Connect peers with local host. This method sets up peer statuses and the appropriate handlers
// for each test peer.
func connectPeers(t *testing.T, host *p2pt.TestP2P, data []*peerData) {
	const topic = "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz"

	for _, d := range data {
		peer := p2pt.NewTestP2P(t)

		// Copy pointer for callback scope.
		var datum = d

		peer.SetStreamHandler(topic, func(stream network.Stream) {
			defer stream.Close()

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

			ret := make([]*eth.BeaconBlock, 0)
			for _, slot := range blocks {
				if (slot-req.StartSlot)%req.Step != 0 {
					continue
				}
				ret = append(ret, &eth.BeaconBlock{Slot: slot})
			}

			if uint64(len(ret)) > req.Count {
				ret = ret[:req.Count]
			}

			for i := 0; i < len(ret); i++ {
				if err := sync.WriteChunk(stream, peer.Encoding(), ret[i]); err != nil {
					t.Error(err)
				}
			}
		})

		peer.Connect(host)

		peerstatus.Set(peer.PeerID(), &p2ppb.Status{
			HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
			FinalizedRoot:   []byte(fmt.Sprintf("finalized_root %d", datum.finalizedEpoch)),
			FinalizedEpoch:  datum.finalizedEpoch,
			HeadRoot:        []byte("head_root"),
			HeadSlot:        datum.headSlot,
		})
	}
}

// makeGenesisTime where now is the current slot.
func makeGenesisTime(currentSlot uint64) time.Time {
	return roughtime.Now().Add(-1 * time.Second * time.Duration(currentSlot) * time.Duration(params.BeaconConfig().SecondsPerSlot))
}

// sanity test on helper function
func TestMakeGenesisTime(t *testing.T) {
	currentSlot := uint64(64)
	gt := makeGenesisTime(currentSlot)
	if slotsSinceGenesis(gt) != currentSlot {
		t.Fatalf("Wanted %d, got %d", currentSlot, slotsSinceGenesis(gt))
	}
}

// helper function for sequences of block slots
func makeSequence(start, end uint64) []uint64 {
	if end < start {
		panic("cannot make sequence where end is before start")
	}
	seq := make([]uint64, 0, end-start+1)
	for i := start; i <= end; i++ {
		seq = append(seq, i)
	}
	return seq
}

// sanity test on helper function
func TestMakeSequence(t *testing.T) {
	got := makeSequence(3, 5)
	want := []uint64{3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Wanted %v, got %v", want, got)
	}
}

func TestBestFinalized_returnsMaxValue(t *testing.T) {
	defer peerstatus.Clear()

	for i := 0; i <= maxPeersToSync+100; i++ {
		peerstatus.Set(peer.ID(i), &pb.Status{
			FinalizedEpoch: 10,
		})
	}

	_, _, pids := bestFinalized()
	if len(pids) != maxPeersToSync {
		t.Fatalf("returned wrong number of peers, wanted %d, got %d", maxPeersToSync, len(pids))
	}
}
