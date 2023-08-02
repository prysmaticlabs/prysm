package backfill

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/runtime"
	log "github.com/sirupsen/logrus"
)

const defaultWorkerCount = 1

type Service struct {
	ctx         context.Context
	clockWaiter startup.ClockWaiter
	clock       *startup.Clock
	su          *StatusUpdater
	p2p         p2p.P2P
	nWorkers    int
	todo        chan batch
	done        chan batch
	errChan     chan error
	workers     map[workerId]*p2pWorker
	batchSeq    *batchSequencer
	batchSize   uint64
}

var _ runtime.Service = (*Service)(nil)

type ServiceOption func(*Service) error

func WithStatusUpdater(su *StatusUpdater) ServiceOption {
	return func(s *Service) error {
		s.su = su
		return nil
	}
}

func WithClockWaiter(gw startup.ClockWaiter) ServiceOption {
	return func(s *Service) error {
		s.clockWaiter = gw
		return nil
	}
}

func WithP2P(p p2p.P2P) ServiceOption {
	return func(s *Service) error {
		s.p2p = p
		return nil
	}
}

func WithWorkerCount(n int) ServiceOption {
	return func(s *Service) error {
		s.nWorkers = n
		return nil
	}
}

func WithBatchSize(n uint64) ServiceOption {
	return func(s *Service) error {
		s.batchSize = n
		return nil
	}
}

func NewService(ctx context.Context, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		ctx: ctx,
	}
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	if s.nWorkers == 0 {
		s.nWorkers = defaultWorkerCount
	}
	if s.todo == nil {
		s.todo = make(chan batch)
	}
	if s.done == nil {
		s.done = make(chan batch)
	}
	return s, nil
}

func (s *Service) Start() {
	var err error
	s.clock, err = s.clockWaiter.WaitForClock(s.ctx)
	if err != nil {
		log.WithError(err).Error("backfill service failed to start while waiting for genesis data")
	}
	status := s.su.Status()
	s.batchSeq = newBatchSequencer(s.nWorkers, primitives.Slot(status.LowSlot), primitives.Slot(status.HighSlot), primitives.Slot(s.batchSize))
	err = s.spawnWorkers()
	if err != nil {
		log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
	}
	for {
		select {
		case b := <-s.done:
			s.batchSeq.update(b)
			importable := s.batchSeq.importable()
			for i := range importable {
				ib := importable[i]
				if err := s.importBatch(ib); err != nil {
					s.downscore(ib)
					ib.state = batchErrRetryable
					s.batchSeq.update(b)
					break
				}
				ib.state = batchImportComplete
				s.batchSeq.update(b)
			}
			b, err = s.batchSeq.sequence()
			if err != nil {
				if !errors.Is(err, errEndSequence) {
					log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
				}
			}
			s.todo <- b
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) downscore(b batch) {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}

func (s *Service) importBatch(b batch) error {
	return nil
}

func (s *Service) spawnWorkers() error {
	for i := 0; i < s.nWorkers; i++ {
		id := workerId(i)
		s.workers[id] = newP2pWorker(id, s.p2p, s.todo, s.done)
		go s.workers[id].run(s.ctx)
		b, err := s.batchSeq.sequence()
		if err != nil {
			// don't bother spawning workers if all batches have already been generated.
			if errors.Is(err, errEndSequence) {
				return nil
			}
			return err
		}
		s.todo <- b
	}
	return nil
}
