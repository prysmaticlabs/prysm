package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/proto"
)

// Ensure Service implements chain info interface.
var _ ChainInfoFetcher = (*Service)(nil)
var _ TimeFetcher = (*Service)(nil)
var _ ForkFetcher = (*Service)(nil)

func TestHeadRoot_Nil(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	c := setupBeaconChain(t, beaconDB)
	headRoot, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], headRoot, "Incorrect pre chain start value")
}

func TestService_ForkChoiceStore(t *testing.T) {
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New(0, 0)}}
	p := c.ForkChoiceStore()
	require.Equal(t, 0, int(p.FinalizedEpoch()))
}

func TestFinalizedCheckpt_CanRetrieve(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 5, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c := setupBeaconChain(t, beaconDB)
	c.store.SetFinalizedCheckptAndPayloadHash(cp, [32]byte{'a'})

	cp, err := c.FinalizedCheckpt()
	require.NoError(t, err)
	assert.Equal(t, cp.Epoch, cp.Epoch, "Unexpected finalized epoch")
}

func TestFinalizedCheckpt_GenesisRootOk(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	genesisRoot := [32]byte{'A'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, beaconDB)
	c.store.SetFinalizedCheckptAndPayloadHash(cp, [32]byte{'a'})
	c.originBlockRoot = genesisRoot
	cp, err := c.FinalizedCheckpt()
	require.NoError(t, err)
	assert.DeepEqual(t, c.originBlockRoot[:], cp.Root)
}

func TestCurrentJustifiedCheckpt_CanRetrieve(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	c := setupBeaconChain(t, beaconDB)
	_, err := c.CurrentJustifiedCheckpt()
	require.ErrorIs(t, err, store.ErrNilCheckpoint)
	cp := &ethpb.Checkpoint{Epoch: 6, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c.store.SetJustifiedCheckptAndPayloadHash(cp, [32]byte{})
	jp, err := c.CurrentJustifiedCheckpt()
	require.NoError(t, err)
	assert.Equal(t, cp.Epoch, jp.Epoch, "Unexpected justified epoch")
}

func TestJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	c := setupBeaconChain(t, beaconDB)
	genesisRoot := [32]byte{'B'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c.store.SetJustifiedCheckptAndPayloadHash(cp, [32]byte{})
	c.originBlockRoot = genesisRoot
	cp, err := c.CurrentJustifiedCheckpt()
	require.NoError(t, err)
	assert.DeepEqual(t, c.originBlockRoot[:], cp.Root)
}

func TestPreviousJustifiedCheckpt_CanRetrieve(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	cp := &ethpb.Checkpoint{Epoch: 7, Root: bytesutil.PadTo([]byte("foo"), 32)}
	c := setupBeaconChain(t, beaconDB)
	_, err := c.PreviousJustifiedCheckpt()
	require.ErrorIs(t, err, store.ErrNilCheckpoint)
	c.store.SetPrevJustifiedCheckpt(cp)
	pcp, err := c.PreviousJustifiedCheckpt()
	require.NoError(t, err)
	assert.Equal(t, cp.Epoch, pcp.Epoch, "Unexpected previous justified epoch")
}

func TestPrevJustifiedCheckpt_GenesisRootOk(t *testing.T) {
	beaconDB := testDB.SetupDB(t)

	genesisRoot := [32]byte{'C'}
	cp := &ethpb.Checkpoint{Root: genesisRoot[:]}
	c := setupBeaconChain(t, beaconDB)
	c.store.SetPrevJustifiedCheckpt(cp)
	c.originBlockRoot = genesisRoot
	pcp, err := c.PreviousJustifiedCheckpt()
	require.NoError(t, err)
	assert.DeepEqual(t, c.originBlockRoot[:], pcp.Root)
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &Service{}
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	c.head = &head{slot: 100, state: s}
	assert.Equal(t, types.Slot(100), c.HeadSlot())
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	c := &Service{}
	c.head = &head{root: [32]byte{'A'}}
	r, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, [32]byte{'A'}, bytesutil.ToBytes32(r))
}

func TestHeadRoot_UseDB(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	c := &Service{cfg: &config{BeaconDB: beaconDB}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:]}))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(context.Background(), br))
	r, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, br, bytesutil.ToBytes32(r))
}

func TestHeadBlock_CanRetrieve(t *testing.T) {
	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{block: wsb, state: s}

	recevied, err := c.HeadBlock(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, b, recevied.Proto(), "Incorrect head block received")
}

