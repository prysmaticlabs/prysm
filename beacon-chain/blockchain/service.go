// Package blockchain defines the life-cycle of the blockchain at the core of
// eth2, including processing of new blocks and attestations using casper
// proof of stake.
package blockchain

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	f "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"go.opencensus.io/trace"
)

// Service represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type Service struct {
	ctx                       context.Context
	cancel                    context.CancelFunc
	beaconDB                  db.HeadAccessDatabase
	depositCache              *depositcache.DepositCache
	chainStartFetcher         powchain.ChainStartFetcher
	attPool                   attestations.Pool
	slashingPool              *slashings.Pool
	exitPool                  *voluntaryexits.Pool
	genesisTime               time.Time
	p2p                       p2p.Broadcaster
	maxRoutines               int64
	head                      *head
	headLock                  sync.RWMutex
	stateNotifier             statefeed.Notifier
	genesisRoot               [32]byte
	epochParticipation        map[uint64]*precompute.Balance
	epochParticipationLock    sync.RWMutex
	forkChoiceStore           f.ForkChoicer
	justifiedCheckpt          *ethpb.Checkpoint
	prevJustifiedCheckpt      *ethpb.Checkpoint
	bestJustifiedCheckpt      *ethpb.Checkpoint
	finalizedCheckpt          *ethpb.Checkpoint
	prevFinalizedCheckpt      *ethpb.Checkpoint
	nextEpochBoundarySlot     uint64
	voteLock                  sync.RWMutex
	initSyncState             map[[32]byte]*stateTrie.BeaconState
	boundaryRoots             [][32]byte
	initSyncStateLock         sync.RWMutex
	checkpointState           *cache.CheckpointStateCache
	checkpointStateLock       sync.Mutex
	stateGen                  *stategen.State
	opsService                *attestations.Service
	initSyncBlocks            map[[32]byte]*ethpb.SignedBeaconBlock
	initSyncBlocksLock        sync.RWMutex
	recentCanonicalBlocks     map[[32]byte]bool
	recentCanonicalBlocksLock sync.RWMutex
	justifiedBalances         []uint64
	justifiedBalancesLock     sync.RWMutex
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf    int
	ChainStartFetcher powchain.ChainStartFetcher
	BeaconDB          db.HeadAccessDatabase
	DepositCache      *depositcache.DepositCache
	AttPool           attestations.Pool
	ExitPool          *voluntaryexits.Pool
	SlashingPool      *slashings.Pool
	P2p               p2p.Broadcaster
	MaxRoutines       int64
	StateNotifier     statefeed.Notifier
	ForkChoiceStore   f.ForkChoicer
	OpsService        *attestations.Service
	StateGen          *stategen.State
}

// NewService instantiates a new block service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		beaconDB:              cfg.BeaconDB,
		depositCache:          cfg.DepositCache,
		chainStartFetcher:     cfg.ChainStartFetcher,
		attPool:               cfg.AttPool,
		exitPool:              cfg.ExitPool,
		slashingPool:          cfg.SlashingPool,
		p2p:                   cfg.P2p,
		maxRoutines:           cfg.MaxRoutines,
		stateNotifier:         cfg.StateNotifier,
		epochParticipation:    make(map[uint64]*precompute.Balance),
		forkChoiceStore:       cfg.ForkChoiceStore,
		initSyncState:         make(map[[32]byte]*stateTrie.BeaconState),
		boundaryRoots:         [][32]byte{},
		checkpointState:       cache.NewCheckpointStateCache(),
		opsService:            cfg.OpsService,
		stateGen:              cfg.StateGen,
		initSyncBlocks:        make(map[[32]byte]*ethpb.SignedBeaconBlock),
		recentCanonicalBlocks: make(map[[32]byte]bool),
		justifiedBalances:     make([]uint64, 0),
	}, nil
}

