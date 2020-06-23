package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Ensure Service implements chain info interface.
var _ = ChainInfoFetcher(&Service{})
var _ = TimeFetcher(&Service{})
var _ = ForkFetcher(&Service{})

func TestFinalizedCheckpt_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	if !bytes.Equal(c.FinalizedCheckpt().Root, params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestHeadRoot_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	headRoot, err := c.HeadRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(headRoot, params.BeaconConfig().ZeroHash[:]) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 5, Root: []byte("foo")}
	c := setupBeaconChain(t, db, sc)
	c.finalizedCheckpt = cp

	if c.FinalizedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Finalized epoch at genesis should be %d, got: %d", cp.Epoch, c.FinalizedCheckpt().Epoch)
	}
}

func TestFinalizedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'A'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.finalizedCheckpt = cp
	c.genesisRoot = genesisRoot

	if !bytes.Equal(c.FinalizedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.FinalizedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestCurrentJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 6, Root: []byte("foo")}
	c := setupBeaconChain(t, db, sc)
	c.justifiedCheckpt = cp

	if c.CurrentJustifiedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Current Justifiied epoch at genesis should be %d, got: %d", cp.Epoch, c.CurrentJustifiedCheckpt().Epoch)
	}
}

func TestJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'B'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.justifiedCheckpt = cp
	c.genesisRoot = genesisRoot

	if !bytes.Equal(c.CurrentJustifiedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.CurrentJustifiedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestPreviousJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 7, Root: []byte("foo")}
	c := setupBeaconChain(t, db, sc)
	c.prevJustifiedCheckpt = cp

	if c.PreviousJustifiedCheckpt().Epoch != cp.Epoch {
		t.Errorf("Previous Justifiied epoch at genesis should be %d, got: %d", cp.Epoch, c.PreviousJustifiedCheckpt().Epoch)
	}
}

func TestPrevJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'C'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.prevJustifiedCheckpt = cp
	c.genesisRoot = genesisRoot

	if !bytes.Equal(c.PreviousJustifiedCheckpt().Root, c.genesisRoot[:]) {
		t.Errorf("Got: %v, wanted: %v", c.PreviousJustifiedCheckpt().Root, c.genesisRoot[:])
	}
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &Service{}
	s, err := state.InitializeFromProto(&pb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	c.head = &head{slot: 100, state: s}
	if c.HeadSlot() != 100 {
		t.Errorf("Wanted head slot: %d, got: %d", 100, c.HeadSlot())
	}
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	c := &Service{}
	c.head = &head{root: [32]byte{'A'}}
	if [32]byte{'A'} != c.headRoot() {
		t.Errorf("Wanted head root: %v, got: %d", []byte{'A'}, c.headRoot())
	}
}

func TestHeadBlock_CanRetrieve(t *testing.T) {
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	s, err := state.InitializeFromProto(&pb.BeaconState{})
	if err != nil {
		t.Fatal(err)
	}
	c := &Service{}
	c.head = &head{block: b, state: s}

	recevied, err := c.HeadBlock(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(b, recevied) {
		t.Error("incorrect head block received")
	}
}

func TestHeadState_CanRetrieve(t *testing.T) {
	s, err := state.InitializeFromProto(&pb.BeaconState{Slot: 2, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	if err != nil {
		t.Fatal(err)
	}
	c := &Service{}
	c.head = &head{state: s}
	headState, err := c.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(s.InnerStateUnsafe(), headState.InnerStateUnsafe()) {
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
	s, err := state.InitializeFromProto(&pb.BeaconState{Fork: f})
	if err != nil {
		t.Fatal(err)
	}
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.CurrentFork(), f) {
		t.Error("Received incorrect fork version")
	}
}

func TestGenesisValidatorRoot_CanRetrieve(t *testing.T) {
	// Should not panic if head state is nil.
	c := &Service{}
	if c.GenesisValidatorRoot() != [32]byte{} {
		t.Error("Did not get correct genesis validator root")
	}

	s, err := state.InitializeFromProto(&pb.BeaconState{GenesisValidatorsRoot: []byte{'a'}})
	if err != nil {
		t.Fatal(err)
	}
	c.head = &head{state: s}
	if c.GenesisValidatorRoot() != [32]byte{'a'} {
		t.Error("Did not get correct genesis validator root")
	}
}

func TestHeadETH1Data_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	if !reflect.DeepEqual(c.HeadETH1Data(), &ethpb.Eth1Data{}) {
		t.Error("Incorrect pre chain start value")
	}
}

func TestHeadETH1Data_CanRetrieve(t *testing.T) {
	d := &ethpb.Eth1Data{DepositCount: 999}
	s, err := state.InitializeFromProto(&pb.BeaconState{Eth1Data: d})
	if err != nil {
		t.Fatal(err)
	}
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.HeadETH1Data(), d) {
		t.Error("Received incorrect eth1 data")
	}
}