func TestHeadState_CanRetrieve(t *testing.T) {
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Slot: 2, GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:]})
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
	f := &ethpb.Fork{Epoch: 999}
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Fork: f})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.CurrentFork(), f) {
		t.Error("Received incorrect fork version")
	}
}

func TestCurrentFork_NilHeadSTate(t *testing.T) {
	f := &ethpb.Fork{
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
	}
	c := &Service{}
	if !proto.Equal(c.CurrentFork(), f) {
		t.Error("Received incorrect fork version")
	}
}

func TestGenesisValidatorsRoot_CanRetrieve(t *testing.T) {
	// Should not panic if head state is nil.
	c := &Service{}
	assert.Equal(t, [32]byte{}, c.GenesisValidatorsRoot(), "Did not get correct genesis validators root")

	s, err := v1.InitializeFromProto(&ethpb.BeaconState{GenesisValidatorsRoot: []byte{'a'}})
	require.NoError(t, err)
	c.head = &head{state: s}
	assert.Equal(t, [32]byte{'a'}, c.GenesisValidatorsRoot(), "Did not get correct genesis validators root")
}

func TestHeadETH1Data_Nil(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	c := setupBeaconChain(t, beaconDB)
	assert.DeepEqual(t, &ethpb.Eth1Data{}, c.HeadETH1Data(), "Incorrect pre chain start value")
}

func TestHeadETH1Data_CanRetrieve(t *testing.T) {
	d := &ethpb.Eth1Data{DepositCount: 999}
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{Eth1Data: d})
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{state: s}
	if !proto.Equal(c.HeadETH1Data(), d) {
		t.Error("Received incorrect eth1 data")
	}
}

func TestIsCanonical_Ok(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	c := setupBeaconChain(t, beaconDB)

	blk := util.NewBeaconBlock()
	blk.Block.Slot = 0
	root, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(blk)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))
	can, err := c.IsCanonical(ctx, root)
	require.NoError(t, err)
	assert.Equal(t, true, can)

	can, err = c.IsCanonical(ctx, [32]byte{'a'})
	require.NoError(t, err)
	assert.Equal(t, false, can)
}

func TestService_HeadValidatorsIndices(t *testing.T) {
	s, _ := util.DeterministicGenesisState(t, 10)
	c := &Service{}

	c.head = &head{}
	indices, err := c.HeadValidatorsIndices(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 0, len(indices))

	c.head = &head{state: s}
	indices, err = c.HeadValidatorsIndices(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 10, len(indices))
}

func TestService_HeadGenesisValidatorsRoot(t *testing.T) {
	s, _ := util.DeterministicGenesisState(t, 1)
	c := &Service{}

	c.head = &head{}
	root := c.HeadGenesisValidatorsRoot()
	require.Equal(t, [32]byte{}, root)

	c.head = &head{state: s}
	root = c.HeadGenesisValidatorsRoot()
	require.DeepEqual(t, root[:], s.GenesisValidatorsRoot())
}
func TestService_ChainHeads_ProtoArray(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New(0, 0)}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 0, 0))

	roots, slots := c.ChainHeads()
	require.DeepEqual(t, [][32]byte{{'c'}, {'d'}, {'e'}}, roots)
	require.DeepEqual(t, []types.Slot{102, 103, 104}, slots)
}

func TestService_ChainHeads_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New(0, 0)}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 103, [32]byte{'d'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 104, [32]byte{'e'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, 0, 0))

	roots, slots := c.ChainHeads()
	require.Equal(t, 3, len(roots))
	rootMap := map[[32]byte]types.Slot{[32]byte{'c'}: 102, [32]byte{'d'}: 103, [32]byte{'e'}: 104}
	for i, root := range roots {
		slot, ok := rootMap[root]
		require.Equal(t, true, ok)
		require.Equal(t, slot, slots[i])
	}
}

func TestService_HeadPublicKeyToValidatorIndex(t *testing.T) {
	s, _ := util.DeterministicGenesisState(t, 10)
	c := &Service{}
	c.head = &head{state: s}

	_, e := c.HeadPublicKeyToValidatorIndex([fieldparams.BLSPubkeyLength]byte{})
	require.Equal(t, false, e)

	v, err := s.ValidatorAtIndex(0)
	require.NoError(t, err)

	i, e := c.HeadPublicKeyToValidatorIndex(bytesutil.ToBytes48(v.PublicKey))
	require.Equal(t, true, e)
	require.Equal(t, types.ValidatorIndex(0), i)
}

