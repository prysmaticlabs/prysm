package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	prysmP2P "github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func init() {
	state.SkipSlotCache.Disable()
}

// expectSuccess status code from a stream in regular sync.
func expectSuccess(t *testing.T, stream network.Stream) {
	code, errMsg, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	require.NoError(t, err)
	require.Equal(t, uint8(0), code, "Received non-zero response code")
	require.Equal(t, "", errMsg, "Received error message from stream")
}

// expectSuccess status code from a stream in regular sync.
func expectFailure(t *testing.T, expectedCode uint8, expectedErrorMsg string, stream network.Stream) {
	code, errMsg, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	require.NoError(t, err)
	require.NotEqual(t, uint8(0), code, "Expected request to fail but got a 0 response code")
	require.Equal(t, expectedCode, code, "Received incorrect response code")
	require.Equal(t, expectedErrorMsg, errMsg)
}

// expectResetStream status code from a stream in regular sync.
func expectResetStream(t *testing.T, stream network.Stream) {
	expectedErr := "stream reset"
	_, _, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	require.ErrorContains(t, expectedErr, err)
}

func TestRegisterRPC_ReceivesValidMessage(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	r := &Service{
		ctx:         context.Background(),
		cfg:         &Config{P2P: p2p},
		rateLimiter: newRateLimiter(p2p),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	topic := "/testing/foobar/1"
	handler := func(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
		m, ok := msg.(*p2ppb.Fork)
		if !ok {
			t.Error("Object is not of type *pb.TestSimpleMessage")
		}
		assert.DeepEqual(t, []byte("fooo"), m.CurrentVersion, "Unexpected incoming message")
		wg.Done()

		return nil
	}
	prysmP2P.RPCTopicMappings[topic] = new(p2ppb.Fork)
	// Cleanup Topic mappings
	defer func() {
		delete(prysmP2P.RPCTopicMappings, topic)
	}()
	r.registerRPC(topic, handler)

	p2p.ReceiveRPC(topic, &p2ppb.Fork{CurrentVersion: []byte("fooo"), PreviousVersion: []byte("barr")})

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive RPC in 1 second")
	}
}
