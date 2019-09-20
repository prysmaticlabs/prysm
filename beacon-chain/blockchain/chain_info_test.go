package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Ensure Service implements chain info interface.
var _ = ChainInfoFetcher(&Service{})
var _ = GenesisTimeFetcher(&Service{})

func TestFinalizedCheckpt_Nil(t *testing.T) {
	c := setupBeaconChain(t, nil)
	if !bytes.Equal(c.FinalizedCheckpt().Root, params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestHeadRoot_Nil(t *testing.T) {
	c := setupBeaconChain(t, nil)
	if !bytes.Equal(c.HeadRoot(), params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	c := setupBeaconChain(t, db)

	if err := c.forkChoiceStore.GenesisStore(ctx, &ethpb.Checkpoint{}, &ethpb.Checkpoint{}); err != nil {
		t.Fatal(err)
	}

	if c.FinalizedCheckpt().Epoch != 0 {
		t.Errorf("Finalized epoch at genesis should be 0, got: %d", c.FinalizedCheckpt().Epoch)
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
	if !reflect.DeepEqual(s, c.HeadState()) {
		t.Error("incorrect head state received")
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