// Start a blockchain service's main event loop.
func (s *Service) Start() {
	ctx := context.TODO()
	beaconState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		log.Fatalf("Could not fetch beacon state: %v", err)
	}

	// For running initial sync with state cache, in an event of restart, we use
	// last finalized check point as start point to sync instead of head
	// state. This is because we no longer save state every slot during sync.
	cp, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		log.Fatalf("Could not fetch finalized cp: %v", err)
	}
	if beaconState == nil {
		beaconState, err = s.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(cp.Root))
		if err != nil {
			log.Fatalf("Could not fetch beacon state by root: %v", err)
		}
	}

	// Make sure that attestation processor is subscribed and ready for state initializing event.
	attestationProcessorSubscribed := make(chan struct{}, 1)

	// If the chain has already been initialized, simply start the block processing routine.
	if beaconState != nil {
		log.Info("Blockchain data already exists in DB, initializing...")
		s.genesisTime = time.Unix(int64(beaconState.GenesisTime()), 0)
		s.opsService.SetGenesisTime(beaconState.GenesisTime())
		if err := s.initializeChainInfo(ctx); err != nil {
			log.Fatalf("Could not set up chain info: %v", err)
		}

		// We start a counter to genesis, if needed.
		go slotutil.CountdownToGenesis(ctx, s.genesisTime, uint64(beaconState.NumValidators()))

		justifiedCheckpoint, err := s.beaconDB.JustifiedCheckpoint(ctx)
		if err != nil {
			log.Fatalf("Could not get justified checkpoint: %v", err)
		}
		finalizedCheckpoint, err := s.beaconDB.FinalizedCheckpoint(ctx)
		if err != nil {
			log.Fatalf("Could not get finalized checkpoint: %v", err)
		}

		// Resume fork choice.
		s.justifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		if err := s.cacheJustifiedStateBalances(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root)); err != nil {
			log.Fatalf("Could not cache justified state balances: %v", err)
		}
		s.prevJustifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		s.bestJustifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		s.finalizedCheckpt = stateTrie.CopyCheckpoint(finalizedCheckpoint)
		s.prevFinalizedCheckpt = stateTrie.CopyCheckpoint(finalizedCheckpoint)
		s.resumeForkChoice(justifiedCheckpoint, finalizedCheckpoint)

		s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime:             s.genesisTime,
				GenesisValidatorsRoot: beaconState.GenesisValidatorRoot(),
			},
		})
	} else {
		log.Info("Waiting to reach the validator deposit threshold to start the beacon chain...")
		if s.chainStartFetcher == nil {
			log.Fatal("Not configured web3Service for POW chain")
			return // return need for TestStartUninitializedChainWithoutConfigPOWChain.
		}
		go func() {
			stateChannel := make(chan *feed.Event, 1)
			stateSub := s.stateNotifier.StateFeed().Subscribe(stateChannel)
			defer stateSub.Unsubscribe()
			<-attestationProcessorSubscribed
			for {
				select {
				case event := <-stateChannel:
					if event.Type == statefeed.ChainStarted {
						data, ok := event.Data.(*statefeed.ChainStartedData)
						if !ok {
							log.Error("event data is not type *statefeed.ChainStartedData")
							return
						}
						log.WithField("starttime", data.StartTime).Debug("Received chain start event")
						s.processChainStartTime(ctx, data.StartTime)
						return
					}
				case <-s.ctx.Done():
					log.Debug("Context closed, exiting goroutine")
					return
				case err := <-stateSub.Err():
					log.WithError(err).Error("Subscription to state notifier failed")
					return
				}
			}
		}()
	}

	go s.processAttestation(attestationProcessorSubscribed)
}

