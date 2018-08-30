package p2p

import (
	"context"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	floodsub "github.com/libp2p/go-floodsub"
	floodsubPb "github.com/libp2p/go-floodsub/pb"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure that server implements service.
var _ = shared.Service(&Server{})

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestBroadcast(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	msg := &shardpb.CollationBodyRequest{}
	s.Broadcast(msg)

	// TODO: test that topic was published
}

func TestEmitFailsNonProtobuf(t *testing.T) {
	s, _ := NewServer()
	hook := logTest.NewGlobal()
	s.emit(nil /*feed*/, nil /*msg*/, reflect.TypeOf(""))
	want := "Received message is not a protobuf message"
	if hook.LastEntry().Message != want {
		t.Errorf("Expected log to contain %s. Got = %s", want, hook.LastEntry().Message)
	}
}

func TestEmitFailsUnmarshal(t *testing.T) {
	s, _ := NewServer()
	hook := logTest.NewGlobal()
	msg := &floodsub.Message{
		&floodsubPb.Message{
			Data: []byte("bogus"),
		},
	}

	s.emit(nil /*feed*/, msg, reflect.TypeOf(testpb.TestMessage{}))
	want := "Failed to decode data:"
	if !strings.Contains(hook.LastEntry().Message, want) {
		t.Errorf("Expected log to contain %s. Got = %s", want, hook.LastEntry().Message)
	}
}

func TestSubscribeToTopic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := floodsub.NewFloodSub(ctx, h)
	if err != nil {
		t.Errorf("Failed to create floodsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]*event.Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	feed := s.Feed(shardpb.CollationBodyRequest{})
	ch := make(chan Message)
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()

	testSubscribe(ctx, t, s, gsub, ch)
}

func TestSubscribe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := floodsub.NewFloodSub(ctx, h)
	if err != nil {
		t.Errorf("Failed to create floodsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]*event.Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	ch := make(chan Message)
	sub := s.Subscribe(shardpb.CollationBodyRequest{}, ch)
	defer sub.Unsubscribe()

	testSubscribe(ctx, t, s, gsub, ch)
}

func testSubscribe(ctx context.Context, t *testing.T, s Server, gsub *floodsub.PubSub, ch chan Message) {
	topic := shardpb.Topic_COLLATION_BODY_REQUEST

	go s.RegisterTopic(topic.String(), shardpb.CollationBodyRequest{})

	// Short delay to let goroutine add subscription.
	time.Sleep(time.Millisecond * 10)

	// The topic should be subscribed with gsub.
	topics := gsub.GetTopics()
	if len(topics) < 1 || topics[0] != topic.String() {
		t.Errorf("Unexpected subscribed topics: %v. Wanted %s", topics, topic)
	}

	pbMsg := &shardpb.CollationBodyRequest{ShardId: 5}

	done := make(chan bool)
	go func() {
		// The message should be received from the feed.
		msg := <-ch
		if !proto.Equal(msg.Data.(proto.Message), pbMsg) {
			t.Errorf("Unexpected msg: %+v. Wanted %+v.", msg.Data, pbMsg)
		}

		done <- true
	}()

	b, err := proto.Marshal(pbMsg)
	if err != nil {
		t.Errorf("Failed to marshal service %v", err)
	}
	if err = gsub.Publish(topic.String(), b); err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	// Wait for our message assertion to complete.
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("Context timed out before a message was received!")
	}
}

func TestRegisterTopic_WithoutAdapters(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}
	topic := "test_topic"
	testMessage := testpb.TestMessage{Foo: "bar"}

	s.RegisterTopic(topic, testpb.TestMessage{})

	ch := make(chan Message)
	sub := s.Subscribe(testMessage, ch)
	defer sub.Unsubscribe()

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		<-ch
	}()

	if err := simulateIncomingMessage(t, s, topic, []byte{}); err != nil {
		t.Errorf("Failed to send to topic %s", topic)
	}

	select {
	case <-wait:
		return // OK
	case <-time.After(1 * time.Second):
		t.Fatal("TestMessage not received within 1 seconds")
	}
}

func TestRegisterTopic_WithAdapers(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}
	topic := "test_topic"
	testMessage := testpb.TestMessage{Foo: "bar"}

	i := 0
	var testAdapter Adapter = func(next Handler) Handler {
		return func(ctx context.Context, msg Message) {
			i++
			next(ctx, msg)
		}
	}

	adapters := []Adapter{
		testAdapter,
		testAdapter,
		testAdapter,
		testAdapter,
		testAdapter,
	}

	s.RegisterTopic(topic, testpb.TestMessage{}, adapters...)

	ch := make(chan Message)
	sub := s.Subscribe(testMessage, ch)
	defer sub.Unsubscribe()

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		<-ch
	}()

	if err := simulateIncomingMessage(t, s, topic, []byte{}); err != nil {
		t.Errorf("Failed to send to topic %s", topic)
	}

	select {
	case <-wait:
		if i != 5 {
			t.Errorf("Expected testAdapter to increment i to 5, but was %d", i)
		}
		return // OK
	case <-time.After(1 * time.Second):
		t.Fatal("TestMessage not received within 1 seconds")
	}
}

func simulateIncomingMessage(t *testing.T, s *Server, topic string, b []byte) error {
	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := floodsub.NewFloodSub(ctx, h)
	if err != nil {
		return err
	}

	pinfo := h.Peerstore().PeerInfo(h.ID())
	if err = s.host.Connect(ctx, pinfo); err != nil {
		return err
	}

	// Short timeout to allow libp2p to handle peer connection.
	time.Sleep(time.Millisecond * 10)

	return gsub.Publish(topic, b)
}
