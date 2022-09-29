package sync

import (
	"github.com/prysmaticlabs/prysm/v3/async/event"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
)

type Option func(s *Service) error

func WithAttestationNotifier(notifier operation.Notifier) Option {
	return func(s *Service) error {
		s.cfg.attestationNotifier = notifier
		return nil
	}
}

func WithP2P(p2p p2p.P2P) Option {
	return func(s *Service) error {
		s.cfg.p2p = p2p
		return nil
	}
}

func WithDatabase(db db.NoHeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.beaconDB = db
		return nil
	}
}

func WithAttestationPool(attPool attestations.Pool) Option {
	return func(s *Service) error {
		s.cfg.attPool = attPool
		return nil
	}
}

func WithExitPool(exitPool voluntaryexits.PoolManager) Option {
	return func(s *Service) error {
		s.cfg.exitPool = exitPool
		return nil
	}
}

func WithSlashingPool(slashingPool slashings.PoolManager) Option {
	return func(s *Service) error {
		s.cfg.slashingPool = slashingPool
		return nil
	}
}

func WithSyncCommsPool(syncCommsPool synccommittee.Pool) Option {
	return func(s *Service) error {
		s.cfg.syncCommsPool = syncCommsPool
		return nil
	}
}

func WithChainService(chain blockchainService) Option {
	return func(s *Service) error {
		s.cfg.chain = chain
		return nil
	}
}

func WithInitialSync(initialSync Checker) Option {
	return func(s *Service) error {
		s.cfg.initialSync = initialSync
		return nil
	}
}

func WithStateNotifier(stateNotifier statefeed.Notifier) Option {
	return func(s *Service) error {
		s.cfg.stateNotifier = stateNotifier
		return nil
	}
}

func WithBlockNotifier(blockNotifier blockfeed.Notifier) Option {
	return func(s *Service) error {
		s.cfg.blockNotifier = blockNotifier
		return nil
	}
}

func WithOperationNotifier(operationNotifier operation.Notifier) Option {
	return func(s *Service) error {
		s.cfg.operationNotifier = operationNotifier
		return nil
	}
}

func WithStateGen(stateGen *stategen.State) Option {
	return func(s *Service) error {
		s.cfg.stateGen = stateGen
		return nil
	}
}

func WithSlasherAttestationsFeed(slasherAttestationsFeed *event.Feed) Option {
	return func(s *Service) error {
		s.cfg.slasherAttestationsFeed = slasherAttestationsFeed
		return nil
	}
}

func WithSlasherBlockHeadersFeed(slasherBlockHeadersFeed *event.Feed) Option {
	return func(s *Service) error {
		s.cfg.slasherBlockHeadersFeed = slasherBlockHeadersFeed
		return nil
	}
}

func WithExecutionPayloadReconstructor(r execution.ExecutionPayloadReconstructor) Option {
	return func(s *Service) error {
		s.cfg.executionPayloadReconstructor = r
		return nil
	}
}
