package light

import (
	"context"
	"fmt"
	"sync"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	syncSrv "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	block2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

type UpdatesFetcher interface {
	BestUpdateForPeriod(ctx context.Context, period uint64) (*ethpb.LightClientUpdate, error)
	LatestFinalizedUpdate(ctx context.Context) *ethpb.LightClientUpdate
	LatestNonFinalizedUpdate(ctx context.Context) *ethpb.LightClientUpdate
}

type Config struct {
	StateGen            stategen.StateManager
	Database            iface.Database
	HeadFetcher         blockchain.HeadFetcher
	FinalizationFetcher blockchain.FinalizationFetcher
	StateNotifier       statefeed.Notifier
	TimeFetcher         blockchain.TimeFetcher
	SyncChecker         syncSrv.Checker
}

type Service struct {
	cfg                      *Config
	cancelFunc               context.CancelFunc
	prevHeadData             map[[32]byte]*ethpb.SyncAttestedData
	lock                     sync.RWMutex
	genesisTime              time.Time
	finalizedByEpoch         map[types.Epoch]*ethpb.LightClientFinalizedCheckpoint
	bestUpdateByPeriod       map[uint64]*ethpb.LightClientUpdate
	latestFinalizedUpdate    *ethpb.LightClientUpdate
	latestNonFinalizedUpdate *ethpb.LightClientUpdate
}

// New --
func New(ctx context.Context, cfg *Config) *Service {
	return &Service{
		cfg:                cfg,
		prevHeadData:       make(map[[32]byte]*ethpb.SyncAttestedData),
		finalizedByEpoch:   make(map[types.Epoch]*ethpb.LightClientFinalizedCheckpoint),
		bestUpdateByPeriod: make(map[uint64]*ethpb.LightClientUpdate),
	}
}

func (s *Service) Start() {
	go s.run()
}

func (s *Service) Stop() error {
	s.cancelFunc()
	return nil
}

func (s *Service) Status() error {
	return nil
}

func (s *Service) BestUpdateForPeriod(ctx context.Context, period uint64) (*ethpb.LightClientUpdate, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	update, ok := s.bestUpdateByPeriod[period]
	if !ok {
		return nil, fmt.Errorf("no update found for period %d", period)
	}
	return update, nil
}

func (s *Service) LatestFinalizedUpdate(ctx context.Context) *ethpb.LightClientUpdate {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.latestFinalizedUpdate
}

func (s *Service) LatestNonFinalizedUpdate(ctx context.Context) *ethpb.LightClientUpdate {
	return s.latestNonFinalizedUpdate
}

func (s *Service) run() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	s.waitForChainInitialization(ctx)
	s.waitForSync(ctx)
	// Initialize the service from finalized (state, block) data.
	log.Info("Initializing from finalized data")
	if err := s.initializeFromFinalizedData(ctx); err != nil {
		log.Fatal(err)
	}
	log.Info("Beginning subscriptions")
	// Begin listening for new chain head and finalized checkpoint events.
	go s.subscribeHeadEvent(ctx)
	go s.subscribeFinalizedEvent(ctx)
}

func (s *Service) waitForChainInitialization(ctx context.Context) {
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	defer close(stateChannel)
	for {
		select {
		case stateEvent := <-stateChannel:
			// Wait for us to receive the genesis time via a chain started notification.
			if stateEvent.Type == statefeed.Initialized {
				// Alternatively, if the chain has already started, we then read the genesis
				// time value from this data.
				data, ok := stateEvent.Data.(*statefeed.InitializedData)
				if !ok {
					log.Error(
						"Could not receive chain start notification, want *statefeed.ChainStartedData",
					)
					return
				}
				s.genesisTime = data.StartTime
				log.WithField("genesisTime", s.genesisTime).Info(
					"Received chain initialization event",
				)
				return
			}
		case err := <-stateSub.Err():
			log.WithError(err).Error(
				"Could not subscribe to state events",
			)
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) waitForSync(ctx context.Context) {
	slotTicker := slots.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	defer slotTicker.Done()
	for {
		select {
		case <-slotTicker.C():
			if slots.ToEpoch(slots.SinceGenesis(s.genesisTime)) < 6 {
				continue
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) finalizedBlockOrGenesis(ctx context.Context, cpt *ethpb.Checkpoint) (block2.SignedBeaconBlock, error) {
	checkpointRoot := bytesutil.ToBytes32(cpt.Root)
	block, err := s.cfg.Database.Block(ctx, checkpointRoot)
	if err != nil {
		return nil, err
	}
	if block == nil || block.IsNil() {
		return s.cfg.Database.GenesisBlock(ctx)
	}
	return block, nil
}

func (s *Service) finalizedStateOrGenesis(ctx context.Context, cpt *ethpb.Checkpoint) (state.BeaconState, error) {
	checkpointRoot := bytesutil.ToBytes32(cpt.Root)
	st, err := s.cfg.StateGen.StateByRoot(ctx, checkpointRoot)
	if err != nil {
		return nil, err
	}
	if st == nil || st.IsNil() {
		return s.cfg.Database.GenesisState(ctx)
	}
	return st, nil
}

func (s *Service) initializeFromFinalizedData(ctx context.Context) error {
	log.Info("Getting finalized checkpoint")
	cpt, err := s.cfg.Database.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	log.Info("Getting finalized block or genesis")
	finalizedBlock, err := s.finalizedBlockOrGenesis(ctx, cpt)
	if err != nil {
		return err
	}
	log.Info("Getting finalized state or genesis")
	finalizedState, err := s.finalizedStateOrGenesis(ctx, cpt)
	if err != nil {
		return err
	}
	log.Info("Calling onFinalized")
	return s.onFinalized(ctx, finalizedBlock, finalizedState)
}
