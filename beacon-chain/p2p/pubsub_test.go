package p2p

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_PublishToTopicConcurrentMapWrite(t *testing.T) {
	s, err := NewService(&Config{})
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
