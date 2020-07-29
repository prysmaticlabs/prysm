package sync

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// General note for writing validation tests: Use a random value for any field
// on the beacon block to avoid hitting shared global cache conditions across
// tests in this package.

func TestValidateBeaconBlockPubSub_InvalidSignature(t *testing.T) {
	ctx := context.Background()
	db, _ := dbtest.SetupDB(t)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       1,
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: bytesutil.PadTo([]byte("fake"), 96),
	}

	p := p2ptest.NewTestP2P(t)

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	chainService := &mock.ChainService{Genesis: time.Now(),
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:             db,
		p2p:            p,
		initialSync:    &mockSync.Sync{IsSyncing: false},
		chain:          chainService,
		seenBlockCache: c,
		badBlockCache:  c2,
		blockNotifier:  chainService.BlockNotifier(),
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_BlockAlreadyPresentInDB(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	ctx := context.Background()

	p := p2ptest.NewTestP2P(t)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			Slot:       100,
			ParentRoot: testutil.Random32Bytes(t),
		},
	}
	require.NoError(t, db.SaveBlock(context.Background(), msg))

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	chainService := &mock.ChainService{Genesis: time.Now()}
	r := &Service{
		db:                db,
		p2p:               p,
		initialSync:       &mockSync.Sync{IsSyncing: false},
		chain:             chainService,
		seenBlockCache:    c,
		badBlockCache:     c2,
		stateSummaryCache: cache.NewStateSummaryCache(),
		blockNotifier:     chainService.BlockNotifier(),
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)

	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_ValidProposerSignature(t *testing.T) {
	db, stateSummaryCache := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	parentBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 0,
			Slot:          0,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, parentBlock))
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:]}))
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(1))
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	stateGen := stategen.New(db, stateSummaryCache)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		State: beaconState,
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:                  db,
		p2p:                 p,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, true, result)
	assert.NotNil(t, m.ValidatorData, "Decoded message was not set on the message validator data")
}

func TestValidateBeaconBlockPubSub_AdvanceEpochsForState(t *testing.T) {
	db, stateSummaryCache := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	parentBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 0,
			Slot:          0,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, parentBlock))
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:]}))
	copied := beaconState.Copy()
	// The next block is at least 2 epochs ahead to induce shuffling and a new seed.
	blkSlot := params.BeaconConfig().SlotsPerEpoch * 2
	copied, err = state.ProcessSlots(context.Background(), copied, blkSlot)
	require.NoError(t, err)
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          blkSlot,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	stateGen := stategen.New(db, stateSummaryCache)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(blkSlot*params.BeaconConfig().SecondsPerSlot), 0),
		State: beaconState,
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:                  db,
		p2p:                 p,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, true, result)
	assert.NotNil(t, m.ValidatorData, "Decoded message was not set on the message validator data")
}

func TestValidateBeaconBlockPubSub_Syncing(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}
	chainService := &mock.ChainService{
		Genesis: time.Now(),
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:            db,
		p2p:           p,
		initialSync:   &mockSync.Sync{IsSyncing: true},
		chain:         chainService,
		blockNotifier: chainService.BlockNotifier(),
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromFuture(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       1000,
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	chainService := &mock.ChainService{Genesis: time.Now()}
	r := &Service{
		p2p:                 p,
		db:                  db,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromThePast(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	sk, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       10,
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}

	genesisTime := time.Now()
	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	chainService := &mock.ChainService{
		Genesis: time.Unix(genesisTime.Unix()-1000, 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 1,
		}}
	r := &Service{
		db:             db,
		p2p:            p,
		initialSync:    &mockSync.Sync{IsSyncing: false},
		chain:          chainService,
		blockNotifier:  chainService.BlockNotifier(),
		seenBlockCache: c,
		badBlockCache:  c2,
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_SeenProposerSlot(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	parentBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 0,
			Slot:          0,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, parentBlock))
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, bRoot))
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	require.NoError(t, err)

	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		State: beaconState,
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:                  db,
		p2p:                 p,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	r.setSeenBlockIndexSlot(msg.Block.Slot, msg.Block.ProposerIndex)
	time.Sleep(10 * time.Millisecond) // Wait for cached value to pass through buffers.
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}

func TestValidateBeaconBlockPubSub_FilterByFinalizedEpoch(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	parent := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	require.NoError(t, db.SaveBlock(context.Background(), parent))
	parentRoot, err := stateutil.BlockRoot(parent.Block)
	require.NoError(t, err)
	chain := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 1,
		}}
	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		db:             db,
		p2p:            p,
		chain:          chain,
		blockNotifier:  chain.BlockNotifier(),
		attPool:        attestations.NewPool(),
		seenBlockCache: c,
		badBlockCache:  c2,
		initialSync:    &mockSync.Sync{IsSyncing: false},
	}

	b := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: parentRoot[:], Body: &ethpb.BeaconBlockBody{}},
	}
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(b)],
			},
		},
	}

	r.validateBeaconBlockPubSub(context.Background(), "", m)
	testutil.AssertLogsContain(t, hook, "Block slot older/equal than last finalized epoch start slot, rejecting it")

	hook.Reset()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	buf = new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, b)
	require.NoError(t, err)
	m = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(b)],
			},
		},
	}

	r.validateBeaconBlockPubSub(context.Background(), "", m)
	testutil.AssertLogsDoNotContain(t, hook, "Block slot older/equal than last finalized epoch start slot, rejecting itt")
}

func TestValidateBeaconBlockPubSub_ParentNotFinalizedDescendant(t *testing.T) {
	hook := logTest.NewGlobal()
	db, stateSummaryCache := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	parentBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 0,
			Slot:          0,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, parentBlock))
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:]}))
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(1))
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	stateGen := stategen.New(db, stateSummaryCache)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		State: beaconState,
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		},
		VerifyBlkDescendantErr: errors.New("not part of finalized chain"),
	}
	r := &Service{
		db:                  db,
		p2p:                 p,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	assert.Equal(t, pubsub.ValidationReject, r.validateBeaconBlockPubSub(ctx, "", m), "Wrong validation result returned")
	testutil.AssertLogsContain(t, hook, "not part of finalized chain")
}

func TestValidateBeaconBlockPubSub_InvalidParentBlock(t *testing.T) {
	db, stateSummaryCache := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	parentBlock := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: 0,
			Slot:          0,
		},
	}
	require.NoError(t, db.SaveBlock(ctx, parentBlock))
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, bRoot))
	require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: bRoot[:]}))
	copied := beaconState.Copy()
	require.NoError(t, copied.SetSlot(1))
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	require.NoError(t, err)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	// Mutate Signature
	copy(blockSig[:4], []byte{1, 2, 3, 4})
	msg.Signature = blockSig

	currBlockRoot, err := stateutil.BlockRoot(msg.Block)
	require.NoError(t, err)

	c, err := lru.New(10)
	require.NoError(t, err)
	c2, err := lru.New(10)
	require.NoError(t, err)
	stateGen := stategen.New(db, stateSummaryCache)
	chainService := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		State: beaconState,
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 0,
		}}
	r := &Service{
		db:                  db,
		p2p:                 p,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		badBlockCache:       c2,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)

	require.NoError(t, copied.SetSlot(2))
	proposerIdx, err = helpers.BeaconProposerIndex(copied)
	require.NoError(t, err)

	msg = &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          2,
			ParentRoot:    currBlockRoot[:],
		},
	}

	buf = new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, msg)
	require.NoError(t, err)
	m = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}

	// Expect block with bad parent to fail too
	result = r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, result)
}
