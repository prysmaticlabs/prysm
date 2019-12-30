package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// Ensure Service implements chain info interface.
var _ = ChainInfoFetcher(&Service{})
var _ = GenesisTimeFetcher(&Service{})
var _ = ForkFetcher(&Service{})

func TestFinalizedCheckpt_Nil(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	c := setupBeaconChain(t, db)
	c.headState, _ = testutil.DeterministicGenesisState(t, 1)
	if !bytes.Equal(c.FinalizedCheckpt().Root, params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestHeadRoot_Nil(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	c := setupBeaconChain(t, db)
	if !bytes.Equal(c.HeadRoot(), params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Epoch: 5}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{FinalizedCheckpoint: cp}

	if c.FinalizedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Finalized epoch at genesis should be %d, got: %d", cp.Epoch, c.FinalizedCheckpt().Epoch)
	}
}

func TestFinalizedCheckpt_GenesisRootOk(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{FinalizedCheckpoint: cp}
	c.genesisRoot = [32]byte{'A'}

	if !bytes.Equal(c.FinalizedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.FinalizedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestCurrentJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Epoch: 6}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{CurrentJustifiedCheckpoint: cp}

	if c.CurrentJustifiedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Current Justifiied epoch at genesis should be %d, got: %d", cp.Epoch, c.CurrentJustifiedCheckpt().Epoch)
	}
}

func TestJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{CurrentJustifiedCheckpoint: cp}
	c.genesisRoot = [32]byte{'B'}

	if !bytes.Equal(c.CurrentJustifiedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.CurrentJustifiedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestPreviousJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Epoch: 7}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{PreviousJustifiedCheckpoint: cp}

	if c.PreviousJustifiedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Previous Justifiied epoch at genesis should be %d, got: %d", cp.Epoch, c.PreviousJustifiedCheckpt().Epoch)
	}
}

func TestPrevJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cp := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	c := setupBeaconChain(t, db)
	c.headState = &pb.BeaconState{PreviousJustifiedCheckpoint: cp}
	c.genesisRoot = [32]byte{'C'}

	if !bytes.Equal(c.PreviousJustifiedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.PreviousJustifiedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &Service{}
	c.headSlot = 100
	if c.HeadSlot() != 100 {
		t.Errorf("Wanted head slot: %d, got: %d", 100, c.HeadSlot())
	}
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	c := &Service{canonicalRoots: make(map[uint64][]byte)}
	c.headSlot = 100
	c.canonicalRoots[c.headSlot] = []byte{'A'}
	if !bytes.Equal([]byte{'A'}, c.HeadRoot()) {
		t.Errorf("Wanted head root: %v, got: %d", []byte{'A'}, c.HeadRoot())
	}
}

func TestHeadBlock_CanRetrieve(t *testing.T) {
	b := &ethpb.BeaconBlock{Slot: 1}
	c := &Service{headBlock: b}
	if !reflect.DeepEqual(b, c.HeadBlock()) {
		t.Error("incorrect head block received")
	}
}

func TestHeadState_CanRetrieve(t *testing.T) {
	s := &pb.BeaconState{Slot: 2}
	c := &Service{headState: s}
	headState, err := c.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, headState) {
		t.Error("incorrect head state received")
	}
}

func TestGenesisTime_CanRetrieve(t *testing.T) {
	c := &Service{genesisTime: time.Unix(999, 0)}
	wanted := time.Unix(999, 0)
	if c.GenesisTime() != wanted {
		t.Error("Did not get wanted genesis time")
	}
}

func TestCurrentFork_CanRetrieve(t *testing.T) {
	f := &pb.Fork{Epoch: 999}
	s := &pb.BeaconState{Fork: f}
	c := &Service{headState: s}
	if !reflect.DeepEqual(c.CurrentFork(), f) {
		t.Error("Recieved incorrect fork version")
	}
}

func TestCanonicalRoot_CanRetrieve(t *testing.T) {
	c := &Service{canonicalRoots: make(map[uint64][]byte)}
	slot := uint64(123)
	r := []byte{'B'}
	c.canonicalRoots[slot] = r
	if !bytes.Equal(r, c.CanonicalRoot(slot)) {
		t.Errorf("Wanted head root: %v, got: %d", []byte{'A'}, c.CanonicalRoot(slot))
	}
}
