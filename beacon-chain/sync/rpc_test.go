package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/testing"
)

func TestRegisterRPC(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := &RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	topic := "/testing/foobar"
	handler := func(ctx context.Context, msg proto.Message, stream libp2pcore.Stream) error {
		m := msg.(*pb.TestSimpleMessage)
		if !bytes.Equal(m.Foo, []byte("foo")) {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()

		return nil
	}
	r.registerRPC(topic, &pb.TestSimpleMessage{}, handler)

	p2p.ReceiveRPC(topic+"/ssz", &pb.TestSimpleMessage{Foo: []byte("foo")})

	if waitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive RPC in 1 second")
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	select {
	case <-ch:
		return false
	case <-time.After(timeout):
		return true
	}
}
