package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNewRateLimiter(t *testing.T) {
	rlimiter := newRateLimiter(mockp2p.NewTestP2P(t))
	assert.Equal(t, len(rlimiter.limiterMap), 6, "correct number of topics not registered")
}

func TestNewRateLimiter_FreeCorrectly(t *testing.T) {
	rlimiter := newRateLimiter(mockp2p.NewTestP2P(t))
	rlimiter.free()
	assert.Equal(t, len(rlimiter.limiterMap), 0, "rate limiter not freed correctly")

}

func TestRateLimiter_ExceedCapacity(t *testing.T) {
	p1 := mockp2p.NewTestP2P(t)
	p2 := mockp2p.NewTestP2P(t)
	p1.Connect(p2)
	rlimiter := newRateLimiter(p1)

	bFlags := flags.Get()
	bFlags.BlockBatchLimit = 64
	bFlags.BlockBatchLimitBurstFactor = 10
	reset := flags.InitWithReset(bFlags)
	defer reset()

	// BlockByRange
	topic := p2p.RPCBlocksByRangeTopic + p1.Encoding().ProtocolSuffix()

	wg := sync.WaitGroup{}
	p2.BHost.SetStreamHandler(protocol.ID(topic), func(stream network.Stream) {
		defer wg.Done()
		code, errMsg, err := readStatusCodeNoDeadline(stream, p2.Encoding())
		require.NoError(t, err, "could not read incoming stream")
		assert.Equal(t, responseCodeInvalidRequest, code, "not equal response codes")
		assert.Equal(t, rateLimitedError, errMsg, "not equal errors")

	})
	wg.Add(1)
	stream, err := p1.BHost.NewStream(context.Background(), p2.PeerID(), protocol.ID(topic))
	require.NoError(t, err, "could not create stream")

	err = rlimiter.validateRequest(stream, 64)
	require.NoError(t, err, "could not validate incoming request")

	// Attempt to create an error, rate limit and lead to disconnect
	err = rlimiter.validateRequest(stream, 1000)
	require.NotNil(t, err, "could not get error from leaky bucket")

	require.NoError(t, stream.Close(), "could not close stream")

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}
