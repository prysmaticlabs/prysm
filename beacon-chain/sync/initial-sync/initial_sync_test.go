package initialsync

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"sync"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2pt "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
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
	os.Exit(m.Run())
}

func initializeTestServices(t *testing.T, blocks []uint64, peers []*peerData) (*mock.ChainService, *p2pt.TestP2P, db.Database) {
	cache.initializeRootCache(blocks, t)
	beaconDB := dbtest.SetupDB(t)

	p := p2pt.NewTestP2P(t)
	connectPeers(t, p, peers, p.Peers())
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
	if helpers.SlotsSince(gt) != currentSlot {
		t.Fatalf("Wanted %d, got %d", currentSlot, helpers.SlotsSince(gt))
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
	if err != nil {
		t.Fatal(err)
	}
	c.rootCache[0] = genesisRoot
	parentRoot := genesisRoot
	for _, slot := range reqSlots {
		currentBlock := &eth.BeaconBlock{
			Slot:       slot,
			ParentRoot: parentRoot[:],
		}
		parentRoot, err = stateutil.BlockRoot(currentBlock)
		if err != nil {
			t.Fatal(err)
		}
		c.rootCache[slot] = parentRoot
		c.parentSlotCache[slot] = parentSlot
		parentSlot = slot
	}
}

// sanity test on helper function
func TestMakeSequence(t *testing.T) {
	got := makeSequence(3, 5)
	want := []uint64{3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Wanted %v, got %v", want, got)
	}
}
