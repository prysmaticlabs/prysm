// Package blockchain defines the life-cycle of the blockchain at the core of
// eth2, including processing of new blocks and attestations using casper
// proof of stake.
package blockchain

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/orchestrator"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	f "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// headSyncMinEpochsAfterCheckpoint defines how many epochs should elapse after known finalization
// checkpoint for head sync to be triggered.
const headSyncMinEpochsAfterCheckpoint = 128

// Service represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	beaconDB              db.HeadAccessDatabase
	depositCache          *depositcache.DepositCache
	chainStartFetcher     powchain.ChainStartFetcher
	attPool               attestations.Pool
	slashingPool          slashings.PoolManager
	exitPool              voluntaryexits.PoolManager
	genesisTime           time.Time
	p2p                   p2p.Broadcaster
	maxRoutines           int
	head                  *head
	headLock              sync.RWMutex
	stateNotifier         statefeed.Notifier
	blockNotifier         blockfeed.Notifier
	genesisRoot           [32]byte
	forkChoiceStore       f.ForkChoicer
	justifiedCheckpt      *ethpb.Checkpoint
	prevJustifiedCheckpt  *ethpb.Checkpoint
	bestJustifiedCheckpt  *ethpb.Checkpoint
	finalizedCheckpt      *ethpb.Checkpoint
	prevFinalizedCheckpt  *ethpb.Checkpoint
	nextEpochBoundarySlot types.Slot
	boundaryRoots         [][32]byte
	checkpointStateCache  *cache.CheckpointStateCache
	stateGen              *stategen.State
	opsService            *attestations.Service
	initSyncBlocks        map[[32]byte]*ethpb.SignedBeaconBlock
	initSyncBlocksLock    sync.RWMutex
	justifiedBalances     []uint64
	justifiedBalancesLock sync.RWMutex
	wsEpoch               types.Epoch
	wsRoot                []byte
	wsVerified            bool

	// Vanguard: unconfirmed blocks need to store in cache for waiting final confirmation from orchestrator
	enableVanguardNode bool
	pendingBlockCache  *cache.PendingBlocksCache
	confirmedBlockCh   chan *ethpb.SignedBeaconBlock
	orcRPCClient       orchestrator.Client
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf    int
	ChainStartFetcher powchain.ChainStartFetcher
	BeaconDB          db.HeadAccessDatabase
	DepositCache      *depositcache.DepositCache
	AttPool           attestations.Pool
	ExitPool          voluntaryexits.PoolManager
	SlashingPool      slashings.PoolManager
	P2p               p2p.Broadcaster
	MaxRoutines       int
	StateNotifier     statefeed.Notifier
	BlockNotifier     blockfeed.Notifier
	ForkChoiceStore   f.ForkChoicer
	OpsService        *attestations.Service
	StateGen          *stategen.State
	WspBlockRoot      []byte
	WspEpoch          types.Epoch

	// Vanguard: orchestrator client reference to get confirmation status
	OrcRPCClient       orchestrator.Client
	EnableVanguardNode bool
}

// NewService instantiates a new block service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &Service{
		ctx:                  ctx,
		cancel:               cancel,
		beaconDB:             cfg.BeaconDB,
		depositCache:         cfg.DepositCache,
		chainStartFetcher:    cfg.ChainStartFetcher,
		attPool:              cfg.AttPool,
		exitPool:             cfg.ExitPool,
		slashingPool:         cfg.SlashingPool,
		p2p:                  cfg.P2p,
		maxRoutines:          cfg.MaxRoutines,
		stateNotifier:        cfg.StateNotifier,
		blockNotifier:        cfg.BlockNotifier,
		forkChoiceStore:      cfg.ForkChoiceStore,
		boundaryRoots:        [][32]byte{},
		checkpointStateCache: cache.NewCheckpointStateCache(),
		opsService:           cfg.OpsService,
		stateGen:             cfg.StateGen,
		initSyncBlocks:       make(map[[32]byte]*ethpb.SignedBeaconBlock),
		justifiedBalances:    make([]uint64, 0),
		wsEpoch:              cfg.WspEpoch,
		wsRoot:               cfg.WspBlockRoot,

		pendingBlockCache:  cache.NewPendingBlocksCache(), // Vanguard: Initialize pending block cache
		confirmedBlockCh:   make(chan *ethpb.SignedBeaconBlock),
		orcRPCClient:       cfg.OrcRPCClient,
		enableVanguardNode: cfg.EnableVanguardNode,
	}

	// vanguard: loop for getting confirmation from orchestrator node
	if s.enableVanguardNode {
		go s.processOrcConfirmationLoop(ctx)
	}
	return s, nil
}

