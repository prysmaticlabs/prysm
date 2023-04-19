package backfill

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	p2ptest "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

type MockAssigner struct {
	err    error
	assign []peer.ID
}

func (m MockAssigner) Assign(busy map[peer.ID]bool, n int) ([]peer.ID, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.assign, nil
}

var _ peerAssigner = &MockAssigner{}

func TestPoolDetectAllEnded(t *testing.T) {
	nw := 5
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	ma := &MockAssigner{}
	pool := newP2PBatchWorkerPool(p2p, nw)
	st, err := util.NewBeaconState()
	require.NoError(t, err)
	v, err := newBackfillVerifier(st)
	require.NoError(t, err)
	pool.Spawn(ctx, nw, startup.NewClock(time.Now(), [32]byte{}), ma, v)
	br := batcher{min: 10, size: 10}
	endSeq := br.before(0)
	require.Equal(t, batchEndSequence, endSeq.state)
	for i := 0; i < nw; i++ {
		pool.Todo(endSeq)
	}
	b, err := pool.Complete()
	require.ErrorIs(t, err, errEndSequence)
	require.Equal(t, b.end, endSeq.end)
}

type mockPool struct {
	spawnCalled  []int
	finishedChan chan batch
	finishedErr  chan error
	todoChan     chan batch
}

func (m *mockPool) Spawn(_ context.Context, _ int, _ *startup.Clock, _ peerAssigner, _ *verifier) {
}

func (m *mockPool) Todo(b batch) {
	m.todoChan <- b
}

func (m *mockPool) Complete() (batch, error) {
	select {
	case b := <-m.finishedChan:
		return b, nil
	case err := <-m.finishedErr:
		return batch{}, err
	}
}

var _ BatchWorkerPool = &mockPool{}
