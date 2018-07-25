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
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure that server implements service.
var _ = shared.Service(&Server{})

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()

	s, err := NewServer()
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	s.Start()
	msg := hook.Entries[0].Message
	want := "Starting shardp2p server"
	if msg != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

	s.Stop()
	msg = hook.LastEntry().Message
	want = "Stopping shardp2p server"
	if msg != want {
		t.Errorf("incorrect log. wanted: %s. got: %v", want, msg)
	}

	// The context should have been cancelled.
	if s.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}

func TestBroadcast(t *testing.T) {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	msg := &pb.CollationBodyRequest{}
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

	feed := s.Feed(pb.CollationBodyRequest{})
	ch := make(chan Message)
	sub := feed.Subscribe(ch)
	defer sub.Unsubscribe()

	topic := pb.Topic_COLLATION_BODY_REQUEST
	msgType := topicTypeMapping[topic]
	go s.subscribeToTopic(topic, msgType)

	// Short delay to let goroutine add subscription.
	time.Sleep(time.Millisecond * 10)

	// The topic should be subscribed with gsub.
	topics := gsub.GetTopics()
	if len(topics) < 1 || topics[0] != topic.String() {
		t.Errorf("Unexpected subscribed topics: %v. Wanted %s", topics, topic)
	}

	pbMsg := &pb.CollationBodyRequest{ShardId: 5}

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
		t.Errorf("Failed to marshal pbMsg: %v", err)
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
