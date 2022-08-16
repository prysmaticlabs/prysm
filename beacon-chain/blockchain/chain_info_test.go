package blockchain

import (
	"context"
	"testing"
	"time"

	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	v3 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v3"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/proto"
)

// Ensure Service implements chain info interface.
var _ ChainInfoFetcher = (*Service)(nil)
var _ TimeFetcher = (*Service)(nil)
var _ ForkFetcher = (*Service)(nil)

// prepareForkchoiceState prepares a beacon state with the given data to mock
// insert into forkchoice
func prepareForkchoiceState(
	_ context.Context,
	slot types.Slot,
	blockRoot [32]byte,
	parentRoot [32]byte,
	payloadHash [32]byte,
	justified *ethpb.Checkpoint,
	finalized *ethpb.Checkpoint,
) (state.BeaconState, [32]byte, error) {
	blockHeader := &ethpb.BeaconBlockHeader{
		ParentRoot: parentRoot[:],
	}

	executionHeader := &enginev1.ExecutionPayloadHeader{
		BlockHash: payloadHash[:],
	}

	base := &ethpb.BeaconStateBellatrix{
		Slot:                         slot,
		RandaoMixes:                  make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		BlockRoots:                   make([][]byte, 1),
		CurrentJustifiedCheckpoint:   justified,
		FinalizedCheckpoint:          finalized,
		LatestExecutionPayloadHeader: executionHeader,
		LatestBlockHeader:            blockHeader,
	}

	base.BlockRoots[0] = append(base.BlockRoots[0], blockRoot[:]...)
	st, err := v3.InitializeFromProto(base)
	return st, blockRoot, err
}

func TestHeadRoot_Nil(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	c := setupBeaconChain(t, beaconDB)
	headRoot, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], headRoot, "Incorrect pre chain start value")
}

func TestService_ForkChoiceStore(t *testing.T) {
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New()}}
	p := c.ForkChoiceStore()
	require.Equal(t, types.Epoch(0), p.FinalizedCheckpoint().Epoch)
}

func TestFinalizedCheckpt_GenesisRootOk(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(fcs),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	gs, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))
	cp := service.FinalizedCheckpt()
	assert.DeepEqual(t, [32]byte{}, bytesutil.ToBytes32(cp.Root))
	cp = service.CurrentJustifiedCheckpt()
	assert.DeepEqual(t, [32]byte{}, bytesutil.ToBytes32(cp.Root))
	// check that forkchoice has the right genesis root as the node root
	root, err := fcs.Head(ctx, []uint64{})
	require.NoError(t, err)
	require.Equal(t, service.originBlockRoot, root)

}

func TestCurrentJustifiedCheckpt_CanRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(fcs),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	cp := &forkchoicetypes.Checkpoint{Epoch: 6, Root: [32]byte{'j'}}
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(cp))
	jp := service.CurrentJustifiedCheckpt()
	assert.Equal(t, cp.Epoch, jp.Epoch, "Unexpected justified epoch")
	require.Equal(t, cp.Root, bytesutil.ToBytes32(jp.Root))
}

func TestHeadSlot_CanRetrieve(t *testing.T) {
	c := &Service{}
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	c.head = &head{slot: 100, state: s}
	assert.Equal(t, types.Slot(100), c.HeadSlot())
}

func TestHeadRoot_CanRetrieve(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(fcs),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	gs, _ := util.DeterministicGenesisState(t, 32)
	require.NoError(t, service.saveGenesisData(ctx, gs))

	r, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	assert.Equal(t, service.originBlockRoot, bytesutil.ToBytes32(r))
}

func TestHeadRoot_UseDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(fcs),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	service.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: br[:]}))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, br))
	r, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	assert.Equal(t, br, bytesutil.ToBytes32(r))
}

func TestHeadBlock_CanRetrieve(t *testing.T) {
	b := util.NewBeaconBlock()
	b.Block.Slot = 1
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{})
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	c := &Service{}
	c.head = &head{block: wsb, state: s}

	received, err := c.HeadBlock(context.Background())
	require.NoError(t, err)
	pb, err := received.Proto()
	require.NoError(t, err)
	assert.DeepEqual(t, b, pb, "Incorrect head block received")
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
	util.SaveBlock(t, ctx, beaconDB, blk)
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
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New()}}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	st, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

	roots, slots := c.ChainHeads()
	require.DeepEqual(t, [][32]byte{{'c'}, {'d'}, {'e'}}, roots)
	require.DeepEqual(t, []types.Slot{102, 103, 104}, slots)
}

//
//  A <- B <- C
//   \    \
//    \    ---------- E
//     ---------- D

func TestService_ChainHeads_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New()}}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	st, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 102, [32]byte{'c'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 103, [32]byte{'d'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 104, [32]byte{'e'}, [32]byte{'b'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

	roots, slots := c.ChainHeads()
	require.Equal(t, 3, len(roots))
	rootMap := map[[32]byte]types.Slot{{'c'}: 102, {'d'}: 103, {'e'}: 104}
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
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	st, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

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
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	st, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

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
	c := &Service{cfg: &config{ForkChoiceStore: protoarray.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	st, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

	opt, err := c.IsOptimisticForRoot(ctx, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimisticForRoot_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	c := &Service{cfg: &config{ForkChoiceStore: doublylinkedtree.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	st, blkRoot, err := prepareForkchoiceState(ctx, 100, [32]byte{'a'}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))
	st, blkRoot, err = prepareForkchoiceState(ctx, 101, [32]byte{'b'}, [32]byte{'a'}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, c.cfg.ForkChoiceStore.InsertNode(ctx, st, blkRoot))

	opt, err := c.IsOptimisticForRoot(ctx, [32]byte{'a'})
	require.NoError(t, err)
	require.Equal(t, true, opt)
}

func TestService_IsOptimisticForRoot_DB_ProtoArray(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, b)
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, optimisticBlock)

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, validatedBlock)

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
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: doublylinkedtree.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, b)
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, optimisticBlock)

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, validatedBlock)

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
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: doublylinkedtree.New()}, head: &head{slot: 101, root: [32]byte{'b'}}}
	c.head = &head{root: params.BeaconConfig().ZeroHash}
	b := util.NewBeaconBlock()
	b.Block.Slot = 10
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, b)
	require.NoError(t, beaconDB.SaveStateSummary(context.Background(), &ethpb.StateSummary{Root: br[:], Slot: 10}))

	optimisticBlock := util.NewBeaconBlock()
	optimisticBlock.Block.Slot = 97
	optimisticRoot, err := optimisticBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, optimisticBlock)

	validatedBlock := util.NewBeaconBlock()
	validatedBlock.Block.Slot = 9
	validatedRoot, err := validatedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, context.Background(), beaconDB, validatedBlock)

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

func TestService_IsFinalized(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	c := &Service{cfg: &config{BeaconDB: beaconDB, ForkChoiceStore: doublylinkedtree.New()}}
	r1 := [32]byte{'a'}
	require.NoError(t, c.ForkChoiceStore().UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{
		Root: r1,
	}))
	b := util.NewBeaconBlock()
	br, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	util.SaveBlock(t, ctx, beaconDB, b)
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: br[:], Slot: 10}))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, br))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{
		Root: br[:],
	}))
	require.Equal(t, true, c.IsFinalized(ctx, r1))
	require.Equal(t, true, c.IsFinalized(ctx, br))
	require.Equal(t, false, c.IsFinalized(ctx, [32]byte{'c'}))
}
