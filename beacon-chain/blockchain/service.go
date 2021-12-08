// Package blockchain defines the life-cycle of the blockchain at the core of
// Ethereum, including processing of new blocks and attestations using proof of stake.
package blockchain

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	f "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// headSyncMinEpochsAfterCheckpoint defines how many epochs should elapse after known finalization
// checkpoint for head sync to be triggered.
const headSyncMinEpochsAfterCheckpoint = 128

// Service represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type Service struct {
	cfg         *config
	ctx         context.Context
	cancel      context.CancelFunc
	genesisTime time.Time
	head        *head
	headLock    sync.RWMutex
	// originBlockRoot is the genesis root, or weak subjectivity checkpoint root, depending on how the node is initialized
	originBlockRoot       [32]byte
	justifiedCheckpt      *ethpb.Checkpoint
	prevJustifiedCheckpt  *ethpb.Checkpoint
	bestJustifiedCheckpt  *ethpb.Checkpoint
	finalizedCheckpt      *ethpb.Checkpoint
	prevFinalizedCheckpt  *ethpb.Checkpoint
	nextEpochBoundarySlot types.Slot
	boundaryRoots         [][32]byte
	checkpointStateCache  *cache.CheckpointStateCache
	initSyncBlocks        map[[32]byte]block.SignedBeaconBlock
	initSyncBlocksLock    sync.RWMutex
	//justifiedBalances     []uint64
	justifiedBalances *stateBalanceCache
	wsVerifier        *WeakSubjectivityVerifier
}

// config options for the service.
type config struct {
	BeaconBlockBuf          int
	ChainStartFetcher       powchain.ChainStartFetcher
	BeaconDB                db.HeadAccessDatabase
	DepositCache            *depositcache.DepositCache
	AttPool                 attestations.Pool
	ExitPool                voluntaryexits.PoolManager
	SlashingPool            slashings.PoolManager
	P2p                     p2p.Broadcaster
	MaxRoutines             int
	StateNotifier           statefeed.Notifier
	ForkChoiceStore         f.ForkChoicer
	AttService              *attestations.Service
	StateGen                *stategen.State
	SlasherAttestationsFeed *event.Feed
	WeakSubjectivityCheckpt *ethpb.Checkpoint
	FinalizedStateAtStartUp state.BeaconState
}

// NewService instantiates a new block service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	srv := &Service{
		ctx:                  ctx,
		cancel:               cancel,
		boundaryRoots:        [][32]byte{},
		checkpointStateCache: cache.NewCheckpointStateCache(),
		initSyncBlocks:       make(map[[32]byte]block.SignedBeaconBlock),
		cfg:                  &config{},
	}
	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, err
		}
	}
	var err error
	if srv.justifiedBalances == nil {
		srv.justifiedBalances, err = newStateBalanceCache(srv.cfg.StateGen)
		if err != nil {
			return nil, err
		}
	}
	srv.wsVerifier, err = NewWeakSubjectivityVerifier(srv.cfg.WeakSubjectivityCheckpt, srv.cfg.BeaconDB)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

// Start a blockchain service's main event loop.
func (s *Service) Start() {
	saved := s.cfg.FinalizedStateAtStartUp

	if saved != nil && !saved.IsNil() {
		if err := s.startFromSavedState(saved); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := s.startFromPOWChain(); err != nil {
			log.Fatal(err)
		}
	}
}

// Stop the blockchain service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()

	if s.cfg.StateGen != nil && s.head != nil && s.head.state != nil {
		if err := s.cfg.StateGen.ForceCheckpoint(s.ctx, s.head.state.FinalizedCheckpoint().Root); err != nil {
			return err
		}
	}

	// Save initial sync cached blocks to the DB before stop.
	return s.cfg.BeaconDB.SaveBlocks(s.ctx, s.getInitSyncBlocks())
}

