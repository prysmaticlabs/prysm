package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPipelineProcessing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)

	var validateCalled bool
	var handleCalled bool

	topic := "/foo/bar"
	sub, err := p.PubSub().Subscribe(topic + p.Encoding().ProtocolSuffix())
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		pipe := &pipeline{
			ctx:   context.Background(),
			topic: topic,
			base:  &testpb.TestSimpleMessage{},
			validate: func(_ context.Context, _ proto.Message, _ p2p.Broadcaster, _ bool) (bool, error) {
				validateCalled = true
				wg.Done()
				return true, nil
			},
			handle: func(_ context.Context, _ proto.Message) error {
				handleCalled = true
				wg.Done()
				return nil
			},
			encoding:     p.Encoding(),
			self:         p.PeerID(),
			sub:          sub,
			broadcaster:  p,
			chainStarted: func() bool { return true },
		}

		go pipe.messageLoop()
	}()

	p.ReceivePubSub(topic, &testpb.TestSimpleMessage{Foo: []byte("foo")})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive waitgroup finished within 1s")
	}

	if !validateCalled {
		t.Error("Validate was not called.")
	}
	if !handleCalled {
		t.Error("Handle was not called.")
	}
}
