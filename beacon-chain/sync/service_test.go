package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	gcache "github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/async/abool"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state-proto/v1"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
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
	r := Service{
		ctx: context.Background(),
		cfg: &config{
			p2p:           p2p,
			chain:         chainService,
			stateNotifier: chainService.StateNotifier(),
			initialSync:   &mockSync.Sync{IsSyncing: false},
		},
		chainStarted: abool.New(),
		subHandler:   newSubTopicHandler(),
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
	p2p.Digest, err = r.currentForkDigest()
	r.cfg.blockNotifier = chainService.BlockNotifier()
	blockChan := make(chan feed.Event, 1)
	sub := r.cfg.blockNotifier.BlockFeed().Subscribe(blockChan)

	require.NoError(t, err)
	p2p.ReceivePubSub(topic, msg)

	// wait for chainstart to be sent
	time.Sleep(2 * time.Second)
	require.Equal(t, true, r.chainStarted.IsSet(), "Did not receive chain start event.")

	assert.Equal(t, 0, len(blockChan), "block was received by sync service despite not being fully synced")

	i = r.cfg.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Synced,
		Data: &statefeed.SyncedData{
			StartTime: time.Now(),
		},
	})

	if i == 0 {
		t.Fatal("didn't send genesis time to sync event subscribers")
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		// Wait for block to be received by service.
		<-blockChan
		wg.Done()
		sub.Unsubscribe()
	}()

	p2p.ReceivePubSub(topic, msg)
	// wait for message to be sent
	util.WaitTimeout(wg, 2*time.Second)
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
