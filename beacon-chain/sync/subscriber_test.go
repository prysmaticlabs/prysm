package sync

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSubscribe_ReceivesValidMessage(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	topic := "/eth2/voluntary_exit"
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, noopValidator, func(_ context.Context, msg proto.Message) error {
		m := msg.(*pb.VoluntaryExit)
		if m.Epoch != 55 {
			t.Errorf("Unexpected incoming message: %+v", m)
		}
		wg.Done()
		return nil
	})

	p2p.ReceivePubSub(topic, &pb.VoluntaryExit{Epoch: 55})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestSubscribe_HandlesPanic(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	topic := "/eth2/voluntary_exit"
	var wg sync.WaitGroup
	wg.Add(1)

	r.subscribe(topic, noopValidator, func(_ context.Context, msg proto.Message) error {
		defer wg.Done()
		panic("bad")
	})

	p2p.ReceivePubSub(topic, &pb.VoluntaryExit{Epoch: 55})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestSubscribe_IgnoreMessageFromSelf(t *testing.T) {
	hook := logTest.NewGlobal()
	p2p := p2ptest.NewTestP2P(t)
	r := RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	topic := "/eth2/voluntary_exit"
	errorMsg := "Message entered into pipeline despite coming from same peer"

	r.subscribe(topic, noopValidator, func(_ context.Context, msg proto.Message) error {
		return errors.New(errorMsg)
	})
	buf := new(bytes.Buffer)
	msg := &pb.VoluntaryExit{Epoch: 55}
	if _, err := p2p.Encoding().Encode(buf, msg); err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	if err := p2p.PubSub().Publish(topic+p2p.Encoding().ProtocolSuffix(), buf.Bytes()); err != nil {
		t.Fatalf("Failed to publish message; %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	testutil.AssertLogsDoNotContain(t, hook, errorMsg)
}
