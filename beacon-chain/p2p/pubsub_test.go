package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/snappy"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	testp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_PublishToTopicConcurrentMapWrite(t *testing.T) {
	s, err := New(context.Background(), &Config{
		StateNotifier: &mock.MockStateNotifier{},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go s.awaitStateInitialized()
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
	invalidSnappy := [32]byte{'J', 'U', 'N', 'K'}
	pMsg := &pubsubpb.Message{Data: invalidSnappy[:]}
	hashedData := hashutil.Hash(append(params.BeaconNetworkConfig().MessageDomainInvalidSnappy[:], pMsg.Data...))
	msgID := string(hashedData[:20])
	assert.Equal(t, msgID, msgIDFunction(pMsg), "Got incorrect msg id")

	validObj := [32]byte{'v', 'a', 'l', 'i', 'd'}
	enc := snappy.Encode(nil, validObj[:])
	nMsg := &pubsubpb.Message{Data: enc}
	hashedData = hashutil.Hash(append(params.BeaconNetworkConfig().MessageDomainValidSnappy[:], validObj[:]...))
	msgID = string(hashedData[:20])
	assert.Equal(t, msgID, msgIDFunction(nMsg), "Got incorrect msg id")
}
