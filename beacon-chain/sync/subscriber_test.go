package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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
	r.chainStarted = true

	p2p.ReceivePubSub(topic, &pb.VoluntaryExit{Epoch: 55})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}

func TestSubscribe_WaitToSync(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := RegularSync{
		ctx: context.Background(),
		p2p: p2p,
	}

	r.chain = &mockChain.ChainService{}
	topic := "/eth2/beacon_block"
	r.registerSubscribers()
	i := r.chain.StateInitializedFeed().Send(time.Now())
	if i == 0 {
		t.Fatal("didn't send genesis time to subscribers")
	}
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}

	msg := &pb.BeaconBlock{
		ParentRoot: testutil.Random32Bytes(t),
		Signature:  sk.Sign([]byte("data"), 0).Marshal(),
	}
	p2p.ReceivePubSub(topic, msg)
	if !r.chainStarted {
		t.Fatal("Did not receive chain start event.")
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
	r.chainStarted = true
	p2p.ReceivePubSub(topic, &pb.VoluntaryExit{Epoch: 55})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive PubSub in 1 second")
	}
}
