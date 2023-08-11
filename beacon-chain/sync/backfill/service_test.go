package backfill

import (
	"context"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/proto/dbval"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestServiceInit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()
	db := &mockBackfillDB{}
	su := NewStatus(db)
	nWorkers := 5
	var batchSize uint64 = 100
	nBatches := nWorkers * 2
	su.status = &dbval.BackfillStatus{
		HighSlot: 11235,
		LowSlot:  11235 - batchSize*uint64(nBatches),
	}
	remaining := nBatches
	cw := startup.NewClockSynchronizer()
	require.NoError(t, cw.SetClock(startup.NewClock(time.Now(), [32]byte{})))
	pool := &mockPool{todoChan: make(chan batch, nWorkers), finishedChan: make(chan batch, nWorkers)}
	srv, err := NewService(ctx, su, cw, pool, WithBatchSize(batchSize), WithWorkerCount(nWorkers))
	require.NoError(t, err)
	go srv.Start()
	todo := make([]batch, 0)
	todo = testReadN(t, ctx, pool.todoChan, nWorkers, todo)
	require.Equal(t, nWorkers, len(todo))
	for i := 0; i < remaining; i++ {
		b := todo[i]
		if b.state == batchSequenced {
			b.state = batchImportable
		}
		pool.finishedChan <- b
		todo = testReadN(t, ctx, pool.todoChan, 1, todo)
	}
	require.Equal(t, remaining+nWorkers, len(todo))
	for i := remaining; i < remaining+nWorkers; i++ {
		require.Equal(t, batchEndSequence, todo[i].state)
	}
}

func testReadN(t *testing.T, ctx context.Context, c chan batch, n int, into []batch) []batch {
	for i := 0; i < n; i++ {
		select {
		case b := <-c:
			into = append(into, b)
		case <-ctx.Done():
			// this means we hit the timeout, so something went wrong.
			require.Equal(t, true, false)
		}
	}
	return into
}
