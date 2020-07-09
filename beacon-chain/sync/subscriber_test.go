package sync

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubscribe_ReceivesValidMessage(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := Service{
		ctx:         context.Background(),
		p2p:         p2p,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mockChain.ChainService{
			ValidatorsRoot: [32]byte{'A'},
			Genesis:        time.Now(),
		},
	}
	var err error
	p2p.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	topic := "/eth2/%x/voluntary_exit"
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, r.noopValidator, func(_ context.Context, msg proto.Message) error {
		m, ok := msg.(*pb.SignedVoluntaryExit)
		if !ok {
			t.Error("Object is not of type *pb.SignedVoluntaryExit")
		}
		if m.Exit == nil || m.Exit.Epoch != 55 {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()
		return nil
	})
	r.chainStarted = true

	p2p.ReceivePubSub(topic, &pb.SignedVoluntaryExit{Exit: &pb.VoluntaryExit{Epoch: 55}})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestSubscribe_ReceivesAttesterSlashing(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	d, _ := db.SetupDB(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := Service{
		ctx:                       ctx,
		p2p:                       p2p,
		initialSync:               &mockSync.Sync{IsSyncing: false},
		slashingPool:              slashings.NewPool(),
		chain:                     chainService,
		db:                        d,
		seenAttesterSlashingCache: c,
	}
	topic := "/eth2/%x/attester_slashing"
	var wg sync.WaitGroup
	wg.Add(1)
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	r.subscribe(topic, r.noopValidator, func(ctx context.Context, msg proto.Message) error {
		if err := r.attesterSlashingSubscriber(ctx, msg); err != nil {
			t.Fatal(err)
		}
		wg.Done()
		return nil
	})
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	chainService.State = beaconState
	r.chainStarted = true
	attesterSlashing, err := testutil.GenerateAttesterSlashingForValidator(
		beaconState,
		privKeys[1],
		1, /* validator index */
	)
	if err != nil {
		t.Fatalf("Error generating attester slashing")
	}
	err = r.db.SaveState(ctx, beaconState, bytesutil.ToBytes32(attesterSlashing.Attestation_1.Data.BeaconBlockRoot))
	if err != nil {
		t.Fatal(err)
	}
	p2p.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	p2p.ReceivePubSub(topic, attesterSlashing)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
	as := r.slashingPool.PendingAttesterSlashings(ctx, beaconState)
	if len(as) != 1 {
		t.Errorf("Expected attester slashing: %v to be added to slashing pool. got: %v", attesterSlashing, as[0])
	}
}

func TestSubscribe_ReceivesProposerSlashing(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	chainService := &mockChain.ChainService{
		ValidatorsRoot: [32]byte{'A'},
		Genesis:        time.Now(),
	}
	d, _ := db.SetupDB(t)
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := Service{
		ctx:                       ctx,
		p2p:                       p2p,
		initialSync:               &mockSync.Sync{IsSyncing: false},
		slashingPool:              slashings.NewPool(),
		chain:                     chainService,
		db:                        d,
		seenProposerSlashingCache: c,
	}
	topic := "/eth2/%x/proposer_slashing"
	var wg sync.WaitGroup
	wg.Add(1)
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	r.subscribe(topic, r.noopValidator, func(ctx context.Context, msg proto.Message) error {
		if err := r.proposerSlashingSubscriber(ctx, msg); err != nil {
			t.Fatal(err)
		}
		wg.Done()
		return nil
	})
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	chainService.State = beaconState
	r.chainStarted = true
	proposerSlashing, err := testutil.GenerateProposerSlashingForValidator(
		beaconState,
		privKeys[1],
		1, /* validator index */
	)
	if err != nil {
		t.Fatalf("Error generating proposer slashing")
	}
	p2p.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	p2p.ReceivePubSub(topic, proposerSlashing)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
	ps := r.slashingPool.PendingProposerSlashings(ctx, beaconState)
	if len(ps) != 1 {
		t.Errorf("Expected proposer slashing: %v to be added to slashing pool. got: %v", proposerSlashing, ps)
	}
}

func TestSubscribe_HandlesPanic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	r := Service{
		ctx: context.Background(),
		chain: &mockChain.ChainService{
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		p2p: p,
	}
	var err error
	p.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.SignedVoluntaryExit{})]
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, r.noopValidator, func(_ context.Context, msg proto.Message) error {
		defer wg.Done()
		panic("bad")
	})
	r.chainStarted = true
	p.ReceivePubSub(topic, &pb.SignedVoluntaryExit{Exit: &pb.VoluntaryExit{Epoch: 55}})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestRevalidateSubscription_CorrectlyFormatsTopic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	hook := logTest.NewGlobal()
	r := Service{
		ctx: context.Background(),
		chain: &mockChain.ChainService{
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		p2p: p,
	}
	digest, err := r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	subscriptions := make(map[uint64]*pubsub.Subscription, params.BeaconConfig().MaxCommitteesPerSlot)

	defaultTopic := "/eth2/testing/%#x/committee%d"
	// committee index 1
	fullTopic := fmt.Sprintf(defaultTopic, digest, 1) + r.p2p.Encoding().ProtocolSuffix()
	err = r.p2p.PubSub().RegisterTopicValidator(fullTopic, r.noopValidator)
	if err != nil {
		t.Fatal(err)
	}
	subscriptions[1], err = r.p2p.PubSub().Subscribe(fullTopic)
	if err != nil {
		t.Fatal(err)
	}

	// committee index 2
	fullTopic = fmt.Sprintf(defaultTopic, digest, 2) + r.p2p.Encoding().ProtocolSuffix()
	err = r.p2p.PubSub().RegisterTopicValidator(fullTopic, r.noopValidator)
	if err != nil {
		t.Fatal(err)
	}
	subscriptions[2], err = r.p2p.PubSub().Subscribe(fullTopic)
	if err != nil {
		t.Fatal(err)
	}

	r.reValidateSubscriptions(subscriptions, []uint64{2}, defaultTopic, digest)
	testutil.AssertLogsDoNotContain(t, hook, "Failed to unregister topic validator")
}

func TestStaticSubnets(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx, cancel := context.WithCancel(context.Background())
	r := Service{
		ctx: ctx,
		chain: &mockChain.ChainService{
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		p2p: p,
	}
	defaultTopic := "/eth2/%x/beacon_attestation_%d"
	r.subscribeStaticWithSubnets(defaultTopic, r.noopValidator, func(_ context.Context, msg proto.Message) error {
		// no-op
		return nil
	})
	topics := r.p2p.PubSub().GetTopics()
	if uint64(len(topics)) != params.BeaconNetworkConfig().AttestationSubnetCount {
		t.Errorf("Wanted the number of subnet topics registered to be %d but got %d", params.BeaconNetworkConfig().AttestationSubnetCount, len(topics))
	}
	cancel()
}
