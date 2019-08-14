package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSubscribe_ReceivesValidMessage(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	topic := "/testing/foobar"
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, &pb.TestSimpleMessage{}, noopValidator, func(_ context.Context, msg proto.Message) error {
		m := msg.(*pb.TestSimpleMessage)
		if !bytes.Equal(m.Foo, []byte("foo")) {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()
		return nil
	})

	p2p.ReceivePubSub(topic, &pb.TestSimpleMessage{Foo: []byte("foo")})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}
