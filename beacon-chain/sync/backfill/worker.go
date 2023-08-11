package backfill

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	log "github.com/sirupsen/logrus"
)

type workerState int

const (
	workerIdle workerState = iota
	workerBusy
)

type workerId int

type p2pWorker struct {
	sync.Mutex
	ws   workerState
	id   workerId
	p2p  p2p.P2P
	todo chan batch
	done chan batch
}

func (w *p2pWorker) run(ctx context.Context) {
	for {
		select {
		case b := <-w.todo:
			log.WithFields(b.logFields()).WithField("backfill_worker", w.id).Debug("Backfill worker received batch.")
			w.done <- w.handle(ctx, b)
		case <-ctx.Done():
			log.WithField("backfill_worker", w.id).Info("Backfill worker exiting after context canceled.")
			return
		}
	}
}

func (w *p2pWorker) handle(ctx context.Context, b batch) batch {
	// if the batch is not successfully fetched and validated, increment the attempts counter
	return b
}

func (w *p2pWorker) updateState(ws workerState) {
	w.Lock()
	defer w.Unlock()
	w.ws = ws
}

func (w *p2pWorker) state() workerState {
	w.Lock()
	defer w.Unlock()
	return w.ws
}

func newP2pWorker(id workerId, p p2p.P2P, todo, done chan batch) *p2pWorker {
	return &p2pWorker{
		id:   id,
		p2p:  p,
		todo: todo,
		done: done,
	}
}
