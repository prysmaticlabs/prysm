package p2p

import (
	"context"
	"io/ioutil"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared"

	floodsub "github.com/libp2p/go-floodsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/sirupsen/logrus"
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

func TestSubscribeToTopic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	h := bhost.New(swarmt.GenSwarm(t, ctx))

	gsub, err := floodsub.NewFloodSub(ctx, h)
	if err != nil {
		t.Errorf("Failed to create floodsub: %v", err)
	}

	s := Server{
		ctx:   ctx,
		gsub:  gsub,
		host:  h,
		feeds: make(map[reflect.Type]*event.Feed),
		mutex: &sync.Mutex{},
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
	h := bhost.New(swarmt.GenSwarm(t, ctx))

	gsub, err := floodsub.NewFloodSub(ctx, h)
	if err != nil {
		t.Errorf("Failed to create floodsub: %v", err)
	}

	s := Server{
		ctx:   ctx,
		gsub:  gsub,
		host:  h,
		feeds: make(map[reflect.Type]*event.Feed),
		mutex: &sync.Mutex{},
	}

	ch := make(chan Message)
	sub := s.Subscribe(shardpb.CollationBodyRequest{}, ch)
	defer sub.Unsubscribe()

	testSubscribe(ctx, t, s, gsub, ch)
}

func testSubscribe(ctx context.Context, t *testing.T, s Server, gsub *floodsub.PubSub, ch chan Message) {
	topic := shardpb.Topic_COLLATION_BODY_REQUEST
	msgType := topicTypeMapping[topic]
	go s.subscribeToTopic(topic, msgType)

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

func TestRegisterTopic(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create new server: %v", err)
	}

	topic := "test_topic"

	type TestMessage struct{}

	s.RegisterTopic(topic, TestMessage{})

	ch := make(chan Message)
	sub := s.Subscribe(TestMessage{}, ch)
	defer sub.Unsubscribe()

	wait := make(chan struct{})
	go (func() {
		defer close(wait)
		msg := <-ch
		_ = msg
	})()

	if err := simulateIncomingMessage(s, topic, []byte{}); err != nil {
		t.Errorf("Failed to send to topic %s", topic)
	}

	select {
	case <-wait:
		return // OK
	case <-time.After(5 * time.Second):
		t.Fatal("TestMessage not received within 5 seconds")
	}
}

func simulateIncomingMessage(s *Server, topic string, b []byte) error {
	// TODO
	// Create a new host

	// Connect to s.Host

	// Use the new connection to Publish msg on topic

	return nil
}

func TestRegisterTopic_WithAdapers(t *testing.T) {
	// TODO: Test that adapters are called.
	// TODO: Use a test suite for different conditions.
}