// Status always returns nil unless there is an error condition that causes
// this service to be unhealthy.
func (s *Service) Status() error {
	if s.originBlockRoot == params.BeaconConfig().ZeroHash {
		return errors.New("genesis state has not been created")
	}
	if runtime.NumGoroutine() > s.cfg.MaxRoutines {
		return fmt.Errorf("too many goroutines %d", runtime.NumGoroutine())
	}
	return nil
}

func (s *Service) startFromSavedState(saved state.BeaconState) error {
	log.Info("Blockchain data already exists in DB, initializing...")
	s.genesisTime = time.Unix(int64(saved.GenesisTime()), 0)
	s.cfg.AttService.SetGenesisTime(saved.GenesisTime())

	originRoot, err := s.originRootFromSavedState(s.ctx)
	if err != nil {
		return err
	}
	s.originBlockRoot = originRoot

	if err := s.initializeHeadFromDB(s.ctx); err != nil {
		return errors.Wrap(err, "could not set up chain info")
	}
	spawnCountdownIfPreGenesis(s.ctx, s.genesisTime, s.cfg.BeaconDB)

	justified, err := s.cfg.BeaconDB.JustifiedCheckpoint(s.ctx)
	if err != nil {
		return errors.Wrap(err, "could not get justified checkpoint")
	}
	s.prevJustifiedCheckpt = ethpb.CopyCheckpoint(justified)
	s.bestJustifiedCheckpt = ethpb.CopyCheckpoint(justified)
	s.justifiedCheckpt = ethpb.CopyCheckpoint(justified)

	finalized, err := s.cfg.BeaconDB.FinalizedCheckpoint(s.ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized checkpoint")
	}
	s.prevFinalizedCheckpt = ethpb.CopyCheckpoint(finalized)
	s.finalizedCheckpt = ethpb.CopyCheckpoint(finalized)

	store := protoarray.New(justified.Epoch, finalized.Epoch, bytesutil.ToBytes32(finalized.Root))
	s.cfg.ForkChoiceStore = store

	ss, err := slots.EpochStart(s.finalizedCheckpt.Epoch)
	if err != nil {
		return errors.Wrap(err, "could not get start slot of finalized epoch")
	}
	h := s.headBlock().Block()
	if h.Slot() > ss {
		log.WithFields(logrus.Fields{
			"startSlot": ss,
			"endSlot":   h.Slot(),
		}).Info("Loading blocks to fork choice store, this may take a while.")
		if err := s.fillInForkChoiceMissingBlocks(s.ctx, h, s.finalizedCheckpt, s.justifiedCheckpt); err != nil {
			return errors.Wrap(err, "could not fill in fork choice store missing blocks")
		}
	}

	// not attempting to save initial sync blocks here, because there shouldn't be any until
	// after the statefeed.Initialized event is fired (below)
	if err := s.wsVerifier.VerifyWeakSubjectivity(s.ctx, s.finalizedCheckpt.Epoch); err != nil {
		// Exit run time if the node failed to verify weak subjectivity checkpoint.
		return errors.Wrap(err, "could not verify initial checkpoint provided for chain sync")
	}

	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime:             s.genesisTime,
			GenesisValidatorsRoot: saved.GenesisValidatorRoot(),
		},
	})

	s.spawnProcessAttestationsRoutine(s.cfg.StateNotifier.StateFeed())

	return nil
}

func (s *Service) originRootFromSavedState(ctx context.Context) ([32]byte, error) {
	// first check if we have started from checkpoint sync and have a root
	originRoot, err := s.cfg.BeaconDB.OriginBlockRoot(ctx)
	if err == nil {
		return originRoot, nil
	}
	if !errors.Is(err, db.ErrNotFound) {
		return originRoot, errors.Wrap(err, "could not retrieve checkpoint sync chain origin data from db")
	}

	// we got here because OriginBlockRoot gave us an ErrNotFound. this means the node was started from a genesis state,
	// so we should have a value for GenesisBlock
	genesisBlock, err := s.cfg.BeaconDB.GenesisBlock(ctx)
	if err != nil {
		return originRoot, errors.Wrap(err, "could not get genesis block from db")
	}
	if err := helpers.BeaconBlockIsNil(genesisBlock); err != nil {
		return originRoot, err
	}
	genesisBlkRoot, err := genesisBlock.Block().HashTreeRoot()
	if err != nil {
		return genesisBlkRoot, errors.Wrap(err, "could not get signing root of genesis block")
	}
	return genesisBlkRoot, nil
}