// Start a blockchain service's main event loop.
func (s *Service) Start() {
	// For running initial sync with state cache, in an event of restart, we use
	// last finalized check point as start point to sync instead of head
	// state. This is because we no longer save state every slot during sync.
	cp, err := s.beaconDB.FinalizedCheckpoint(s.ctx)
	if err != nil {
		log.Fatalf("Could not fetch finalized cp: %v", err)
	}

	r := bytesutil.ToBytes32(cp.Root)
	// Before the first finalized epoch, in the current epoch,
	// the finalized root is defined as zero hashes instead of genesis root hash.
	// We want to use genesis root to retrieve for state.
	if r == params.BeaconConfig().ZeroHash {
		genesisBlock, err := s.beaconDB.GenesisBlock(s.ctx)
		if err != nil {
			log.Fatalf("Could not fetch finalized cp: %v", err)
		}
		if genesisBlock != nil {
			r, err = genesisBlock.Block.HashTreeRoot()
			if err != nil {
				log.Fatalf("Could not tree hash genesis block: %v", err)
			}
		}
	}
	beaconState, err := s.stateGen.StateByRoot(s.ctx, r)
	if err != nil {
		log.Fatalf("Could not fetch beacon state by root: %v", err)
	}

	// Make sure that attestation processor is subscribed and ready for state initializing event.
	attestationProcessorSubscribed := make(chan struct{}, 1)

	// If the chain has already been initialized, simply start the block processing routine.
	if beaconState != nil {
		log.Info("Blockchain data already exists in DB, initializing...")
		s.genesisTime = time.Unix(int64(beaconState.GenesisTime()), 0)
		s.opsService.SetGenesisTime(beaconState.GenesisTime())
		if err := s.initializeChainInfo(s.ctx); err != nil {
			log.Fatalf("Could not set up chain info: %v", err)
		}

		// We start a counter to genesis, if needed.
		gState, err := s.beaconDB.GenesisState(s.ctx)
		if err != nil {
			log.Fatalf("Could not retrieve genesis state: %v", err)
		}
		gRoot, err := gState.HashTreeRoot(s.ctx)
		if err != nil {
			log.Fatalf("Could not hash tree root genesis state: %v", err)
		}
		go slotutil.CountdownToGenesis(s.ctx, s.genesisTime, uint64(gState.NumValidators()), gRoot)

		justifiedCheckpoint, err := s.beaconDB.JustifiedCheckpoint(s.ctx)
		if err != nil {
			log.Fatalf("Could not get justified checkpoint: %v", err)
		}
		finalizedCheckpoint, err := s.beaconDB.FinalizedCheckpoint(s.ctx)
		if err != nil {
			log.Fatalf("Could not get finalized checkpoint: %v", err)
		}

		// Resume fork choice.
		s.justifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		if err := s.cacheJustifiedStateBalances(s.ctx, s.ensureRootNotZeros(bytesutil.ToBytes32(s.justifiedCheckpt.Root))); err != nil {
			log.Fatalf("Could not cache justified state balances: %v", err)
		}
		s.prevJustifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		s.bestJustifiedCheckpt = stateTrie.CopyCheckpoint(justifiedCheckpoint)
		s.finalizedCheckpt = stateTrie.CopyCheckpoint(finalizedCheckpoint)
		s.prevFinalizedCheckpt = stateTrie.CopyCheckpoint(finalizedCheckpoint)
		s.resumeForkChoice(justifiedCheckpoint, finalizedCheckpoint)

		ss, err := helpers.StartSlot(s.finalizedCheckpt.Epoch)
		if err != nil {
			log.Fatalf("Could not get start slot of finalized epoch: %v", err)
		}
		h := s.headBlock().Block
		if h.Slot > ss {
			log.WithFields(logrus.Fields{
				"startSlot": ss,
				"endSlot":   h.Slot,
			}).Info("Loading blocks to fork choice store, this may take a while.")
			if err := s.fillInForkChoiceMissingBlocks(s.ctx, h, s.finalizedCheckpt, s.justifiedCheckpt); err != nil {
				log.Fatalf("Could not fill in fork choice store missing blocks: %v", err)
			}
		}

		if err := s.VerifyWeakSubjectivityRoot(s.ctx); err != nil {
			// Exit run time if the node failed to verify weak subjectivity checkpoint.
			log.Fatalf("Could not verify weak subjectivity checkpoint: %v", err)
		}

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
						s.processChainStartTime(s.ctx, data.StartTime)
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

	go s.processAttestationsRoutine(attestationProcessorSubscribed)
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
	gRoot, err := initializedState.HashTreeRoot(s.ctx)
	if err != nil {
		log.Fatalf("Could not hash tree root genesis state: %v", err)
	}
	go slotutil.CountdownToGenesis(ctx, genesisTime, uint64(initializedState.NumValidators()), gRoot)

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
	ctx, span := trace.StartSpan(ctx, "beacon-chain.Service.initializeBeaconChain")
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
	if err := helpers.UpdateProposerIndicesInCache(genesisState); err != nil {
		return nil, err
	}

	// TODO: trigger helpers.PastConsensusInfo() here

	s.opsService.SetGenesisTime(genesisState.GenesisTime())

	return genesisState, nil
}

// Stop the blockchain service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()

	if s.stateGen != nil && s.head != nil && s.head.state != nil {
		if err := s.stateGen.ForceCheckpoint(s.ctx, s.head.state.FinalizedCheckpoint().Root); err != nil {
			return err
		}
	}

	// Save initial sync cached blocks to the DB before stop.
	return s.beaconDB.SaveBlocks(s.ctx, s.getInitSyncBlocks())
}

