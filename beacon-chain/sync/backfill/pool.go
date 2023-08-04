package backfill

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
)

type BatchWorkerPool interface {
	Spawn(n int)
	Todo(b batch)
	Finished() (batch, error)
}

func NewP2PBatchWorkerPool(ctx context.Context, p p2p.P2P) *p2pBatchWorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	return &p2pBatchWorkerPool{
		ctx:      ctx,
		cancel:   cancel,
		p2p:      p,
		todo:     make(chan batch),
		finished: make(chan batch),
		workers:  make(map[workerId]*p2pWorker),
	}
}

type p2pBatchWorkerPool struct {
	ctx      context.Context
	cancel   func()
	p2p      p2p.P2P
	todo     chan batch
	finished chan batch
	workers  map[workerId]*p2pWorker
}

func (p *p2pBatchWorkerPool) Todo(b batch) {
	p.todo <- b
}

func (p *p2pBatchWorkerPool) Finished() (batch, error) {
	select {
	case b := <-p.finished:
		return b, nil
	case <-p.ctx.Done():
		return batch{}, p.ctx.Err()
	}
}

var _ BatchWorkerPool = &p2pBatchWorkerPool{}

func (p *p2pBatchWorkerPool) Spawn(n int) {
	idm := len(p.workers)
	for i := 0; i < n; i++ {
		id := workerId(i + idm)
		p.workers[id] = newP2pWorker(id, p.p2p, p.todo, p.finished)
		go p.workers[id].run(p.ctx)
	}
}