// processChainStartTime initializes a series of deposits from the ChainStart deposits in the eth1
// deposit contract, initializes the beacon chain's state, and kicks off the beacon chain.
func (s *Service) processChainStartTime(ctx context.Context, genesisTime time.Time) {
	preGenesisState := s.chainStartFetcher.PreGenesisState()
	initializedState, err := s.initializeBeaconChain(ctx, genesisTime, preGenesisState, s.chainStartFetcher.ChainStartEth1Data())
	if err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	// We start a counter to genesis, if needed.
	go slotutil.CountdownToGenesis(ctx, genesisTime, uint64(initializedState.NumValidators()))

	// We send out a state initialized event to the rest of the services
	// running in the beacon node.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime:             genesisTime,
			GenesisValidatorsRoot: initializedState.GenesisValidatorRoot(),
		},
	})
}

// initializes the state and genesis block of the beacon chain to persistent storage
// based on a genesis timestamp value obtained from the ChainStart event emitted
// by the ETH1.0 Deposit Contract and the POWChain service of the node.
func (s *Service) initializeBeaconChain(
	ctx context.Context,
	genesisTime time.Time,
	preGenesisState *stateTrie.BeaconState,
	eth1data *ethpb.Eth1Data) (*stateTrie.BeaconState, error) {
	_, span := trace.StartSpan(context.Background(), "beacon-chain.Service.initializeBeaconChain")
	defer span.End()
	s.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())

	genesisState, err := state.OptimizedGenesisBeaconState(unixTime, preGenesisState, eth1data)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize genesis state")
	}

	if err := s.saveGenesisData(ctx, genesisState); err != nil {
		return nil, errors.Wrap(err, "could not save genesis data")
	}

	log.Info("Initialized beacon chain genesis state")

	// Clear out all pre-genesis data now that the state is initialized.
	s.chainStartFetcher.ClearPreGenesisData()

	// Update committee shuffled indices for genesis epoch.
	if err := helpers.UpdateCommitteeCache(genesisState, 0 /* genesis epoch */); err != nil {
		return nil, err
	}
	if err := helpers.UpdateProposerIndicesInCache(genesisState, 0 /* genesis epoch */); err != nil {
		return nil, err
	}

	s.opsService.SetGenesisTime(genesisState.GenesisTime())

	return genesisState, nil
}

// Stop the blockchain service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status always returns nil unless there is an error condition that causes
// this service to be unhealthy.
func (s *Service) Status() error {
	if int64(runtime.NumGoroutine()) > s.maxRoutines {
		return fmt.Errorf("too many goroutines %d", runtime.NumGoroutine())
	}
	return nil
}

// ClearCachedStates removes all stored caches states. This is done after the node
// is synced.
func (s *Service) ClearCachedStates() {
	s.initSyncState = map[[32]byte]*stateTrie.BeaconState{}
}

// This gets called when beacon chain is first initialized to save genesis data (state, block, and more) in db.
func (s *Service) saveGenesisData(ctx context.Context, genesisState *stateTrie.BeaconState) error {
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}
	s.genesisRoot = genesisBlkRoot

	if err := s.beaconDB.SaveBlock(ctx, genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.beaconDB.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 0,
		Root: genesisBlkRoot[:],
	}); err != nil {
		return err
	}

	if err := s.beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis block root")
	}

	s.stateGen.SaveFinalizedState(0, genesisBlkRoot, genesisState)

	// Finalized checkpoint at genesis is a zero hash.
	genesisCheckpoint := genesisState.FinalizedCheckpoint()

	s.justifiedCheckpt = stateTrie.CopyCheckpoint(genesisCheckpoint)
	if err := s.cacheJustifiedStateBalances(ctx, genesisBlkRoot); err != nil {
		return err
	}
	s.prevJustifiedCheckpt = stateTrie.CopyCheckpoint(genesisCheckpoint)
	s.bestJustifiedCheckpt = stateTrie.CopyCheckpoint(genesisCheckpoint)
	s.finalizedCheckpt = stateTrie.CopyCheckpoint(genesisCheckpoint)
	s.prevFinalizedCheckpt = stateTrie.CopyCheckpoint(genesisCheckpoint)

	if err := s.forkChoiceStore.ProcessBlock(ctx,
		genesisBlk.Block.Slot,
		genesisBlkRoot,
		params.BeaconConfig().ZeroHash,
		[32]byte{},
		genesisCheckpoint.Epoch,
		genesisCheckpoint.Epoch); err != nil {
		log.Fatalf("Could not process genesis block for fork choice: %v", err)
	}

	s.setHead(genesisBlkRoot, genesisBlk, genesisState)

	return nil
}