func TestService_HeadPublicKeyToValidatorIndexNil(t *testing.T) {
	c := &Service{}
	c.head = nil

	idx, e := c.HeadPublicKeyToValidatorIndex([fieldparams.BLSPubkeyLength]byte{})
	require.Equal(t, false, e)
	require.Equal(t, types.ValidatorIndex(0), idx)

	c.head = &head{state: nil}
	i, e := c.HeadPublicKeyToValidatorIndex([fieldparams.BLSPubkeyLength]byte{})
	require.Equal(t, false, e)
	require.Equal(t, types.ValidatorIndex(0), i)
}

func TestService_HeadValidatorIndexToPublicKey(t *testing.T) {
	s, _ := util.DeterministicGenesisState(t, 10)
	c := &Service{}
	c.head = &head{state: s}

	p, err := c.HeadValidatorIndexToPublicKey(context.Background(), 0)
	require.NoError(t, err)

	v, err := s.ValidatorAtIndex(0)
	require.NoError(t, err)

	require.Equal(t, bytesutil.ToBytes48(v.PublicKey), p)
}

func TestService_HeadValidatorIndexToPublicKeyNil(t *testing.T) {
	c := &Service{}
	c.head = nil

	p, err := c.HeadValidatorIndexToPublicKey(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, [fieldparams.BLSPubkeyLength]byte{}, p)

	c.head = &head{state: nil}
	p, err = c.HeadValidatorIndexToPublicKey(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, [fieldparams.BLSPubkeyLength]byte{}, p)
}

func TestService_IsOptimistic_ProtoArray(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))

	opt, err := c.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimistic_DoublyLinkedTree(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))

	opt, err := c.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimisticBeforeBellatrix(t *testing.T) {
	ctx := context.Background()
	c := &Service{genesisTime: time.Now()}
	opt, err := c.IsOptimistic(ctx)
	require.NoError(t, err)
	require.Equal(t, false, opt)
}

func TestService_IsOptimisticForRoot_ProtoArray(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))

	opt, err := c.IsOptimisticForRoot(ctx, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimisticForRoot_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, c.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, 0, 0))

	opt, err := c.IsOptimisticForRoot(ctx, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimisticForRoot_DB_ProtoArray(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(optimisticBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(validatedBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedCheckpoint := &ethpb.Checkpoint{Root: br[:]}
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, validatedCheckpoint))

	_, err = c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.ErrorContains(t, "nil summary returned from the DB", err)

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: optimisticRoot[:], Slot: 11}))
	optimistic, err := c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: validatedRoot[:], Slot: 9}))
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  validatedRoot[:],
	}
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, validatedRoot))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, cp))

	validated, err := c.IsOptimisticForRoot(ctx, validatedRoot)
	require.NoError(t, err)
	require.Equal(t, false, validated)

	// Before the first finalized epoch, finalized root could be zeros.
	validatedCheckpoint = &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, br))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: params.BeaconConfig().ZeroHash[:], Slot: 10}))
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, validatedCheckpoint))

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: optimisticRoot[:], Slot: 11}))
	optimistic, err = c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
}

func TestService_IsOptimisticForRoot_DB_DoublyLinkedTree(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: doublylinkedtree.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(optimisticBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(validatedBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedCheckpoint := &ethpb.Checkpoint{Root: br[:]}
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, validatedCheckpoint))

	_, err = c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.ErrorContains(t, "nil summary returned from the DB", err)

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: optimisticRoot[:], Slot: 11}))
	optimistic, err := c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: validatedRoot[:], Slot: 9}))
	cp := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  validatedRoot[:],
	}
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, validatedRoot))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, cp))
	validated, err := c.IsOptimisticForRoot(ctx, validatedRoot)
	require.NoError(t, err)
	require.Equal(t, false, validated)

	// Before the first finalized epoch, finalized root could be zeros.
	validatedCheckpoint = &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, br))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: params.BeaconConfig().ZeroHash[:], Slot: 10}))
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, validatedCheckpoint))

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: optimisticRoot[:], Slot: 11}))
	optimistic, err = c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)
}

func TestService_IsOptimisticForRoot_DB_non_canonical(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: doublylinkedtree.New(0, 0)}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(optimisticBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = wrapper.WrappedSignedBeaconBlock(validatedBlock)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wsb))

	validatedCheckpoint := &ethpb.Checkpoint{Root: br[:]}
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, validatedCheckpoint))

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: optimisticRoot[:], Slot: 11}))
	optimistic, err := c.IsOptimisticForRoot(ctx, optimisticRoot)
	require.NoError(t, err)
	require.Equal(t, true, optimistic)

	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: validatedRoot[:], Slot: 9}))
	validated, err := c.IsOptimisticForRoot(ctx, validatedRoot)
	require.NoError(t, err)
	require.Equal(t, true, validated)

}
