package initialsync

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	beaconsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
)

type testCache struct {
	sync.RWMutex
	rootCache       map[uint64][32]byte
	parentSlotCache map[uint64]uint64
}

var cache = &testCache{}

type peerData struct {
	blocks         []uint64 // slots that peer has blocks
	finalizedEpoch uint64
	headSlot       uint64
	failureSlots   []uint64 // slots at which the peer will return an error
	forkedPeer     bool
}

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
		NewStateMgmt:     true,
		BatchBlockVerify: true,
		EnablePeerScorer: true,
	})
	defer resetCfg()

	resetFlags := flags.Get()
	flags.Init(&flags.GlobalFlags{
		BlockBatchLimit:            64,
		BlockBatchLimitBurstFactor: 10,
	})
	defer func() {
		flags.Init(resetFlags)
	}()
	os.Exit(m.Run())
}

func initializeTestServices(t *testing.T, blocks []uint64, peers []*peerData) (*mock.ChainService, *p2pt.TestP2P, db.Database) {
	cache.initializeRootCache(blocks, t)
	beaconDB, _ := dbtest.SetupDB(t)

	p := p2pt.NewTestP2P(t)
	connectPeers(t, p, peers, p.Peers())
	cache.RLock()
	genesisRoot := cache.rootCache[0]
	cache.RUnlock()

	err := beaconDB.SaveBlock(context.Background(), &eth.SignedBeaconBlock{Block: &eth.BeaconBlock{Slot: 0}})
	require.NoError(t, err)

	st, err := stateTrie.InitializeFromProto(&p2ppb.BeaconState{})
	require.NoError(t, err)

	return &mock.ChainService{
		State: st,
		Root:  genesisRoot[:],
		DB:    beaconDB,
	}, p, beaconDB
}

// makeGenesisTime where now is the current slot.
func makeGenesisTime(currentSlot uint64) time.Time {
	return roughtime.Now().Add(-1 * time.Second * time.Duration(currentSlot) * time.Duration(params.BeaconConfig().SecondsPerSlot))
}

// sanity test on helper function
func TestMakeGenesisTime(t *testing.T) {
	currentSlot := uint64(64)
	gt := makeGenesisTime(currentSlot)
	require.Equal(t, currentSlot, helpers.SlotsSince(gt))
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

func (c *testCache) initializeRootCache(reqSlots []uint64, t *testing.T) {
	c.Lock()
	defer c.Unlock()

	c.rootCache = make(map[uint64][32]byte)
	c.parentSlotCache = make(map[uint64]uint64)
	parentSlot := uint64(0)
	genesisBlock := &eth.BeaconBlock{
		Slot: 0,
	}
	genesisRoot, err := stateutil.BlockRoot(genesisBlock)
	require.NoError(t, err)
	c.rootCache[0] = genesisRoot
	parentRoot := genesisRoot
	for _, slot := range reqSlots {
		currentBlock := &eth.BeaconBlock{
			Slot:       slot,
			ParentRoot: parentRoot[:],
		}
		parentRoot, err = stateutil.BlockRoot(currentBlock)
		require.NoError(t, err)
		c.rootCache[slot] = parentRoot
		c.parentSlotCache[slot] = parentSlot
		parentSlot = slot
	}
}

// sanity test on helper function
func TestMakeSequence(t *testing.T) {
	got := makeSequence(3, 5)
	want := []uint64{3, 4, 5}
	require.DeepEqual(t, want, got)
}

// Connect peers with local host. This method sets up peer statuses and the appropriate handlers
// for each test peer.
func connectPeers(t *testing.T, host *p2pt.TestP2P, data []*peerData, peerStatus *peers.Status) {
	for _, d := range data {
		connectPeer(t, host, d, peerStatus)
	}
}

// connectPeer connects a peer to a local host.
func connectPeer(t *testing.T, host *p2pt.TestP2P, datum *peerData, peerStatus *peers.Status) peer.ID {
	const topic = "/eth2/beacon_chain/req/beacon_blocks_by_range/1/ssz_snappy"
	p := p2pt.NewTestP2P(t)
	p.SetStreamHandler(topic, func(stream network.Stream) {
		defer func() {
			if err := stream.Close(); err != nil {
				t.Log(err)
			}
		}()

		req := &p2ppb.BeaconBlocksByRangeRequest{}
		assert.NoError(t, p.Encoding().DecodeWithMaxLength(stream, req))

		requestedBlocks := makeSequence(req.StartSlot, req.StartSlot+(req.Count*req.Step))

		// Expected failure range
		if len(sliceutil.IntersectionUint64(datum.failureSlots, requestedBlocks)) > 0 {
			if _, err := stream.Write([]byte{0x01}); err != nil {
				t.Error(err)
			}
			if _, err := p.Encoding().EncodeWithMaxLength(stream, "bad"); err != nil {
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
			require.NoError(t, err)
			logrus.Tracef("block with slot %d , signing root %#x and parent root %#x", slot, currRoot, parentRoot)
		}

		if uint64(len(ret)) > req.Count {
			ret = ret[:req.Count]
		}

		for i := 0; i < len(ret); i++ {
			assert.NoError(t, beaconsync.WriteChunk(stream, p.Encoding(), ret[i]))
		}
	})

	p.Connect(host)

	peerStatus.Add(new(enr.Record), p.PeerID(), nil, network.DirOutbound)
	peerStatus.SetConnectionState(p.PeerID(), peers.PeerConnected)
	peerStatus.SetChainState(p.PeerID(), &p2ppb.Status{
		ForkDigest:     params.BeaconConfig().GenesisForkVersion,
		FinalizedRoot:  []byte(fmt.Sprintf("finalized_root %d", datum.finalizedEpoch)),
		FinalizedEpoch: datum.finalizedEpoch,
		HeadRoot:       []byte("head_root"),
		HeadSlot:       datum.headSlot,
	})

	return p.PeerID()
}
