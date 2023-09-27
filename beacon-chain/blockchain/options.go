package blockchain

import (
	"github.com/prysmaticlabs/prysm/v4/async/event"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/execution"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/blstoexec"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stategen"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type Option func(s *Service) error

// WithMaxGoroutines to control resource use of the blockchain service.
func WithMaxGoroutines(x int) Option {
	return func(s *Service) error {
		s.cfg.MaxRoutines = x
		return nil
	}
}

// WithWeakSubjectivityCheckpoint for checkpoint sync.
func WithWeakSubjectivityCheckpoint(c *ethpb.Checkpoint) Option {
	return func(s *Service) error {
		s.cfg.WeakSubjectivityCheckpt = c
		return nil
	}
}

// WithDatabase for head access.
func WithDatabase(beaconDB db.HeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.BeaconDB = beaconDB
		return nil
	}
}

// WithChainStartFetcher to retrieve information about genesis.
func WithChainStartFetcher(f execution.ChainStartFetcher) Option {
	return func(s *Service) error {
		s.cfg.ChainStartFetcher = f
		return nil
	}
}

// WithExecutionEngineCaller to call execution engine.
func WithExecutionEngineCaller(c execution.EngineCaller) Option {
	return func(s *Service) error {
		s.cfg.ExecutionEngineCaller = c
		return nil
	}
}

// WithDepositCache for deposit lifecycle after chain inclusion.
func WithDepositCache(c cache.DepositCache) Option {
	return func(s *Service) error {
		s.cfg.DepositCache = c
		return nil
	}
}

// WithProposerIdsCache for proposer id cache.
func WithProposerIdsCache(c *cache.ProposerPayloadIDsCache) Option {
	return func(s *Service) error {
		s.cfg.ProposerSlotIndexCache = c
		return nil
	}
}

// WithAttestationPool for attestation lifecycle after chain inclusion.
func WithAttestationPool(p attestations.Pool) Option {
	return func(s *Service) error {
		s.cfg.AttPool = p
		return nil
	}
}

// WithExitPool for exits lifecycle after chain inclusion.
func WithExitPool(p voluntaryexits.PoolManager) Option {
	return func(s *Service) error {
		s.cfg.ExitPool = p
		return nil
	}
}

// WithSlashingPool for slashings lifecycle after chain inclusion.
func WithSlashingPool(p slashings.PoolManager) Option {
	return func(s *Service) error {
		s.cfg.SlashingPool = p
		return nil
	}
}

// WithBLSToExecPool to keep track of BLS to Execution address changes.
func WithBLSToExecPool(p blstoexec.PoolManager) Option {
	return func(s *Service) error {
		s.cfg.BLSToExecPool = p
		return nil
	}
}

// WithP2PBroadcaster to broadcast messages after appropriate processing.
func WithP2PBroadcaster(p p2p.Broadcaster) Option {
	return func(s *Service) error {
		s.cfg.P2p = p
		return nil
	}
}

// WithStateNotifier to notify an event feed of state processing.
func WithStateNotifier(n statefeed.Notifier) Option {
	return func(s *Service) error {
		s.cfg.StateNotifier = n
		return nil
	}
}

// WithForkChoiceStore to update an optimized fork-choice representation.
func WithForkChoiceStore(f forkchoice.ForkChoicer) Option {
	return func(s *Service) error {
		s.cfg.ForkChoiceStore = f
		return nil
	}
}

// WithAttestationService for dealing with attestation lifecycles.
func WithAttestationService(srv *attestations.Service) Option {
	return func(s *Service) error {
		s.cfg.AttService = srv
		return nil
	}
}

// WithStateGen for managing state regeneration and replay.
func WithStateGen(g *stategen.State) Option {
	return func(s *Service) error {
		s.cfg.StateGen = g
		return nil
	}
}

// WithSlasherAttestationsFeed to forward attestations into slasher if enabled.
func WithSlasherAttestationsFeed(f *event.Feed) Option {
	return func(s *Service) error {
		s.cfg.SlasherAttestationsFeed = f
		return nil
	}
}

// WithFinalizedStateAtStartUp to store finalized state at start up.
func WithFinalizedStateAtStartUp(st state.BeaconState) Option {
	return func(s *Service) error {
		s.cfg.FinalizedStateAtStartUp = st
		return nil
	}
}

func WithClockSynchronizer(gs *startup.ClockSynchronizer) Option {
	return func(s *Service) error {
		s.clockSetter = gs
		s.clockWaiter = gs
		return nil
	}
}

func WithSyncComplete(c chan struct{}) Option {
	return func(s *Service) error {
		s.syncComplete = c
		return nil
	}
}
