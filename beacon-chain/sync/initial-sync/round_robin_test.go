package initialsync

import (
	"reflect"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/sirupsen/logrus"
)

type peerData struct {
	blocks         []uint64 // slots that peer has blocks
	finalizedEpoch uint64
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestRoundRobinSync(t *testing.T) {

	tests := []struct {
		name        string
		currentSlot uint64
		peers       []peerData
	}{
		{
			name:        "Single peer with all blocks",
			currentSlot: 130,
			peers: []peerData{
				{
					blocks: makeSequence(1, 130),
					finalizedEpoch: 1,
				},
			},
		},
		//{
		//	name: "Single peer with skipped epochs",
		//	peers: []peerData{
		//		{blocks: makeSequence(100, 164)},
		//	},
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerstatus.Clear()

			p := p2pt.NewTestP2P(t)
			connectPeers(t, p, tt.peers)

			s := &InitialSync{
				chain: &mock.ChainService{
					State: &p2ppb.BeaconState{},
				}, // no-op mock
				p2p:          p,
				synced:       false,
				chainStarted: true,
			}

			if err := s.roundRobinSync(makeGenesisTime(tt.currentSlot)); err != nil {
				t.Error(err)
			}
		})
	}
}

// Connect peers with local host. This method sets up peer statuses and the appropriate handlers
// for each test peer.
func connectPeers(t *testing.T, host *p2pt.TestP2P, data []peerData) {
	const topic = "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz"

	for _, datum := range data {
		peer := p2pt.NewTestP2P(t)
		peer.SetStreamHandler(topic, func(stream network.Stream) {
			req := &p2ppb.BeaconBlocksByRangeRequest{}
			if err := peer.Encoding().DecodeWithLength(stream, req); err != nil {
				t.Error(err)
			}

			// TODO: error ranges.

			// Write success response.
			if _, err := stream.Write([]byte{0x00}); err != nil {
				t.Error(err)
			}

			// Determine the correct subset of blocks to return as dictated by the test scenario.
			blocks := sliceutil.IntersectionUint64(datum.blocks, makeSequence(req.StartSlot, req.StartSlot+(req.Count*req.Step)))
			ret := make([]*eth.BeaconBlock, 0)
			for _, slot := range blocks {
				if slot%req.Step != 0 {
					continue
				}
				ret = append(ret, &eth.BeaconBlock{Slot: slot})
			}

			if _, err := peer.Encoding().EncodeWithLength(stream, ret); err != nil {
				t.Error(err)
			}
		})

		peer.Connect(host)

		peerstatus.Set(peer.PeerID(), &p2ppb.Status{
			HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
			FinalizedRoot:   []byte("finalized_root"),
			FinalizedEpoch:  datum.finalizedEpoch,
			HeadRoot:        []byte("head_root"),
			HeadSlot:        0,
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
