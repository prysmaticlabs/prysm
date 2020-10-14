package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

// Ensure Service implements chain info interface.
var _ ChainInfoFetcher = (*Service)(nil)
var _ TimeFetcher = (*Service)(nil)
var _ ForkFetcher = (*Service)(nil)

func TestFinalizedCheckpt_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], c.FinalizedCheckpt().Root, "Incorrect pre chain start value")
}

func TestHeadRoot_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	headRoot, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], headRoot, "Incorrect pre chain start value")
}

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 5, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c := setupBeaconChain(t, db, sc)
	c.finalizedCheckpt = cp

	assert.Equal(t, cp.Epoch, c.FinalizedCheckpt().Epoch, "Unexpected finalized epoch")
}

func TestFinalizedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'A'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.finalizedCheckpt = cp
	c.genesisRoot = genesisRoot
	assert.DeepEqual(t, c.genesisRoot[:], c.FinalizedCheckpt().Root)
}

func TestCurrentJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 6, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c := setupBeaconChain(t, db, sc)
	c.justifiedCheckpt = cp

	assert.Equal(t, cp.Epoch, c.CurrentJustifiedCheckpt().Epoch, "Unexpected justified epoch")
}

func TestJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'B'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.justifiedCheckpt = cp
	c.genesisRoot = genesisRoot
	assert.DeepEqual(t, c.genesisRoot[:], c.CurrentJustifiedCheckpt().Root)
}

func TestPreviousJustifiedCheckpt_CanRetrieve(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 7, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c := setupBeaconChain(t, db, sc)
	c.prevJustifiedCheckpt = cp
	assert.Equal(t, cp.Epoch, c.PreviousJustifiedCheckpt().Epoch, "Unexpected previous justified epoch")
}

func TestPrevJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	db, sc := testDB.SetupDB(t)

	genesisRoot := [32]byte{'C'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, db, sc)
	c.prevJustifiedCheckpt = cp
	c.genesisRoot = genesisRoot
	assert.DeepEqual(t, c.genesisRoot[:], c.PreviousJustifiedCheckpt().Root)
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &Service{}
	s, err := state.InitializeFromProto(&pb.BeaconState{})
	require.NoError(t, err)
	c.head = &head{slot: 100, state: s}
	assert.Equal(t, uint64(100), c.HeadSlot())
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	c := &Service{}
	c.head = &head{root: [32]byte{'A'}}
	r, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, [32]byte{'A'}, bytesutil.ToBytes32(r))
}

func TestHeadBlock_CanRetrieve(t *testing.T) {
	b := testutil.NewBeaconBlock()
	b.Block.Slot = 1
	s, err := state.InitializeFromProto(&pb.BeaconState{})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{block: b, state: s}

	recevied, err := c.HeadBlock(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, b, recevied, "Incorrect head block received")
}

func TestHeadState_CanRetrieve(t *testing.T) {
	s, err := state.InitializeFromProto(&pb.BeaconState{Slot: 2, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{state: s}
	headState, err := c.HeadState(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Incorrect head state received")
}

func TestGenesisTime_CanRetrieve(t *testing.T) {
	c := &Service{genesisTime: time.Unix(999, 0)}
	wanted := time.Unix(999, 0)
	assert.Equal(t, wanted, c.GenesisTime(), "Did not get wanted genesis time")
}

func TestCurrentFork_CanRetrieve(t *testing.T) {
	f := &pb.Fork{Epoch: 999}
	s, err := state.InitializeFromProto(&pb.BeaconState{Fork: f})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.CurrentFork(), f) {
		t.Error("Received incorrect fork version")
	}
}

func TestGenesisValidatorRoot_CanRetrieve(t *testing.T) {
	// Should not panic if head state is nil.
	c := &Service{}
	assert.Equal(t, [32]byte{}, c.GenesisValidatorRoot(), "Did not get correct genesis validator root")

	s, err := state.InitializeFromProto(&pb.BeaconState{GenesisValidatorsRoot: []byte{'a'}})
	require.NoError(t, err)
	c.head = &head{state: s}
	assert.Equal(t, [32]byte{'a'}, c.GenesisValidatorRoot(), "Did not get correct genesis validator root")
}

func TestHeadETH1Data_Nil(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	c := setupBeaconChain(t, db, sc)
	assert.DeepEqual(t, &ethpb.Eth1Data{}, c.HeadETH1Data(), "Incorrect pre chain start value")
}

func TestHeadETH1Data_CanRetrieve(t *testing.T) {
	d := &ethpb.Eth1Data{DepositCount: 999}
	s, err := state.InitializeFromProto(&pb.BeaconState{Eth1Data: d})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.HeadETH1Data(), d) {
		t.Error("Received incorrect eth1 data")
	}
}