// Status always returns nil unless there is an error condition that causes
// this service to be unhealthy.
func (s *Service) Status() error {
	if s.genesisRoot == params.BeaconConfig().ZeroHash {
		return errors.New("genesis state has not been created")
	}
	if runtime.NumGoroutine() > s.maxRoutines {
		return fmt.Errorf("too many goroutines %d", runtime.NumGoroutine())
	}
	return nil
}

// This gets called when beacon chain is first initialized to save genesis data (state, block, and more) in db.
func (s *Service) saveGenesisData(ctx context.Context, genesisState *stateTrie.BeaconState) error {
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
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

	s.stateGen.SaveFinalizedState(0, genesisBlkRoot, genesisState)

	if err := s.beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis block root")
	}

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
	genesisBlkRoot, err := genesisBlock.Block.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not get signing root of genesis block")
	}
	s.genesisRoot = genesisBlkRoot

	finalized, err := s.beaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized checkpoint from db")
	}
	if finalized == nil {
		// This should never happen. At chain start, the finalized checkpoint
		// would be the genesis state and block.
		return errors.New("no finalized epoch in the database")
	}
	finalizedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(finalized.Root))
	var finalizedState *stateTrie.BeaconState

	finalizedState, err = s.stateGen.Resume(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized state from db")
	}

	if flags.Get().HeadSync {
		headBlock, err := s.beaconDB.HeadBlock(ctx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head block")
		}
		headEpoch := helpers.SlotToEpoch(headBlock.Block.Slot)
		var epochsSinceFinality types.Epoch
		if headEpoch > finalized.Epoch {
			epochsSinceFinality = headEpoch - finalized.Epoch
		}
		// Head sync when node is far enough beyond known finalized epoch,
		// this becomes really useful during long period of non-finality.
		if epochsSinceFinality >= headSyncMinEpochsAfterCheckpoint {
			headRoot, err := headBlock.Block.HashTreeRoot()
			if err != nil {
				return errors.Wrap(err, "could not hash head block")
			}
			finalizedState, err := s.stateGen.Resume(ctx)
			if err != nil {
				return errors.Wrap(err, "could not get finalized state from db")
			}
			log.Infof("Regenerating state from the last checkpoint at slot %d to current head slot of %d."+
				"This process may take a while, please wait.", finalizedState.Slot(), headBlock.Block.Slot)
			headState, err := s.stateGen.StateByRoot(ctx, headRoot)
			if err != nil {
				return errors.Wrap(err, "could not retrieve head state")
			}
			s.setHead(headRoot, headBlock, headState)
			return nil
		} else {
			log.Warnf("Finalized checkpoint at slot %d is too close to the current head slot, "+
				"resetting head from the checkpoint ('--%s' flag is ignored).",
				finalizedState.Slot(), flags.HeadSync.Name)
		}
	}

	finalizedBlock, err := s.beaconDB.Block(ctx, finalizedRoot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block from db")
	}

	if finalizedState == nil || finalizedBlock == nil {
		return errors.New("finalized state and block can't be nil")
	}
	s.setHead(finalizedRoot, finalizedBlock, finalizedState)

	return nil
}

// This is called when a client starts from non-genesis slot. This passes last justified and finalized
// information to fork choice service to initializes fork choice store.
func (s *Service) resumeForkChoice(justifiedCheckpoint, finalizedCheckpoint *ethpb.Checkpoint) {
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
