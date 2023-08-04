package backfill

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/runtime"
	log "github.com/sirupsen/logrus"
)

const defaultWorkerCount = 1

// TODO use the correct beacon param for blocks by range size instead
const defaultBatchSize = 64

type Service struct {
	ctx         context.Context
	clockWaiter startup.ClockWaiter
	clock       *startup.Clock
	su          *StatusUpdater
	nWorkers    int
	errChan     chan error
	batchSeq    *batchSequencer
	batchSize   uint64
	pool        BatchWorkerPool
	initialized chan struct{}
	exited      chan struct{}
}

var _ runtime.Service = (*Service)(nil)

type ServiceOption func(*Service) error

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

func NewService(ctx context.Context, su *StatusUpdater, cw startup.ClockWaiter, pool BatchWorkerPool, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		ctx:         ctx,
		su:          su,
		clockWaiter: cw,
		pool:        pool,
		initialized: make(chan struct{}),
		exited:      make(chan struct{}),
	}
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	if s.nWorkers == 0 {
		s.nWorkers = defaultWorkerCount
	}
	return s, nil
}

func (s *Service) Start() {
	defer close(s.exited)
	var err error
	s.clock, err = s.clockWaiter.WaitForClock(s.ctx)
	if err != nil {
		log.WithError(err).Error("backfill service failed to start while waiting for genesis data")
	}

	status := s.su.Status()
	s.batchSeq = newBatchSequencer(s.nWorkers, primitives.Slot(status.LowSlot), primitives.Slot(status.HighSlot), primitives.Slot(s.batchSize))
	s.pool.Spawn(s.nWorkers)

	if err = s.initBatches(); err != nil {
		log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
	}

	close(s.initialized)
	for {
		b, err := s.pool.Finished()
		if err != nil {
			log.WithError(err).Error("Non-recoverable error in backfill service, quitting.")
			return
		}
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
			s.batchSeq.update(ib)
		}
		b, err = s.batchSeq.sequence()
		if err != nil {
			if !errors.Is(err, errEndSequence) {
				log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
			}
		}
		// We want to update the pool with the batchEndSequence batches so that the pool can detect when all batches
		// are finished.
		s.pool.Todo(b)
	}
}

func (s *Service) initBatches() error {
	for i := 0; i < s.nWorkers; i++ {
		b, err := s.batchSeq.sequence()
		if err != nil {
			if errors.Is(err, errEndSequence) {
				return nil
			}
			return err
		}

		s.pool.Todo(b)
	}
	return nil
}

func (s *Service) importBatch(b batch) error {
	return nil
}

func (s *Service) downscore(b batch) {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}
