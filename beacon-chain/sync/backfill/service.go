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
	ctx, cancel := context.WithCancel(s.ctx)
	defer func() {
		cancel()
		close(s.exited)
	}()
	var err error
	s.clock, err = s.clockWaiter.WaitForClock(ctx)
	if err != nil {
		log.WithError(err).Error("backfill service failed to start while waiting for genesis data")
	}

	status := s.su.Status()
	s.batchSeq = newBatchSequencer(s.nWorkers, primitives.Slot(status.LowSlot), primitives.Slot(status.HighSlot), primitives.Slot(s.batchSize))
	s.pool.Spawn(ctx, s.nWorkers)

	if err = s.initBatches(); err != nil {
		log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
	}

	close(s.initialized)
	for {
		b, err := s.pool.Complete()
		if err != nil {
			if errors.Is(err, errEndSequence) {
				log.WithField("backfill_slot", b.begin).Info("Backfill is complete")
			} else {
				log.WithError(err).Error("Non-recoverable error in backfill service, quitting.")
			}
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
		batches, err := s.batchSeq.sequence()
		if err != nil {
			// This typically means we have several importable batches, but they are stuck behind a batch that needs
			// to complete first so that we can chain parent roots across batches.
			// ie backfilling [[90..100), [80..90), [70..80)], if we complete [70..80) and [80..90) but not [90..100),
			// we can't move forward until [90..100) completes, because we need to confirm 99 connects to 100,
			// and then we'll have the parent_root expected by 90 to ensure it matches the root for 89,
			// at which point we know we can process [80..90).
			if errors.Is(err, errMaxBatches) {
				continue
			}
		}
		for _, b := range batches {
			s.pool.Todo(b)
		}
	}
}

func (s *Service) initBatches() error {
	batches, err := s.batchSeq.sequence()
	if err != nil {
		return err
	}
	for _, b := range batches {
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
