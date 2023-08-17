package backfill

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	log "github.com/sirupsen/logrus"
)

type workerId int

type p2pWorker struct {
	id   workerId
	todo chan batch
	done chan batch
	p2p  p2p.P2P
	v    *verifier
	c    *startup.Clock
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
	results, err := sync.SendBeaconBlocksByRangeRequest(ctx, w.c, w.p2p, b.pid, b.request(), nil)
	// if the batch is not successfully fetched and validated, increment the attempts counter
	vb, err := w.v.verify(results)
	if err != nil {
		b.err = err
		return b
	}
	b.results = vb
	return b
}

func newP2pWorker(id workerId, p p2p.P2P, todo, done chan batch, c *startup.Clock, v *verifier) *p2pWorker {
	return &p2pWorker{
		id:   id,
		todo: todo,
		done: done,
		p2p:  p,
		v:    v,
		c:    c,
	}
}
