package sync

import (
	"bytes"
	"context"
	"testing"
	"time"

	mock "github.com/OffchainLabs/prysm/v7/beacon-chain/blockchain/testing"
	dbtest "github.com/OffchainLabs/prysm/v7/beacon-chain/db/testing"
	doublylinkedtree "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/doubly-linked-tree"
	p2ptesting "github.com/OffchainLabs/prysm/v7/beacon-chain/p2p/testing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/startup"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stategen"
	lruwrpr "github.com/OffchainLabs/prysm/v7/cache/lru"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
	"github.com/pkg/errors"
)

func TestProcessPendingPayloadEnvelope_NoPendingEnvelope(t *testing.T) {
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg:                      &config{chain: &mock.ChainService{}},
	}
	root := [32]byte{0x01}
	s.processPendingPayloadEnvelope(context.Background(), root)
}

func TestProcessPendingPayloadEnvelope_AlreadySeen(t *testing.T) {
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	chainService := &mock.ChainService{
		Genesis:             time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{},
		DB:                  db,
	}
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg:                      &config{chain: chainService, beaconDB: db},
	}

	bid := util.GenerateTestSignedExecutionPayloadBid(1)
	sb := util.NewBeaconBlockGloas()
	sb.Block.Slot = 1
	sb.Block.Body.SignedExecutionPayloadBid = bid
	signedBlock, err := blocks.NewSignedBeaconBlock(sb)
	require.NoError(t, err)
	root, err := signedBlock.Block().HashTreeRoot()
	require.NoError(t, err)

	builderIdx := primitives.BuilderIndex(bid.Message.BuilderIndex)
	blockHash := bytesutil.ToBytes32(bid.Message.BlockHash)
	env := testSignedExecutionPayloadEnvelope(t, 1, builderIdx, root, blockHash)
	s.pendingPayloadEnvelopes[root] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{uint64(builderIdx): env}

	s.setSeenPayloadEnvelope(root, builderIdx)
	s.processPendingPayloadEnvelope(ctx, root)
	require.Equal(t, 0, len(s.pendingPayloadEnvelopes))
}

func TestProcessPendingPayloadEnvelope_HappyPath(t *testing.T) {
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	chainService := &mock.ChainService{
		Genesis:             time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{},
		DB:                  db,
	}
	stateGen := stategen.New(db, doublylinkedtree.New())
	broadcaster := p2ptesting.NewTestP2P(t)
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg: &config{
			chain:    chainService,
			beaconDB: db,
			stateGen: stateGen,
			clock:    startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot),
			p2p:      broadcaster,
		},
	}

	bid := util.GenerateTestSignedExecutionPayloadBid(1)
	sb := util.NewBeaconBlockGloas()
	sb.Block.Slot = 1
	sb.Block.Body.SignedExecutionPayloadBid = bid
	signedBlock, err := blocks.NewSignedBeaconBlock(sb)
	require.NoError(t, err)
	root, err := signedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBlock))

	st, err := util.NewBeaconStateFulu()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, root))

	builderIdx := primitives.BuilderIndex(bid.Message.BuilderIndex)
	blockHash := bytesutil.ToBytes32(bid.Message.BlockHash)
	env := testSignedExecutionPayloadEnvelope(t, 1, builderIdx, root, blockHash)
	s.pendingPayloadEnvelopes[root] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{uint64(builderIdx): env}

	require.Equal(t, false, s.hasSeenPayloadEnvelope(root, builderIdx))
	s.processPendingPayloadEnvelope(ctx, root)
	require.Equal(t, 0, len(s.pendingPayloadEnvelopes))
	require.Equal(t, true, s.hasSeenPayloadEnvelope(root, builderIdx))
	// EIP-8411: the drain no longer re-broadcasts; chunks already propagated.
	require.Equal(t, false, broadcaster.BroadcastCalled.Load())
	require.Equal(t, true, chainService.ReceivePayloadEnvelopeCtxHadDeadline)
}

