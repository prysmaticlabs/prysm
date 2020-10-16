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
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_PublishToTopicConcurrentMapWrite(t *testing.T) {
	s, err := NewService(context.Background(), &Config{
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

	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			topic := fmt.Sprintf(AttestationSubnetTopicFormat, fd, i) + "/" + encoder.ProtocolSuffixSSZSnappy
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
