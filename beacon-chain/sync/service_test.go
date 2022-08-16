package sync

import (
	"context"
	"testing"
	"time"

	gcache "github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/v3/async/abool"
	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestService_StatusZeroEpoch(t *testing.T) {
	bState, err := v1.InitializeFromProto(&ethpb.BeaconState{Slot: 0})
	require.NoError(t, err)
	r := &Service{
		cfg: &config{
			p2p:         p2ptest.NewTestP2P(t),
			initialSync: new(mockSync.Sync),
			chain: &mockChain.ChainService{
				Genesis: time.Now(),
				State:   bState,
			},
		},
		chainStarted: abool.New(),
	}
	r.chainStarted.Set()

	assert.NoError(t, r.Status(), "Wanted non failing status")
}

func TestSyncHandlers_WaitToSync(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	r := Service{
		ctx: context.Background(),
		cfg: &config{
			p2p:           p2p,
			chain:         chainService,
			stateNotifier: chainService.StateNotifier(),
			initialSync:   &mockSync.Sync{IsSyncing: false},
		},
		chainStarted: abool.New(),
	}

	topic := "/eth2/%x/beacon_block"
	go r.registerHandlers()
	time.Sleep(100 * time.Millisecond)
	i := r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now(),
		},
	})
	if i == 0 {
		t.Fatal("didn't send genesis time to subscribers")
	}
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)

	msg := util.NewBeaconBlock()
	msg.Block.ParentRoot = util.Random32Bytes(t)
	msg.Signature = sk.Sign([]byte("data")).Marshal()
	p2p.ReceivePubSub(topic, msg)
	// wait for chainstart to be sent
	time.Sleep(400 * time.Millisecond)
	require.Equal(t, true, r.chainStarted.IsSet(), "Did not receive chain start event.")
}

func TestSyncHandlers_WaitForChainStart(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	r := Service{
		ctx: context.Background(),
		cfg: &config{
			p2p:           p2p,
			chain:         chainService,
			stateNotifier: chainService.StateNotifier(),
			initialSync:   &mockSync.Sync{IsSyncing: false},
		},
		chainStarted:        abool.New(),
		slotToPendingBlocks: gcache.New(time.Second, 2*time.Second),
	}

	go r.registerHandlers()
	time.Sleep(100 * time.Millisecond)
	i := r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now().Add(2 * time.Second),
		},
	})
	if i == 0 {
		t.Fatal("didn't send genesis time to subscribers")
	}
	require.Equal(t, false, r.chainStarted.IsSet(), "Chainstart was marked prematurely")

	// wait for chainstart to be sent
	time.Sleep(3 * time.Second)
	require.Equal(t, true, r.chainStarted.IsSet(), "Did not receive chain start event.")
}

func TestSyncHandlers_WaitTillSynced(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r := Service{
		ctx: ctx,
		cfg: &config{
			p2p:           p2p,
			beaconDB:      dbTest.SetupDB(t),
			chain:         chainService,
			stateNotifier: chainService.StateNotifier(),
			blockNotifier: chainService.BlockNotifier(),
			initialSync:   &mockSync.Sync{IsSyncing: false},
		},
		chainStarted: abool.New(),
		subHandler:   newSubTopicHandler(),
	}
	r.initCaches()

	syncCompleteCh := make(chan bool)
	go func() {
		r.registerHandlers()
		syncCompleteCh <- true
	}()
	for i := 0; i == 0; {
		assert.NoError(t, ctx.Err())
		i = r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime: time.Now(),
			},
		})
	}
	for !r.chainStarted.IsSet() {
		assert.NoError(t, ctx.Err())
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, true, r.chainStarted.IsSet(), "Did not receive chain start event.")

	blockChan := make(chan *feed.Event, 1)
	sub := r.cfg.blockNotifier.BlockFeed().Subscribe(blockChan)
	defer sub.Unsubscribe()

	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	require.NoError(t, err)
	msg := util.NewBeaconBlock()
	msg.Block.ParentRoot = util.Random32Bytes(t)
	msg.Signature = sk.Sign([]byte("data")).Marshal()
	p2p.Digest, err = r.currentForkDigest()
	require.NoError(t, err)

	// Save block into DB so that validateBeaconBlockPubSub() process gets short cut.
	util.SaveBlock(t, ctx, r.cfg.beaconDB, msg)

	topic := "/eth2/%x/beacon_block"
	p2p.ReceivePubSub(topic, msg)
	assert.Equal(t, 0, len(blockChan), "block was received by sync service despite not being fully synced")

	for i := 0; i == 0; {
		assert.NoError(t, ctx.Err())
		i = r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Synced,
			Data: &statefeed.SyncedData{
				StartTime: time.Now(),
			},
		})
	}
	<-syncCompleteCh

	p2p.ReceivePubSub(topic, msg)

	select {
	case <-blockChan:
	case <-ctx.Done():
	}
	assert.NoError(t, ctx.Err())
}

func TestSyncService_StopCleanly(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := Service{
		ctx:    ctx,
		cancel: cancel,
		cfg: &config{
			p2p:           p2p,
			chain:         chainService,
			stateNotifier: chainService.StateNotifier(),
			initialSync:   &mockSync.Sync{IsSyncing: false},
		},
		chainStarted: abool.New(),
		subHandler:   newSubTopicHandler(),
	}

	go r.registerHandlers()
	time.Sleep(100 * time.Millisecond)
	i := r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now(),
		},
	})
	if i == 0 {
		t.Fatal("didn't send genesis time to subscribers")
	}

	var err error
	p2p.Digest, err = r.currentForkDigest()
	require.NoError(t, err)

	// wait for chainstart to be sent
	time.Sleep(2 * time.Second)
	require.Equal(t, true, r.chainStarted.IsSet(), "Did not receive chain start event.")

	i = r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Synced,
		Data: &statefeed.SyncedData{
			StartTime: time.Now(),
		},
	})
	if i == 0 {
		t.Fatal("didn't send genesis time to sync event subscribers")
	}

	time.Sleep(1 * time.Second)

	require.NotEqual(t, 0, len(r.cfg.p2p.PubSub().GetTopics()))
	require.NotEqual(t, 0, len(r.cfg.p2p.Host().Mux().Protocols()))

	// Both pubsub and rpc topcis should be unsubscribed.
	require.NoError(t, r.Stop())

	// Sleep to allow pubsub topics to be deregistered.
	time.Sleep(1 * time.Second)
	require.Equal(t, 0, len(r.cfg.p2p.PubSub().GetTopics()))
	require.Equal(t, 0, len(r.cfg.p2p.Host().Mux().Protocols()))
}