func TestProcessPendingPayloadEnvelope_DoesNotBroadcastOnReceiveError(t *testing.T) {
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	chainService := &mock.ChainService{
		Genesis:                   time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint:       &ethpb.Checkpoint{},
		DB:                        db,
		ReceivePayloadEnvelopeErr: errors.New("receive failed"),
	}
	stateGen := stategen.New(db, doublylinkedtree.New())
	broadcaster := p2ptesting.NewTestP2P(t)
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg: &config{
			chain:    chainService,
			beaconDB: db,
			stateGen: stateGen,
			clock:    startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot),
			p2p:      broadcaster,
		},
	}

	bid := util.GenerateTestSignedExecutionPayloadBid(1)
	sb := util.NewBeaconBlockGloas()
	sb.Block.Slot = 1
	sb.Block.Body.SignedExecutionPayloadBid = bid
	signedBlock, err := blocks.NewSignedBeaconBlock(sb)
	require.NoError(t, err)
	root, err := signedBlock.Block().HashTreeRoot()
	require.NoError(t, err)

	builderIdx := primitives.BuilderIndex(bid.Message.BuilderIndex)
	blockHash := bytesutil.ToBytes32(bid.Message.BlockHash)
	env := testSignedExecutionPayloadEnvelope(t, 1, builderIdx, root, blockHash)
	s.pendingPayloadEnvelopes[root] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{uint64(builderIdx): env}

	s.processPendingPayloadEnvelope(ctx, root)
	require.Equal(t, false, broadcaster.BroadcastCalled.Load())
	require.Equal(t, false, s.hasSeenPayloadEnvelope(root, builderIdx))
}

func TestProcessPendingPayloadEnvelopes_Sweep(t *testing.T) {
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	chainService := &mock.ChainService{
		Genesis:             time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{},
		DB:                  db,
	}
	stateGen := stategen.New(db, doublylinkedtree.New())
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg: &config{
			chain:    chainService,
			beaconDB: db,
			stateGen: stateGen,
			clock:    startup.NewClock(chainService.Genesis, chainService.ValidatorsRoot),
			p2p:      p2ptesting.NewTestP2P(t),
		},
	}

	bid := util.GenerateTestSignedExecutionPayloadBid(1)
	sb := util.NewBeaconBlockGloas()
	sb.Block.Slot = 1
	sb.Block.Body.SignedExecutionPayloadBid = bid
	signedBlock, err := blocks.NewSignedBeaconBlock(sb)
	require.NoError(t, err)
	root, err := signedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, signedBlock))

	st, err := util.NewBeaconStateFulu()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, root))

	builderIdx := primitives.BuilderIndex(bid.Message.BuilderIndex)
	blockHash := bytesutil.ToBytes32(bid.Message.BlockHash)
	env := testSignedExecutionPayloadEnvelope(t, 1, builderIdx, root, blockHash)
	s.pendingPayloadEnvelopes[root] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{uint64(builderIdx): env}

	require.Equal(t, false, s.hasSeenPayloadEnvelope(root, builderIdx))

	s.processPendingPayloadEnvelopes(ctx)
	require.Equal(t, 0, len(s.pendingPayloadEnvelopes))
	require.Equal(t, true, s.hasSeenPayloadEnvelope(root, builderIdx))
}

func TestProcessPendingPayloadEnvelopes_SkipsUnknownRoot(t *testing.T) {
	ctx := context.Background()
	db := dbtest.SetupDB(t)
	chainService := &mock.ChainService{
		Genesis:             time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{},
		DB:                  db,
		NotFinalized:        true, // InForkchoice returns false
	}
	s := &Service{
		pendingPayloadEnvelopes:  make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		seenPayloadEnvelopeCache: lruwrpr.New(10),
		badBlockCache:            lruwrpr.New(10),
		cfg:                      &config{chain: chainService, beaconDB: db},
	}

	root := [32]byte{0x01}
	blockHash := [32]byte{0x02}
	env := testSignedExecutionPayloadEnvelope(t, 1, 1, root, blockHash)
	s.pendingPayloadEnvelopes[root] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{1: env}

	s.processPendingPayloadEnvelopes(ctx)
	require.Equal(t, 1, len(s.pendingPayloadEnvelopes))
}