// initializeHeadFromDB uses the finalized checkpoint and head block found in the database to set the current head
// note that this may block until stategen replays blocks between the finalized and head blocks
// if the head sync flag was specified and the gap between the finalized and head blocks is at least 128 epochs long
func (s *Service) initializeHeadFromDB(ctx context.Context) error {
	finalized, err := s.cfg.BeaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get finalized checkpoint from db")
	}
	if finalized == nil {
		// This should never happen. At chain start, the finalized checkpoint
		// would be the genesis state and block.
		return errors.New("no finalized epoch in the database")
	}
	finalizedRoot := s.ensureRootNotZeros(bytesutil.ToBytes32(finalized.Root))
	var finalizedState state.BeaconState

	finalizedState, err = s.cfg.StateGen.Resume(ctx, s.cfg.FinalizedStateAtStartUp)
	if err != nil {
		return errors.Wrap(err, "could not get finalized state from db")
	}

	if flags.Get().HeadSync {
		headBlock, err := s.cfg.BeaconDB.HeadBlock(ctx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head block")
		}
		headEpoch := slots.ToEpoch(headBlock.Block().Slot())
		var epochsSinceFinality types.Epoch
		if headEpoch > finalized.Epoch {
			epochsSinceFinality = headEpoch - finalized.Epoch
		}
		// Head sync when node is far enough beyond known finalized epoch,
		// this becomes really useful during long period of non-finality.
		if epochsSinceFinality >= headSyncMinEpochsAfterCheckpoint {
			headRoot, err := headBlock.Block().HashTreeRoot()
			if err != nil {
				return errors.Wrap(err, "could not hash head block")
			}
			finalizedState, err := s.cfg.StateGen.Resume(ctx, s.cfg.FinalizedStateAtStartUp)
			if err != nil {
				return errors.Wrap(err, "could not get finalized state from db")
			}
			log.Infof("Regenerating state from the last checkpoint at slot %d to current head slot of %d."+
				"This process may take a while, please wait.", finalizedState.Slot(), headBlock.Block().Slot())
			headState, err := s.cfg.StateGen.StateByRoot(ctx, headRoot)
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

	finalizedBlock, err := s.cfg.BeaconDB.Block(ctx, finalizedRoot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block from db")
	}

	if finalizedState == nil || finalizedState.IsNil() || finalizedBlock == nil || finalizedBlock.IsNil() {
		return errors.New("finalized state and block can't be nil")
	}
	s.setHead(finalizedRoot, finalizedBlock, finalizedState)

	return nil
}

func (s *Service) startFromPOWChain() error {
	log.Info("Waiting to reach the validator deposit threshold to start the beacon chain...")
	if s.cfg.ChainStartFetcher == nil {
		return errors.New("not configured web3Service for POW chain")
	}
	go func() {
		stateChannel := make(chan *feed.Event, 1)
		stateSub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChannel)
		defer stateSub.Unsubscribe()
		s.spawnProcessAttestationsRoutine(s.cfg.StateNotifier.StateFeed())
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
					s.onPowchainStart(s.ctx, data.StartTime)
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

	return nil
}

// onPowchainStart initializes a series of deposits from the ChainStart deposits in the eth1
// deposit contract, initializes the beacon chain's state, and kicks off the beacon chain.
func (s *Service) onPowchainStart(ctx context.Context, genesisTime time.Time) {
	preGenesisState := s.cfg.ChainStartFetcher.PreGenesisState()
	initializedState, err := s.initializeBeaconChain(ctx, genesisTime, preGenesisState, s.cfg.ChainStartFetcher.ChainStartEth1Data())
	if err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	// We start a counter to genesis, if needed.
	gRoot, err := initializedState.HashTreeRoot(s.ctx)
	if err != nil {
		log.Fatalf("Could not hash tree root genesis state: %v", err)
	}
	go slots.CountdownToGenesis(ctx, genesisTime, uint64(initializedState.NumValidators()), gRoot)

	// We send out a state initialized event to the rest of the services
	// running in the beacon node.
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
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
	preGenesisState state.BeaconState,
	eth1data *ethpb.Eth1Data) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.Service.initializeBeaconChain")
	defer span.End()
	s.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())

	genesisState, err := transition.OptimizedGenesisBeaconState(unixTime, preGenesisState, eth1data)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize genesis state")
	}

	if err := s.saveGenesisData(ctx, genesisState); err != nil {
		return nil, errors.Wrap(err, "could not save genesis data")
	}

	log.Info("Initialized beacon chain genesis state")

	// Clear out all pre-genesis data now that the state is initialized.
	s.cfg.ChainStartFetcher.ClearPreGenesisData()

	// Update committee shuffled indices for genesis epoch.
	if err := helpers.UpdateCommitteeCache(genesisState, 0 /* genesis epoch */); err != nil {
		return nil, err
	}
	if err := helpers.UpdateProposerIndicesInCache(ctx, genesisState); err != nil {
		return nil, err
	}

	s.cfg.AttService.SetGenesisTime(genesisState.GenesisTime())

	return genesisState, nil
}

