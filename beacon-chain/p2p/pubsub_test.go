package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	testp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_PublishToTopicConcurrentMapWrite(t *testing.T) {
	s, err := NewService(&Config{
		StateNotifier: &mock.MockStateNotifier{},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go s.awaitStateInitialized(ctx)
	fd := initializeStateWithForkDigest(ctx, t, s.stateNotifier.StateFeed())

	if !s.isInitialized() {
		t.Fatal("service was not initialized")
	}

	// Set up two connected test hosts.
	p0 := testp2p.NewTestP2P(t)
	p1 := testp2p.NewTestP2P(t)
	p0.Connect(p1)
	s.host = p0.BHost
	s.pubsub = p0.PubSub()

	topic := fmt.Sprintf(BlockSubnetTopicFormat, fd) + "/" + encoder.ProtocolSuffixSSZSnappy

	// Establish the remote peer to be subscribed to the outgoing topic.
	_, err = p1.SubscribeToTopic(topic)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			assert.NoError(t, s.PublishToTopic(ctx, topic, []byte{}))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestMessageIDFunction_HashesCorrectly(t *testing.T) {
	msg := [32]byte{'J', 'U', 'N', 'K'}
	pMsg := &pubsubpb.Message{Data: msg[:]}
	hashedData := hashutil.Hash(pMsg.Data)
	msgID := string(hashedData[:])
	assert.Equal(t, msgID, msgIDFunction(pMsg), "Got incorrect msg id")
}
