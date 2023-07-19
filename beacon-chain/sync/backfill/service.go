package backfill

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/runtime"
	log "github.com/sirupsen/logrus"
)

const defaultWorkerCount = 1

type Service struct {
	ctx           context.Context
	genesisWaiter startup.ClockWaiter
	clock         *startup.Clock
	su            *StatusUpdater
	db            BackfillDB
	p2p           p2p.P2P
	nWorkers      int
	todo          chan batch
	done          chan batch
	errChan       chan error
	workers       map[workerId]*p2pWorker
	batcher       *batcher
	batchSize     uint64
}

var _ runtime.Service = (*Service)(nil)

type ServiceOption func(*Service) error

func WithStatusUpdater(su *StatusUpdater) ServiceOption {
	return func(s *Service) error {
		s.su = su
		return nil
	}
}

func WithGenesisWaiter(gw startup.ClockWaiter) ServiceOption {
	return func(s *Service) error {
		s.genesisWaiter = gw
		return nil
	}
}

func WithBackfillDB(db BackfillDB) ServiceOption {
	return func(s *Service) error {
		s.db = db
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
	clock, err := s.genesisWaiter.WaitForClock(s.ctx)
	if err != nil {
		log.WithError(err).Error("backfill service failed to start while waiting for genesis data")
	}
	s.clock = clock
	if err := s.spawnBatcher(); err != nil {
		log.WithError(err).Fatal("error starting backfill service")
	}
	s.spawnWorkers()
	for {
		select {
		case <-s.ctx.Done():
			return
		case err := <-s.errChan:
			if err := s.tryRecover(err); err != nil {
				log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
			}
		}
	}
}

func (s *Service) tryRecover(err error) error {
	log.WithError(err).Error("error from the batcher")
	// If error is not recoverable, reply with an error, which will shut down the service.
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}

func (s *Service) spawnWorkers() {
	for i := 0; i < s.nWorkers; i++ {
		id := workerId(i)
		s.workers[id] = newP2pWorker(id, s.p2p, s.todo, s.done)
		go s.workers[id].run(s.ctx)
	}
}

func (s *Service) spawnBatcher() error {
	s.batcher = newBatcher(primitives.Slot(s.batchSize), s.su, s.todo, s.done)
	go s.batcher.run(s.ctx)
	return nil
}
