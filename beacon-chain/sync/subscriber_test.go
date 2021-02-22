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
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	pb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/abool"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubscribe_ReceivesValidMessage(t *testing.T) {
	p2pService := p2ptest.NewTestP2P(t)
	r := Service{
		ctx:         context.Background(),
		p2p:         p2pService,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mockChain.ChainService{
			ValidatorsRoot: [32]byte{'A'},
			Genesis:        time.Now(),
		},
		chainStarted: abool.New(),
	}
	var err error
	p2pService.Digest, err = r.forkDigest()
	require.NoError(t, err)
	topic := "/eth2/%x/voluntary_exit"
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, r.noopValidator, func(_ context.Context, msg proto.Message) error {
		m, ok := msg.(*pb.SignedVoluntaryExit)
		assert.Equal(t, true, ok, "Object is not of type *pb.SignedVoluntaryExit")
		if m.Exit == nil || m.Exit.Epoch != 55 {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()
		return nil
	})
	r.markForChainStart()

	p2pService.ReceivePubSub(topic, &pb.SignedVoluntaryExit{Exit: &pb.VoluntaryExit{Epoch: 55}, Signature: make([]byte, 96)})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestSubscribe_ReceivesAttesterSlashing(t *testing.T) {
	p2pService := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	d := db.SetupDB(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	r := Service{
		ctx:                       ctx,
		p2p:                       p2pService,
		initialSync:               &mockSync.Sync{IsSyncing: false},
		slashingPool:              slashings.NewPool(),
		chain:                     chainService,
		db:                        d,
		seenAttesterSlashingCache: make(map[uint64]bool),
		chainStarted:              abool.New(),
	}
	topic := "/eth2/%x/attester_slashing"
	var wg sync.WaitGroup
	wg.Add(1)
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	r.subscribe(topic, r.noopValidator, func(ctx context.Context, msg proto.Message) error {
		require.NoError(t, r.attesterSlashingSubscriber(ctx, msg))
		wg.Done()
		return nil
	})
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	chainService.State = beaconState
	r.markForChainStart()
	attesterSlashing, err := testutil.GenerateAttesterSlashingForValidator(
		beaconState,
		privKeys[1],
		1, /* validator index */
	)
	require.NoError(t, err, "Error generating attester slashing")
	err = r.db.SaveState(ctx, beaconState, bytesutil.ToBytes32(attesterSlashing.Attestation_1.Data.BeaconBlockRoot))
	require.NoError(t, err)
	p2pService.Digest, err = r.forkDigest()
	require.NoError(t, err)
	p2pService.ReceivePubSub(topic, attesterSlashing)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
	as := r.slashingPool.PendingAttesterSlashings(ctx, beaconState, false /*noLimit*/)
	assert.Equal(t, 1, len(as), "Expected attester slashing")
}

func TestSubscribe_ReceivesProposerSlashing(t *testing.T) {
	p2pService := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	chainService := &mockChain.ChainService{
		ValidatorsRoot: [32]byte{'A'},
		Genesis:        time.Now(),
	}
	d := db.SetupDB(t)
	c, err := lru.New(10)
	require.NoError(t, err)
	r := Service{
		ctx:                       ctx,
		p2p:                       p2pService,
		initialSync:               &mockSync.Sync{IsSyncing: false},
		slashingPool:              slashings.NewPool(),
		chain:                     chainService,
		db:                        d,
		seenProposerSlashingCache: c,
		chainStarted:              abool.New(),
	}
	topic := "/eth2/%x/proposer_slashing"
	var wg sync.WaitGroup
	wg.Add(1)
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	r.subscribe(topic, r.noopValidator, func(ctx context.Context, msg proto.Message) error {
		require.NoError(t, r.proposerSlashingSubscriber(ctx, msg))
		wg.Done()
		return nil
	})
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 64)
	chainService.State = beaconState
	r.markForChainStart()
	proposerSlashing, err := testutil.GenerateProposerSlashingForValidator(
		beaconState,
		privKeys[1],
		1, /* validator index */
	)
	require.NoError(t, err, "Error generating proposer slashing")
	p2pService.Digest, err = r.forkDigest()
	require.NoError(t, err)
	p2pService.ReceivePubSub(topic, proposerSlashing)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
	ps := r.slashingPool.PendingProposerSlashings(ctx, beaconState, false /*noLimit*/)
	assert.Equal(t, 1, len(ps), "Expected proposer slashing")
}

