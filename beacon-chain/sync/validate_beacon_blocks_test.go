package sync

import (
	"bytes"
	"context"
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
	if err != nil {
		t.Fatal(err)
	}
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
		blockNotifier:  chainService.BlockNotifier(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept

	if result {
		t.Error("Expected false result, got true")
	}
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
	if err := db.SaveBlock(context.Background(), msg); err != nil {
		t.Fatal(err)
	}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{Genesis: time.Now()}
	r := &Service{
		db:                db,
		p2p:               p,
		initialSync:       &mockSync.Sync{IsSyncing: false},
		chain:             chainService,
		seenBlockCache:    c,
		stateSummaryCache: cache.NewStateSummaryCache(),
		blockNotifier:     chainService.BlockNotifier(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}

	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	if result {
		t.Error("Expected false result, got true")
	}
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
	if err := db.SaveBlock(ctx, parentBlock); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveStateSummary(ctx, &pb.StateSummary{
		Root: bRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	copied := beaconState.Copy()
	if err := copied.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	if err != nil {
		t.Error(err)
	}
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
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
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	if !result {
		t.Error("Expected true result, got false")
	}

	if m.ValidatorData == nil {
		t.Error("Decoded message was not set on the message validator data")
	}
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
	if err := db.SaveBlock(ctx, parentBlock); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveStateSummary(ctx, &pb.StateSummary{
		Root: bRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	copied := beaconState.Copy()
	// The next block is at least 2 epochs ahead to induce shuffling and a new seed.
	blkSlot := params.BeaconConfig().SlotsPerEpoch * 2
	copied, err = state.ProcessSlots(context.Background(), copied, blkSlot)
	if err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(copied)
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          blkSlot,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	if err != nil {
		t.Error(err)
	}
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
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
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   stateSummaryCache,
		stateGen:            stateGen,
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	if !result {
		t.Error("Expected true result, got false")
	}

	if m.ValidatorData == nil {
		t.Error("Decoded message was not set on the message validator data")
	}
}

func TestValidateBeaconBlockPubSub_Syncing(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromFuture(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       1000,
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	chainService := &mock.ChainService{Genesis: time.Now()}
	r := &Service{
		p2p:                 p,
		db:                  db,
		initialSync:         &mockSync.Sync{IsSyncing: false},
		chain:               chainService,
		blockNotifier:       chainService.BlockNotifier(),
		seenBlockCache:      c,
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept
	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_RejectBlocksFromThePast(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
			Slot:       10,
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}

	genesisTime := time.Now()
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
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
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(msg)],
			},
		},
	}
	result := r.validateBeaconBlockPubSub(ctx, "", m) == pubsub.ValidationAccept

	if result {
		t.Error("Expected false result, got true")
	}
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
	if err := db.SaveBlock(ctx, parentBlock); err != nil {
		t.Fatal(err)
	}
	bRoot, err := stateutil.BlockRoot(parentBlock.Block)
	if err := db.SaveState(ctx, beaconState, bRoot); err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ProposerIndex: proposerIdx,
			Slot:          1,
			ParentRoot:    bRoot[:],
		},
	}

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(msg.Block, domain)
	if err != nil {
		t.Error(err)
	}
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()
	msg.Signature = blockSig[:]

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
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
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		stateSummaryCache:   cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		t.Fatal(err)
	}
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
	if result {
		t.Error("Expected false result, got true")
	}
}

func TestValidateBeaconBlockPubSub_FilterByFinalizedEpoch(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	parent := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := stateutil.BlockRoot(parent.Block)
	if err != nil {
		t.Fatal(err)
	}
	chain := &mock.ChainService{Genesis: time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0),
		FinalizedCheckPoint: &ethpb.Checkpoint{
			Epoch: 1,
		}}
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		db:             db,
		p2p:            p,
		chain:          chain,
		blockNotifier:  chain.BlockNotifier(),
		attPool:        attestations.NewPool(),
		seenBlockCache: c,
		initialSync:    &mockSync.Sync{IsSyncing: false},
	}

	b := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: parentRoot[:], Body: &ethpb.BeaconBlockBody{}},
	}
	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, b); err != nil {
		t.Fatal(err)
	}
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
	if _, err := p.Encoding().Encode(buf, b); err != nil {
		t.Fatal(err)
	}
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
