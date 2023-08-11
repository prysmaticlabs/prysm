package backfill

import (
	"context"
	"testing"

	p2ptest "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestPoolDetectAllEnded(t *testing.T) {
	nw := 5
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()
	pool := NewP2PBatchWorkerPool(p2p)
	pool.Spawn(ctx, nw)
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

func (m *mockPool) Spawn(_ context.Context, _ int) {
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
