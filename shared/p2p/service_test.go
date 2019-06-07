package p2p

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	ggio "github.com/gogo/protobuf/io"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared"
	p2pmock "github.com/prysmaticlabs/prysm/shared/p2p/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure that server implements service.
var _ = shared.Service(&Server{})
var _ = Broadcaster(&Server{})
var _ = Sender(&Server{})

const bar = "bar"
const testTopic = "test_topic"

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestNewServer_InvalidMultiaddress(t *testing.T) {
	_, err := NewServer(&ServerConfig{
		RelayNodeAddr: "bad",
	})

	if err.Error() != "invalid multiaddr, must begin with /" {
		t.Fatal("expected invalid multiaddr err")
	}
}

func TestP2P_PortTaken(t *testing.T) {
	port := 10000
	_, err := NewServer(&ServerConfig{
		Port: port,
	})

	if err != nil {
		t.Fatalf("unable to create server: %s", err)
	}

	_, err = NewServer(&ServerConfig{
		Port: port,
	})

	if !strings.Contains(err.Error(), fmt.Sprintf("port %d already taken", port)) {
		t.Fatalf("expected fail when setting another server with same p2p port")
	}
}

func TestBroadcast_OK(t *testing.T) {
	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	msg := &shardpb.CollationBodyRequest{}
	s.Broadcast(context.Background(), msg)

	// TODO(543): test that topic was published
}

func TestEmit_OK(t *testing.T) {
	s, _ := NewServer(&ServerConfig{})
	p := &testpb.TestMessage{Foo: bar}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	feed := p2pmock.NewMockFeed(ctrl)
	var got Message
	feed.EXPECT().Send(gomock.AssignableToTypeOf(Message{})).Times(1).Do(func(m Message) {
		got = m
	})
	s.emit(Message{Ctx: context.Background(), Data: p}, feed)
	if !proto.Equal(p, got.Data) {
		t.Error("feed was not called with the correct data")
	}
}

func TestSubscribeToTopic_onPubSub_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := pubsub.NewFloodSub(ctx, h2)
	if err != nil {
		t.Errorf("Failed to create pubsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	feed := s.Feed(&shardpb.CollationBodyRequest{})
	ch := make(chan Message)
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()

	testSubscribe(ctx, t, s, gsub, ch)
}

func TestSubscribeToTopic_directMessaging_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		t.Errorf("Failed to create pubsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	feed := s.Feed(&shardpb.CollationBodyRequest{})
	ch := make(chan Message)
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()
	topic := shardpb.Topic_COLLATION_BODY_REQUEST

	s.RegisterTopic(topic.String(), &shardpb.CollationBodyRequest{})
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

	h2 := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	if err := h2.Connect(ctx, pstore.PeerInfo{ID: h.ID(), Addrs: h.Addrs()}); err != nil {
		t.Fatal(err)
	}
	stream, err := h2.NewStream(ctx, h.ID(), protocol.ID(prysmProtocolPrefix+"/"+topic.String()))
	if err != nil {
		t.Fatal(err)
	}
	w := ggio.NewDelimitedWriter(stream)
	defer w.Close()
	w.WriteMsg(createEnvelope(t, pbMsg))

	// Wait for our message assertion to complete.
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("Context timed out before a message was received!")
	}
}

func TestSubscribe_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := pubsub.NewFloodSub(ctx, h2)
	if err != nil {
		t.Errorf("Failed to create pubsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	ch := make(chan Message)
	sub := s.Subscribe(&shardpb.CollationBodyRequest{}, ch)
	defer sub.Unsubscribe()

	testSubscribe(ctx, t, s, gsub, ch)
}

func testSubscribe(ctx context.Context, t *testing.T, s Server, gsub *pubsub.PubSub, ch chan Message) {
	topic := shardpb.Topic_COLLATION_BODY_REQUEST

	s.RegisterTopic(topic.String(), &shardpb.CollationBodyRequest{})

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

	if err := gsub.Publish(topic.String(), createEnvelopeBytes(t, pbMsg)); err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	// Wait for our message assertion to complete.
	select {
	case <-done:
	case <-ctx.Done():
		t.Error("Context timed out before a message was received!")
	}
}

func TestRegisterTopic_InvalidProtobufs(t *testing.T) {
	topic := shardpb.Topic_COLLATION_BODY_REQUEST
	hook := logTest.NewGlobal()

	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	h2 := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	gsub, err := pubsub.NewFloodSub(ctx, h2)

	if err != nil {
		t.Errorf("Failed to create floodsub: %v", err)
	}

	s := Server{
		ctx:          ctx,
		gsub:         gsub,
		host:         h,
		feeds:        make(map[reflect.Type]Feed),
		mutex:        &sync.Mutex{},
		topicMapping: make(map[reflect.Type]string),
	}

	s.RegisterTopic(topic.String(), &shardpb.CollationBodyRequest{})
	ch := make(chan Message)
	sub := s.Subscribe(&shardpb.CollationBodyRequest{}, ch)
	defer sub.Unsubscribe()

	if err = gsub.Publish(topic.String(), []byte("invalid protobuf message")); err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}
	pbMsg := &shardpb.CollationBodyRequest{ShardId: 5}
	if err = gsub.Publish(topic.String(), createEnvelopeBytes(t, pbMsg)); err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	select {
	case <-ctx.Done():
		t.Error("Context timed out before a message was received!")
	case <-ch:
		logContains(t, hook, "Failed to decode data", logrus.ErrorLevel)
	}
}

