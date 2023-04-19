package backfill

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	log "github.com/sirupsen/logrus"
)

type workerId int

type p2pWorker struct {
	id   workerId
	p2p  p2p.P2P
	todo chan batch
	done chan batch
}

func (w *p2pWorker) run(ctx context.Context) {
	for {
		select {
		case b := <-w.todo:
			log.WithFields(b.logFields()).Debug("Backfill worker received batch.")
			w.done <- w.handle(b)
		case <-ctx.Done():
			log.WithField("worker_id", w.id).Info("Backfill worker exiting after context canceled.")
			return
		}
	}
}

func (w *p2pWorker) handle(b batch) batch {
	// if the batch is not successfully fetched and validated, increment the attempts counter
	return b
}

func newP2pWorker(id workerId, p p2p.P2P, todo, done chan batch) *p2pWorker {
	return &p2pWorker{
		id:   id,
		p2p:  p,
		todo: todo,
		done: done,
	}
}
