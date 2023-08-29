package backfill

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/runtime"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

const defaultWorkerCount = 1

// TODO use the correct beacon param for blocks by range size instead
const defaultBatchSize = 64

type Service struct {
	ctx           context.Context
	su            *StatusUpdater
	ms            minimumSlotter
	cw            startup.ClockWaiter
	nWorkers      int
	errChan       chan error
	batchSeq      *batchSequencer
	batchSize     uint64
	pool          BatchWorkerPool
	verifier      *verifier
	initialized   chan struct{}
	exited        chan struct{}
	p2p           p2p.P2P
	batchImporter batchImporter
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

type minimumSlotter interface {
	minimumSlot() primitives.Slot
	setClock(*startup.Clock)
}

type defaultMinimumSlotter struct {
	clock *startup.Clock
	cw    startup.ClockWaiter
}

func (d defaultMinimumSlotter) minimumSlot() primitives.Slot {
	return MinimumBackfillSlot(d.clock.CurrentSlot())
}

func (d defaultMinimumSlotter) setClock(c *startup.Clock) {
	d.clock = c
}

var _ minimumSlotter = &defaultMinimumSlotter{}

type batchImporter func(ctx context.Context, b batch, su *StatusUpdater) error

func defaultBatchImporter(ctx context.Context, b batch, su *StatusUpdater) error {
	status := su.Status()
	if err := b.ensureParent(bytesutil.ToBytes32(status.LowParentRoot)); err != nil {
		return err
	}
	/*
		for _, b := range b.results {
			// TODO exposed block saving through su
		}
	*/
	// Update db state to reflect the newly imported blocks. Other parts of the beacon node may look at the
	// backfill status to determine if a range of blocks is available.
	if err := su.fillBack(ctx, b.lowest()); err != nil {
		log.WithError(err).Fatal("Non-recoverable db error in backfill service, quitting.")
	}
	return nil
}

func NewService(ctx context.Context, su *StatusUpdater, cw startup.ClockWaiter, p p2p.P2P, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		ctx:           ctx,
		su:            su,
		cw:            cw,
		ms:            &defaultMinimumSlotter{cw: cw},
		initialized:   make(chan struct{}),
		exited:        make(chan struct{}),
		p2p:           p,
		batchImporter: defaultBatchImporter,
	}
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	if s.nWorkers == 0 {
		s.nWorkers = defaultWorkerCount
	}
	if s.batchSize == 0 {
		s.batchSize = defaultBatchSize
	}
	s.pool = newP2PBatchWorkerPool(p, s.nWorkers)

	return s, nil
}

func (s *Service) initVerifier(ctx context.Context) (*verifier, error) {
	cps, err := s.su.originState(ctx)
	if err != nil {
		return nil, err
	}
	return newBackfillVerifier(cps)
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer func() {
		cancel()
		close(s.exited)
	}()
	clock, err := s.cw.WaitForClock(ctx)
	if err != nil {
		log.WithError(err).Fatal("backfill service failed to start while waiting for genesis data")
	}
	s.ms.setClock(clock)

	status := s.su.Status()
	s.batchSeq = newBatchSequencer(s.nWorkers, s.ms.minimumSlot(), primitives.Slot(status.LowSlot), primitives.Slot(s.batchSize))
	// Exit early if there aren't going to be any batches to backfill.
	if s.batchSeq.numTodo() == 0 {
		return
	}
	originE := slots.ToEpoch(primitives.Slot(status.OriginSlot))
	assigner := peers.NewAssigner(ctx, s.p2p.Peers(), params.BeaconConfig().MaxPeersToSync, originE)
	s.verifier, err = s.initVerifier(ctx)
	if err != nil {
		log.WithError(err).Fatal("Unable to initialize backfill verifier, quitting.")
	}
	s.pool.Spawn(ctx, s.nWorkers, clock, assigner, s.verifier)

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
				log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
			}
			return
		}
		s.batchSeq.update(b)
		importable := s.batchSeq.importable()
		for i := range importable {
			ib := importable[i]
			if err := s.batchImporter(ctx, ib, s.su); err != nil {
				s.downscore(ib)
				ib.state = batchErrRetryable
				s.batchSeq.update(b)
				break
			}
			ib.state = batchImportComplete
			// Calling update with state=batchImportComplete will advance the batch list.
			s.batchSeq.update(ib)
		}
		if err := s.batchSeq.moveMinimum(s.ms.minimumSlot()); err != nil {
			log.WithError(err).Fatal("Non-recoverable error in backfill service, quitting.")
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

func (s *Service) downscore(b batch) {
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() error {
	return nil
}

// MinimumBackfillSlot determines the lowest slot that backfill needs to download based on looking back
// MIN_EPOCHS_FOR_BLOCK_REQUESTS from the current slot.
func MinimumBackfillSlot(current primitives.Slot) primitives.Slot {
	oe := helpers.MinEpochsForBlockRequests()
	if oe > slots.MaxSafeEpoch() {
		oe = slots.MaxSafeEpoch()
	}
	offset := slots.UnsafeEpochStart(oe)
	if offset > current {
		return 0
	}
	return current - offset
}