func TestRegisterTopic_WithoutAdapters(t *testing.T) {
	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}
	topic := testTopic
	testMessage := &testpb.TestMessage{Foo: bar}

	s.RegisterTopic(topic, testMessage)

	ch := make(chan Message)
	sub := s.Subscribe(testMessage, ch)
	defer sub.Unsubscribe()

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		msg := <-ch
		tmsg := msg.Data.(*testpb.TestMessage)
		if tmsg.Foo != bar {
			t.Errorf("Expected test message foo:\"bar\". Got: %v", tmsg)
		}
	}()

	if err := simulateIncomingMessage(t, s, topic, testMessage); err != nil {
		t.Errorf("Failed to send to topic %s", topic)
	}

	select {
	case <-wait:
		return // OK
	case <-time.After(1 * time.Second):
		t.Fatal("TestMessage not received within 1 seconds")
	}
}

func TestRegisterTopic_WithAdapters(t *testing.T) {
	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}
	topic := testTopic
	testMessage := &testpb.TestMessage{Foo: bar}

	i := 0
	var testAdapter Adapter = func(next Handler) Handler {
		return func(msg Message) {
			i++
			next(msg)
		}
	}

	adapters := []Adapter{
		testAdapter,
		testAdapter,
		testAdapter,
		testAdapter,
		testAdapter,
	}

	s.RegisterTopic(topic, testMessage, adapters...)

	ch := make(chan Message)
	sub := s.Subscribe(testMessage, ch)
	defer sub.Unsubscribe()

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		msg := <-ch
		tmsg := msg.Data.(*testpb.TestMessage)
		if tmsg.Foo != bar {
			t.Errorf("Expected test message Foo: \"bar\". Got: %v", tmsg)
		}
	}()

	if err := simulateIncomingMessage(t, s, topic, testMessage); err != nil {
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

func TestRegisterTopic_HandlesPanic(t *testing.T) {
	hook := logTest.NewGlobal()

	s, err := NewServer(&ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}
	topic := testTopic
	testMessage := &testpb.TestMessage{Foo: bar}

	var panicAdapter Adapter = func(next Handler) Handler {
		return func(msg Message) {
			panic("bad!")
		}
	}

	s.RegisterTopic(topic, testMessage, panicAdapter)

	ch := make(chan Message)
	sub := s.Subscribe(testMessage, ch)
	defer sub.Unsubscribe()

	if err := simulateIncomingMessage(t, s, topic, testMessage); err != nil {
		t.Errorf("Failed to send to topic %s", topic)
	}

	testutil.WaitForLog(t, hook, "P2P message caused a panic")
}

func TestStatus_MinimumPeers(t *testing.T) {
	minPeers := 3

	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	s := Server{host: h}

	err := s.Status()
	if err == nil || err.Error() != "less than 3 peers" {
		t.Errorf("p2p server did not return expected status, instead returned %v", err)
	}

	for i := 0; i < minPeers; i++ {
		other := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
		if err := h.Connect(ctx, other.Peerstore().PeerInfo(other.ID())); err != nil {
			t.Fatalf("Could not connect to host for test setup")
		}
	}

	if err := s.Status(); err != nil {
		t.Errorf("Unexpected server status %v", err)
	}
}

func simulateIncomingMessage(t *testing.T, s *Server, topic string, msg proto.Message) error {
	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))

	setHandshakeHandler(h, "")

	gsub, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		return err
	}

	pinfo := h.Peerstore().PeerInfo(h.ID())
	if err = s.host.Connect(ctx, pinfo); err != nil {
		return err
	}

	// Short timeout to allow libp2p to handle peer connection.
	time.Sleep(time.Millisecond * 100)

	return gsub.Publish(topic, createEnvelopeBytes(t, msg))
}

func createEnvelope(t *testing.T, msg proto.Message) *pb.Envelope {
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	// span context test data from
	// https://github.com/census-instrumentation/opencensus-go/blob/3b8e2721f2c3c01fa1bf4a2e455874e7b8319cd7/trace/propagation/propagation_test.go#L69
	envelope := &pb.Envelope{
		SpanContext: []byte{0, 0, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 1, 97, 98, 99, 100, 101, 102, 103, 104, 2, 1},
		Payload:     payload,
	}

	return envelope
}

func createEnvelopeBytes(t *testing.T, msg proto.Message) []byte {
	b, err := proto.Marshal(createEnvelope(t, msg))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func logContains(t *testing.T, hook *logTest.Hook, message string, level logrus.Level) {
	var logs string
	for _, entry := range hook.AllEntries() {
		logs = fmt.Sprintf("%s\nlevel=%s msg=\"%s\"", logs, entry.Level, entry.Message)
		if entry.Level == level && strings.Contains(entry.Message, message) {
			return
		}
	}
	t.Errorf("Expected log to contain level=%s and msg=\"%s\" inside log entries: %s", level, message, logs)
}