func TestSubscribe_HandlesPanic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	r := Service{
		ctx: context.Background(),
		chain: &mockChain.ChainService{
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		p2p:          p,
		chainStarted: abool.New(),
	}
	var err error
	p.Digest, err = r.forkDigest()
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(&pb.SignedVoluntaryExit{})]
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, r.noopValidator, func(_ context.Context, msg proto.Message) error {
		defer wg.Done()
		panic("bad")
	})
	r.markForChainStart()
	p.ReceivePubSub(topic, &pb.SignedVoluntaryExit{Exit: &pb.VoluntaryExit{Epoch: 55}, Signature: make([]byte, 96)})

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
		p2p:          p,
		chainStarted: abool.New(),
	}
	digest, err := r.forkDigest()
	require.NoError(t, err)
	subscriptions := make(map[uint64]*pubsub.Subscription, params.BeaconConfig().MaxCommitteesPerSlot)

	defaultTopic := "/eth2/testing/%#x/committee%d"
	// committee index 1
	fullTopic := fmt.Sprintf(defaultTopic, digest, 1) + r.p2p.Encoding().ProtocolSuffix()
	require.NoError(t, r.p2p.PubSub().RegisterTopicValidator(fullTopic, r.noopValidator))
	subscriptions[1], err = r.p2p.SubscribeToTopic(fullTopic)
	require.NoError(t, err)

	// committee index 2
	fullTopic = fmt.Sprintf(defaultTopic, digest, 2) + r.p2p.Encoding().ProtocolSuffix()
	err = r.p2p.PubSub().RegisterTopicValidator(fullTopic, r.noopValidator)
	require.NoError(t, err)
	subscriptions[2], err = r.p2p.SubscribeToTopic(fullTopic)
	require.NoError(t, err)

	r.reValidateSubscriptions(subscriptions, []uint64{2}, defaultTopic, digest)
	require.LogsDoNotContain(t, hook, "Could not unregister topic validator")
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
		p2p:          p,
		chainStarted: abool.New(),
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

func Test_wrapAndReportValidation(t *testing.T) {
	type args struct {
		topic        string
		v            pubsub.ValidatorEx
		chainstarted bool
		pid          peer.ID
		msg          *pubsub.Message
	}
	tests := []struct {
		name string
		args args
		want pubsub.ValidationResult
	}{
		{
			name: "validator Before chainstart",
			args: args{
				topic: "foo",
				v: func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
					return pubsub.ValidationAccept
				},
				msg: &pubsub.Message{
					Message: &pubsubpb.Message{
						Topic: func() *string {
							s := "foo"
							return &s
						}(),
					},
				},
				chainstarted: false,
			},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "validator panicked",
			args: args{
				topic: "foo",
				v: func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
					panic("oh no!")
				},
				chainstarted: true,
				msg: &pubsub.Message{
					Message: &pubsubpb.Message{
						Topic: func() *string {
							s := "foo"
							return &s
						}(),
					},
				},
			},
			want: pubsub.ValidationIgnore,
		},
		{
			name: "validator OK",
			args: args{
				topic: "foo",
				v: func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
					return pubsub.ValidationAccept
				},
				chainstarted: true,
				msg: &pubsub.Message{
					Message: &pubsubpb.Message{
						Topic: func() *string {
							s := "foo"
							return &s
						}(),
					},
				},
			},
			want: pubsub.ValidationAccept,
		},
		{
			name: "nil topic",
			args: args{
				topic: "foo",
				v: func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
					return pubsub.ValidationAccept
				},
				msg: &pubsub.Message{
					Message: &pubsubpb.Message{
						Topic: nil,
					},
				},
			},
			want: pubsub.ValidationReject,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainStarted := abool.New()
			chainStarted.SetTo(tt.args.chainstarted)
			s := &Service{
				chainStarted: chainStarted,
			}
			_, v := s.wrapAndReportValidation(tt.args.topic, tt.args.v)
			got := v(context.Background(), tt.args.pid, tt.args.msg)
			if got != tt.want {
				t.Errorf("wrapAndReportValidation() got = %v, want %v", got, tt.want)
			}
		})
	}
}
