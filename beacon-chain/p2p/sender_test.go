package p2p

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	testp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestService_Send(t *testing.T) {
	p1 := testp2p.NewTestP2P(t)
	p2 := testp2p.NewTestP2P(t)
	p1.Connect(p2)

	svc := &Service{
		host: p1.BHost,
		cfg:  &Config{},
	}

	msg := &testpb.TestSimpleMessage{
		Foo: []byte("hello"),
		Bar: 55,
	}

	// Register external listener which will repeat the message back.
	var wg sync.WaitGroup
	wg.Add(1)

	p2.SetStreamHandler("/testing/1/ssz_snappy", func(stream network.Stream) {
		rcvd := &testpb.TestSimpleMessage{}
		if err := svc.Encoding().DecodeWithMaxLength(stream, rcvd); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Encoding().EncodeWithMaxLength(stream, rcvd); err != nil {
			t.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
		wg.Done()
	})

	stream, err := svc.Send(context.Background(), msg, "/testing/1", p2.BHost.ID())
	if err != nil {
		t.Fatal(err)
	}

	testutil.WaitTimeout(&wg, 1*time.Second)

	rcvd := &testpb.TestSimpleMessage{}
	if err := svc.Encoding().DecodeWithMaxLength(stream, rcvd); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(rcvd, msg) {
		t.Errorf("Expected identical message to be received. got %v want %v", rcvd, msg)
	}
}