// This gets called when beacon chain is first initialized to save genesis data (state, block, and more) in db.
func (s *Service) saveGenesisData(ctx context.Context, genesisState state.BeaconState) error {
	if err := s.cfg.BeaconDB.SaveGenesisData(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis data")
	}
	genesisBlk, err := s.cfg.BeaconDB.GenesisBlock(ctx)
	if err != nil || genesisBlk == nil || genesisBlk.IsNil() {
		return fmt.Errorf("could not load genesis block: %v", err)
	}
	genesisBlkRoot, err := genesisBlk.Block().HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}

	s.originBlockRoot = genesisBlkRoot
	s.cfg.StateGen.SaveFinalizedState(0 /*slot*/, genesisBlkRoot, genesisState)

	// Finalized checkpoint at genesis is a zero hash.
	genesisCheckpoint := genesisState.FinalizedCheckpoint()

	s.justifiedCheckpt = ethpb.CopyCheckpoint(genesisCheckpoint)
	s.prevJustifiedCheckpt = ethpb.CopyCheckpoint(genesisCheckpoint)
	s.bestJustifiedCheckpt = ethpb.CopyCheckpoint(genesisCheckpoint)
	s.finalizedCheckpt = ethpb.CopyCheckpoint(genesisCheckpoint)
	s.prevFinalizedCheckpt = ethpb.CopyCheckpoint(genesisCheckpoint)

	if err := s.cfg.ForkChoiceStore.ProcessBlock(ctx,
		genesisBlk.Block().Slot(),
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

// This returns true if block has been processed before. Two ways to verify the block has been processed:
// 1.) Check fork choice store.
// 2.) Check DB.
// Checking 1.) is ten times faster than checking 2.)
func (s *Service) hasBlock(ctx context.Context, root [32]byte) bool {
	if s.cfg.ForkChoiceStore.HasNode(root) {
		return true
	}

	return s.cfg.BeaconDB.HasBlock(ctx, root)
}

func spawnCountdownIfPreGenesis(ctx context.Context, genesisTime time.Time, db db.HeadAccessDatabase) {
	currentTime := prysmTime.Now()
	if currentTime.After(genesisTime) {
		return
	}

	gState, err := db.GenesisState(ctx)
	if err != nil {
		log.Fatalf("Could not retrieve genesis state: %v", err)
	}
	gRoot, err := gState.HashTreeRoot(ctx)
	if err != nil {
		log.Fatalf("Could not hash tree root genesis state: %v", err)
	}
	go slots.CountdownToGenesis(ctx, genesisTime, uint64(gState.NumValidators()), gRoot)
}
