package blockchain

import (
	"bytes"
	"context"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Ensure ChainService implements chain info interface.
var _ = ChainInfoRetriever(&ChainService{})

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	c := setupBeaconChain(t, db)

	s := &pb.BeaconState{}
	if err := c.forkChoiceStore.GenesisStore(ctx, s); err != nil {
		t.Fatal(err)
	}

	if c.FinalizedCheckpt().Epoch != 0 {
		t.Errorf("Finalized epoch at genesis should be 0, got: %d", c.FinalizedCheckpt().Epoch)
	}
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &ChainService{}
	c.headSlot = 100
	if c.HeadSlot() != 100 {
		t.Errorf("Wanted head slot: %d, got: %d", 100, c.HeadSlot())
	}
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	c := &ChainService{canonicalRoots: make(map[uint64][]byte)}
	c.headSlot = 100
	c.canonicalRoots[c.headSlot] = []byte{'A'}
	if !bytes.Equal([]byte{'A'}, c.HeadRoot()) {
		t.Errorf("Wanted head root: %v, got: %d", []byte{'A'}, c.HeadRoot())
	}
}

func TestCanonicalRoot_CanRetrieve(t *testing.T) {
	c := &ChainService{canonicalRoots: make(map[uint64][]byte)}
	slot := uint64(123)
	c.canonicalRoots[slot] = []byte{'B'}
	if !bytes.Equal([]byte{'B'}, c.CanonicalRoot(slot)) {
		t.Errorf("Wanted head root: %v, got: %d", []byte{'A'}, c.CanonicalRoot(slot))
	}
}
