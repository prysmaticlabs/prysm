package p2p

import (
	"context"
	"sync"
	"testing"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/proto/prysm/v2"

	"github.com/libp2p/go-libp2p-core/network"
	testp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/proto"
)

func TestService_Send(t *testing.T) {
	p1 := testp2p.NewTestP2P(t)
	p2 := testp2p.NewTestP2P(t)
	p1.Connect(p2)

	svc := &Service{
		host: p1.BHost,
		cfg:  &Config{},
	}

	msg := &pb.Fork{
		CurrentVersion:  []byte("fooo"),
		PreviousVersion: []byte("barr"),
		Epoch:           55,
	}

	// Register external listener which will repeat the message back.
	var wg sync.WaitGroup
	wg.Add(1)
	topic := "/testing/1"
	RPCTopicMappings[topic] = new(pb.Fork)
	defer func() {
		delete(RPCTopicMappings, topic)
	}()
	p2.SetStreamHandler(topic+"/ssz_snappy", func(stream network.Stream) {
		rcvd := &pb.Fork{}
		require.NoError(t, svc.Encoding().DecodeWithMaxLength(stream, rcvd))
		_, err := svc.Encoding().EncodeWithMaxLength(stream, rcvd)
		require.NoError(t, err)
		assert.NoError(t, stream.Close())
		wg.Done()
	})

	stream, err := svc.Send(context.Background(), msg, "/testing/1", p2.BHost.ID())
	require.NoError(t, err)

	testutil.WaitTimeout(&wg, 1*time.Second)

	rcvd := &pb.Fork{}
	require.NoError(t, svc.Encoding().DecodeWithMaxLength(stream, rcvd))
	if !proto.Equal(rcvd, msg) {
		t.Errorf("Expected identical message to be received. got %v want %v", rcvd, msg)
	}
}