func TestPrunePendingPayloadEnvelopes(t *testing.T) {
	finalizedEpoch := primitives.Epoch(3)
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	s := &Service{
		pendingPayloadEnvelopes: make(map[[32]byte]map[uint64]*ethpb.SignedExecutionPayloadEnvelope),
		cfg: &config{
			chain: &mock.ChainService{
				FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: finalizedEpoch},
			},
		},
	}

	oldRoot := [32]byte{0x01}
	oldEnv := &ethpb.SignedExecutionPayloadEnvelope{
		Message: &ethpb.ExecutionPayloadEnvelope{
			Payload: &enginev1.ExecutionPayloadGloas{SlotNumber: primitives.Slot(finalizedEpoch-1) * slotsPerEpoch},
		},
		Signature: bytes.Repeat([]byte{0xAA}, 96),
	}

	atFinalizedRoot := [32]byte{0x03}
	atFinalizedEnv := &ethpb.SignedExecutionPayloadEnvelope{
		Message: &ethpb.ExecutionPayloadEnvelope{
			Payload: &enginev1.ExecutionPayloadGloas{SlotNumber: primitives.Slot(finalizedEpoch) * slotsPerEpoch},
		},
		Signature: bytes.Repeat([]byte{0xCC}, 96),
	}

	freshRoot := [32]byte{0x02}
	freshEnv := &ethpb.SignedExecutionPayloadEnvelope{
		Message: &ethpb.ExecutionPayloadEnvelope{
			Payload: &enginev1.ExecutionPayloadGloas{SlotNumber: primitives.Slot(finalizedEpoch+1) * slotsPerEpoch},
		},
		Signature: bytes.Repeat([]byte{0xBB}, 96),
	}

	s.pendingPayloadEnvelopes[oldRoot] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{1: oldEnv}
	s.pendingPayloadEnvelopes[atFinalizedRoot] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{1: atFinalizedEnv}
	s.pendingPayloadEnvelopes[freshRoot] = map[uint64]*ethpb.SignedExecutionPayloadEnvelope{1: freshEnv}
	require.Equal(t, 3, len(s.pendingPayloadEnvelopes))

	s.prunePendingPayloadEnvelopes()

	require.Equal(t, 2, len(s.pendingPayloadEnvelopes))
	_, ok := s.pendingPayloadEnvelopes[oldRoot]
	require.Equal(t, false, ok)
	_, ok = s.pendingPayloadEnvelopes[atFinalizedRoot]
	require.Equal(t, true, ok)
	_, ok = s.pendingPayloadEnvelopes[freshRoot]
	require.Equal(t, true, ok)
}

func proposerLookaheadWith(t *testing.T, proposer primitives.ValidatorIndex) []primitives.ValidatorIndex {
	t.Helper()
	size := int(params.BeaconConfig().MinSeedLookahead+1) * int(params.BeaconConfig().SlotsPerEpoch)
	la := make([]primitives.ValidatorIndex, size)
	for i := range la {
		la[i] = proposer
	}
	return la
}

func testSignedExecutionPayloadEnvelope(t *testing.T, slot primitives.Slot, builderIdx primitives.BuilderIndex, _, blockHash [32]byte) *ethpb.SignedExecutionPayloadEnvelope {
	t.Helper()

	payload := &enginev1.ExecutionPayloadGloas{
		ParentHash:    bytes.Repeat([]byte{0x01}, 32),
		FeeRecipient:  bytes.Repeat([]byte{0x02}, 20),
		StateRoot:     bytes.Repeat([]byte{0x03}, 32),
		ReceiptsRoot:  bytes.Repeat([]byte{0x04}, 32),
		LogsBloom:     bytes.Repeat([]byte{0x05}, 256),
		PrevRandao:    bytes.Repeat([]byte{0x06}, 32),
		BlockNumber:   1,
		GasLimit:      2,
		GasUsed:       3,
		Timestamp:     4,
		BaseFeePerGas: bytes.Repeat([]byte{0x07}, 32),
		BlockHash:     blockHash[:],
		Transactions:  &enginev1.ProgressiveTransactionList{},
		Withdrawals:   []*enginev1.Withdrawal{},
		BlobGasUsed:   0,
		ExcessBlobGas: 0,
		SlotNumber:    slot,
	}

	return &ethpb.SignedExecutionPayloadEnvelope{
		Message: &ethpb.ExecutionPayloadEnvelope{
			Payload: payload,
			ExecutionRequests: &enginev1.ExecutionRequestsGloas{
				Deposits: []*enginev1.DepositRequest{},
			},
			BuilderIndex:          builderIdx,
			ParentBeaconBlockRoot: make([]byte, 32),
		},
		Signature: bytes.Repeat([]byte{0xAA}, 96),
	}
}
