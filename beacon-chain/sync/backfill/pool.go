package backfill

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
)

type Worker interface {
	run(context.Context)
	state() workerState
}

type NewWorker func(id workerId, todo, done chan batch) Worker

type BatchWorkerPool interface {
	Spawn(ctx context.Context, n int)
	Todo(b batch)
	Complete() (batch, error)
}

func DefaultNewWorker(p p2p.P2P) NewWorker {
	return func(id workerId, todo, done chan batch) Worker {
		return newP2pWorker(id, p, todo, done)
	}
}

func NewP2PBatchWorkerPool(p p2p.P2P) *p2pBatchWorkerPool {
	nw := DefaultNewWorker(p)
	return &p2pBatchWorkerPool{
		newWorker: nw,
		todo:      make(chan batch),
		complete:  make(chan batch),
		workers:   make(map[workerId]Worker),
	}
}

type p2pBatchWorkerPool struct {
	ctx       context.Context
	cancel    func()
	p2p       p2p.P2P
	todo      chan batch
	complete  chan batch
	workers   map[workerId]Worker
	endSeq    []batch
	newWorker NewWorker
}

var _ BatchWorkerPool = &p2pBatchWorkerPool{}

func (p *p2pBatchWorkerPool) Todo(b batch) {
	// Intercept batchEndSequence batches so workers can remain unaware of this state.
	// Workers don't know what to do with batchEndSequence batches. They are a signal to the pool that the batcher
	// has stopped producing things for the workers to do and the pool is close to winding down. See Complete()
	// to understand how the pool manages the state where all workers are idle
	// and all incoming batches signal end of sequence.
	if b.state == batchEndSequence {
		p.endSeq = append(p.endSeq, b)
		return
	}
	p.todo <- b
}

func (p *p2pBatchWorkerPool) Complete() (batch, error) {
	nw := len(p.workers)
	if len(p.endSeq) == nw && nw == len(p.idle()) {
		return p.endSeq[0], errEndSequence
	}
	select {
	case b := <-p.complete:
		return b, nil
	case <-p.ctx.Done():
		return batch{}, p.ctx.Err()
	}
}

func (p *p2pBatchWorkerPool) Spawn(ctx context.Context, n int) {
	p.ctx, p.cancel = context.WithCancel(ctx)
	idm := len(p.workers)
	for i := 0; i < n; i++ {
		id := workerId(i + idm)
		p.workers[id] = p.newWorker(id, p.todo, p.complete)
		go p.workers[id].run(p.ctx)
	}
}

func (p *p2pBatchWorkerPool) idle() []workerId {
	ids := make([]workerId, 0)
	for id, w := range p.workers {
		if w.state() == workerIdle {
			ids = append(ids, id)
		}
	}
	return ids
}