// This gets called to initialize chain info variables using the finalized checkpoint stored in DB
func (s *Service) initializeChainInfo(ctx context.Context) error {
	genesisBlock, err := s.beaconDB.GenesisBlock(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block from db")
	}
	if genesisBlock == nil {
		return errors.New("no genesis block in db")
	}
	genesisBlkRoot, err := stateutil.BlockRoot(genesisBlock.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root of genesis block")
	}
	s.genesisRoot = genesisBlkRoot

	if flags.Get().UnsafeSync {
		headBlock, err := s.beaconDB.HeadBlock(ctx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head block")
		}
		headRoot, err := stateutil.BlockRoot(headBlock.Block)
		if err != nil {
			return errors.Wrap(err, "could not hash head block")
		}
		headState, err := s.beaconDB.HeadState(ctx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head state")
		}
		s.setHead(headRoot, headBlock, headState)
		return nil
	}

	finalized, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized checkpoint from db")
	}
	if finalized == nil {
		// This should never happen. At chain start, the finalized checkpoint
		// would be the genesis state and block.
		return errors.New("no finalized epoch in the database")
	}
	finalizedRoot := bytesutil.ToBytes32(finalized.Root)
	var finalizedState *stateTrie.BeaconState

	finalizedState, err = s.stateGen.Resume(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized state from db")
	}
	if !featureconfig.Get().SkipRegenHistoricalStates {
		// Since historical states were skipped, the node should start from last finalized check point.
		finalizedRoot = s.beaconDB.LastArchivedIndexRoot(ctx)
		if finalizedRoot == params.BeaconConfig().ZeroHash {
			finalizedRoot = bytesutil.ToBytes32(finalized.Root)
		}
	}

	finalizedBlock, err := s.beaconDB.Block(ctx, finalizedRoot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block from db")
	}

	// To skip the regeneration of historical state, the node has to generate the parent of the last finalized state.
	// We don't need to do this for genesis.
	atGenesis := s.CurrentSlot() == 0
	if featureconfig.Get().SkipRegenHistoricalStates && !atGenesis {
		parentRoot := bytesutil.ToBytes32(finalizedBlock.Block.ParentRoot)
		parentState, err := s.generateState(ctx, finalizedRoot, parentRoot)
		if err != nil {
			return err
		}
		if s.beaconDB.SaveState(ctx, parentState, parentRoot) != nil {
			return err
		}
	}

	if finalizedState == nil || finalizedBlock == nil {
		return errors.New("finalized state and block can't be nil")
	}
	s.setHead(finalizedRoot, finalizedBlock, finalizedState)

	return nil
}

// This is called when a client starts from non-genesis slot. This passes last justified and finalized
// information to fork choice service to initializes fork choice store.
func (s *Service) resumeForkChoice(justifiedCheckpoint *ethpb.Checkpoint, finalizedCheckpoint *ethpb.Checkpoint) {
	store := protoarray.New(justifiedCheckpoint.Epoch, finalizedCheckpoint.Epoch, bytesutil.ToBytes32(finalizedCheckpoint.Root))
	s.forkChoiceStore = store
}

// This returns true if block has been processed before. Two ways to verify the block has been processed:
// 1.) Check fork choice store.
// 2.) Check DB.
// Checking 1.) is ten times faster than checking 2.)
func (s *Service) hasBlock(ctx context.Context, root [32]byte) bool {
	if s.forkChoiceStore.HasNode(root) {
		return true
	}

	return s.beaconDB.HasBlock(ctx, root)
}
