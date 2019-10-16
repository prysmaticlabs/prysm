package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// expectSuccess status code from a stream in regular sync.
func expectSuccess(t *testing.T, r *RegularSync, stream network.Stream) {
	code, errMsg, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("Received non-zero response code: %d", code)
	}
	if errMsg != "" {
		t.Fatalf("Received error message from stream: %+v", errMsg)
	}
}

// expectResetStream status code from a stream in regular sync.
func expectResetStream(t *testing.T, r *RegularSync, stream network.Stream) {
	expectedErr := "stream reset"
	_, _, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	if err.Error() != expectedErr {
		t.Fatalf("Wanted this error %s but got %v instead", expectedErr, err)
	}
}

func TestRegisterRPC_ReceivesValidMessage(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := &RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	topic := "/testing/foobar/1"
	handler := func(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
		m := msg.(*pb.TestSimpleMessage)
		if !bytes.Equal(m.Foo, []byte("foo")) {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()

		return nil
	}
	r.registerRPC(topic, &pb.TestSimpleMessage{}, handler)

	p2p.ReceiveRPC(topic, &pb.TestSimpleMessage{Foo: []byte("foo")})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive RPC in 1 second")
	}
}
