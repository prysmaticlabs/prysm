package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_PublishToTopicConcurrentMapWrite(t *testing.T) {
	s, err := NewService(context.Background(), &Config{})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			assert.NoError(t, s.PublishToTopic(ctx, fmt.Sprintf("foo%v", i), []byte{}))
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func TestMessageIDFunction_HashesCorrectly(t *testing.T) {
	msg := [32]byte{'J', 'U', 'N', 'K'}
	pMsg := &pubsubpb.Message{Data: msg[:]}
	hashedData := hashutil.Hash(pMsg.Data)
	msgID := string(hashedData[:8])
	assert.Equal(t, msgID, msgIDFunction(pMsg), "Got incorrect msg id")
}
