// Package blockchain defines the life-cycle and status of the beacon chain
// as well as the Ethereum Serenity beacon chain fork-choice rule based on
// Casper Proof of Stake finality.
package blockchain

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	f "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// Service represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type Service struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	beaconDB               db.HeadAccessDatabase
	depositCache           *depositcache.DepositCache
	chainStartFetcher      powchain.ChainStartFetcher
	attPool                attestations.Pool
	exitPool               *voluntaryexits.Pool
	genesisTime            time.Time
	p2p                    p2p.Broadcaster
	maxRoutines            int64
	headSlot               uint64
	headBlock              *ethpb.SignedBeaconBlock
	headState              *stateTrie.BeaconState
	canonicalRoots         map[uint64][]byte
	headLock               sync.RWMutex
	stateNotifier          statefeed.Notifier
	genesisRoot            [32]byte
	epochParticipation     map[uint64]*precompute.Balance
	epochParticipationLock sync.RWMutex
	forkChoiceStore        f.ForkChoicer
	justifiedCheckpt       *ethpb.Checkpoint
	bestJustifiedCheckpt   *ethpb.Checkpoint
	finalizedCheckpt       *ethpb.Checkpoint
	prevFinalizedCheckpt   *ethpb.Checkpoint
	nextEpochBoundarySlot  uint64
	voteLock               sync.RWMutex
	initSyncState          map[[32]byte]*stateTrie.BeaconState
	initSyncStateLock      sync.RWMutex
	checkpointState        *cache.CheckpointStateCache
	checkpointStateLock    sync.Mutex
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf    int
	ChainStartFetcher powchain.ChainStartFetcher
	BeaconDB          db.HeadAccessDatabase
	DepositCache      *depositcache.DepositCache
	AttPool           attestations.Pool
	ExitPool          *voluntaryexits.Pool
	P2p               p2p.Broadcaster
	MaxRoutines       int64
	StateNotifier     statefeed.Notifier
	ForkChoiceStore   f.ForkChoicer
}

// NewService instantiates a new block service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                ctx,
		cancel:             cancel,
		beaconDB:           cfg.BeaconDB,
		depositCache:       cfg.DepositCache,
		chainStartFetcher:  cfg.ChainStartFetcher,
		attPool:            cfg.AttPool,
		exitPool:           cfg.ExitPool,
		p2p:                cfg.P2p,
		canonicalRoots:     make(map[uint64][]byte),
		maxRoutines:        cfg.MaxRoutines,
		stateNotifier:      cfg.StateNotifier,
		epochParticipation: make(map[uint64]*precompute.Balance),
		forkChoiceStore:    cfg.ForkChoiceStore,
		initSyncState:      make(map[[32]byte]*stateTrie.BeaconState),
		checkpointState:    cache.NewCheckpointStateCache(),
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
	if featureconfig.Get().InitSyncCacheState {
		cp, err := s.beaconDB.FinalizedCheckpoint(ctx)
		if err != nil {
			log.Fatalf("Could not fetch finalized cp: %v", err)
		}
		if beaconState == nil {
			beaconState, err = s.beaconDB.State(ctx, bytesutil.ToBytes32(cp.Root))
			if err != nil {
				log.Fatalf("Could not fetch beacon state: %v", err)
			}
		}
	}

	// If the chain has already been initialized, simply start the block processing routine.
	if beaconState != nil {
		log.Info("Blockchain data already exists in DB, initializing...")
		s.genesisTime = time.Unix(int64(beaconState.GenesisTime()), 0)
		if err := s.initializeChainInfo(ctx); err != nil {
			log.Fatalf("Could not set up chain info: %v", err)
		}
		justifiedCheckpoint, err := s.beaconDB.JustifiedCheckpoint(ctx)
		if err != nil {
			log.Fatalf("Could not get justified checkpoint: %v", err)
		}
		finalizedCheckpoint, err := s.beaconDB.FinalizedCheckpoint(ctx)
		if err != nil {
			log.Fatalf("Could not get finalized checkpoint: %v", err)
		}

		// Resume fork choice.
		s.justifiedCheckpt = proto.Clone(justifiedCheckpoint).(*ethpb.Checkpoint)
		s.bestJustifiedCheckpt = proto.Clone(justifiedCheckpoint).(*ethpb.Checkpoint)
		s.finalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)
		s.prevFinalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)
		s.resumeForkChoice(justifiedCheckpoint, finalizedCheckpoint)

		if finalizedCheckpoint.Epoch > 1 {
			if err := s.pruneGarbageState(ctx, helpers.StartSlot(finalizedCheckpoint.Epoch)-params.BeaconConfig().SlotsPerEpoch); err != nil {
				log.WithError(err).Warn("Could not prune old states")
			}
		}

		s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Initialized,
			Data: &statefeed.InitializedData{
				StartTime: s.genesisTime,
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
			for {
				select {
				case event := <-stateChannel:
					if event.Type == statefeed.ChainStarted {
						data := event.Data.(*statefeed.ChainStartedData)
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

	go s.processAttestation()
}

// processChainStartTime initializes a series of deposits from the ChainStart deposits in the eth1
// deposit contract, initializes the beacon chain's state, and kicks off the beacon chain.
func (s *Service) processChainStartTime(ctx context.Context, genesisTime time.Time) {
	preGenesisState := s.chainStartFetcher.PreGenesisState()
	if err := s.initializeBeaconChain(ctx, genesisTime, preGenesisState, s.chainStartFetcher.ChainStartEth1Data()); err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: genesisTime,
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
	eth1data *ethpb.Eth1Data) error {
	_, span := trace.StartSpan(context.Background(), "beacon-chain.Service.initializeBeaconChain")
	defer span.End()
	s.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())

	genesisState, err := state.OptimizedGenesisBeaconState(unixTime, preGenesisState, eth1data)
	if err != nil {
		return errors.Wrap(err, "could not initialize genesis state")
	}

	if err := s.saveGenesisData(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis data")
	}

	log.Info("Initialized beacon chain genesis state")

	// Clear out all pre-genesis data now that the state is initialized.
	s.chainStartFetcher.ClearPreGenesisData()

	// Update committee shuffled indices for genesis epoch.
	if err := helpers.UpdateCommitteeCache(genesisState, 0 /* genesis epoch */); err != nil {
		return err
	}
	if err := helpers.UpdateProposerIndicesInCache(genesisState, 0 /* genesis epoch */); err != nil {
		return err
	}

	return nil
}

// Stop the blockchain service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	defer s.cancel()
	return nil
}

// Status always returns nil unless there is an error condition that causes
// this service to be unhealthy.
func (s *Service) Status() error {
	if runtime.NumGoroutine() > int(s.maxRoutines) {
		return fmt.Errorf("too many goroutines %d", runtime.NumGoroutine())
	}
	return nil
}

// This gets called to update canonical root mapping.
func (s *Service) saveHead(ctx context.Context, signed *ethpb.SignedBeaconBlock, r [32]byte) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	if signed == nil || signed.Block == nil {
		return errors.New("cannot save nil head block")
	}

	s.headSlot = signed.Block.Slot

	s.canonicalRoots[signed.Block.Slot] = r[:]

	if err := s.beaconDB.SaveHeadBlockRoot(ctx, r); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}
	s.headBlock = proto.Clone(signed).(*ethpb.SignedBeaconBlock)

	headState, err := s.beaconDB.State(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	s.headState = headState

	return nil
}

// This gets called to update canonical root mapping. It does not save head block
// root in DB. With the inception of inital-sync-cache-state flag, it uses finalized
// check point as anchors to resume sync therefore head is no longer needed to be saved on per slot basis.
func (s *Service) saveHeadNoDB(ctx context.Context, b *ethpb.SignedBeaconBlock, r [32]byte) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	if b == nil || b.Block == nil {
		return errors.New("cannot save nil head block")
	}

	s.headSlot = b.Block.Slot

	s.canonicalRoots[b.Block.Slot] = r[:]

	s.headBlock = proto.Clone(b).(*ethpb.SignedBeaconBlock)

	headState, err := s.beaconDB.State(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	s.headState = headState

	return nil
}

// This gets called when beacon chain is first initialized to save validator indices and public keys in db.
func (s *Service) saveGenesisValidators(ctx context.Context, state *stateTrie.BeaconState) error {
	pubkeys := make([][48]byte, state.NumValidators())
	indices := make([]uint64, state.NumValidators())

	for i := 0; i < state.NumValidators(); i++ {
		pubkeys[i] = state.PubkeyAtIndex(uint64(i))
		indices[i] = uint64(i)
	}
	return s.beaconDB.SaveValidatorIndices(ctx, pubkeys, indices)
}

// This gets called when beacon chain is first initialized to save genesis data (state, block, and more) in db.
func (s *Service) saveGenesisData(ctx context.Context, genesisState *stateTrie.BeaconState) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	stateRoot, err := genesisState.HashTreeRoot()
	if err != nil {
		return err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := ssz.HashTreeRoot(genesisBlk.Block)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}

	if err := s.beaconDB.SaveBlock(ctx, genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.beaconDB.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could save genesis block root")
	}
	if err := s.saveGenesisValidators(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis validators")
	}

	genesisCheckpoint := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}

	// Add the genesis block to the fork choice store.
	s.justifiedCheckpt = proto.Clone(genesisCheckpoint).(*ethpb.Checkpoint)
	s.bestJustifiedCheckpt = proto.Clone(genesisCheckpoint).(*ethpb.Checkpoint)
	s.finalizedCheckpt = proto.Clone(genesisCheckpoint).(*ethpb.Checkpoint)
	s.prevFinalizedCheckpt = proto.Clone(genesisCheckpoint).(*ethpb.Checkpoint)

	if err := s.forkChoiceStore.ProcessBlock(ctx,
		genesisBlk.Block.Slot,
		genesisBlkRoot,
		params.BeaconConfig().ZeroHash,
		genesisCheckpoint.Epoch,
		genesisCheckpoint.Epoch); err != nil {
		log.Fatalf("Could not process genesis block for fork choice: %v", err)
	}

	s.genesisRoot = genesisBlkRoot
	s.headBlock = genesisBlk
	s.headState = genesisState
	s.canonicalRoots[genesisState.Slot()] = genesisBlkRoot[:]

	return nil
}

// This gets called to initialize chain info variables using the finalized checkpoint stored in DB
func (s *Service) initializeChainInfo(ctx context.Context) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	genesisBlock, err := s.beaconDB.GenesisBlock(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block from db")
	}
	if genesisBlock == nil {
		return errors.New("no genesis block in db")
	}
	genesisBlkRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
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
	s.headState, err = s.beaconDB.State(ctx, bytesutil.ToBytes32(finalized.Root))
	if err != nil {
		return errors.Wrap(err, "could not get finalized state from db")
	}
	s.headBlock, err = s.beaconDB.Block(ctx, bytesutil.ToBytes32(finalized.Root))
	if err != nil {
		return errors.Wrap(err, "could not get finalized block from db")
	}

	if s.headBlock != nil && s.headBlock.Block != nil {
		s.headSlot = s.headBlock.Block.Slot
	}
	s.canonicalRoots[s.headSlot] = finalized.Root

	return nil
}

// This is called when a client starts from a non-genesis slot. It deletes the states in DB
// from slot 1 (avoid genesis state) to `slot`.
func (s *Service) pruneGarbageState(ctx context.Context, slot uint64) error {
	filter := filters.NewFilter().SetStartSlot(1).SetEndSlot(slot)
	roots, err := s.beaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}
	if err := s.beaconDB.DeleteStates(ctx, roots); err != nil {
		return err
	}

	return nil
}

// This is called when a client starts from non-genesis slot. This passes last justified and finalized
// information to fork choice service to initializes fork choice store.
func (s *Service) resumeForkChoice(justifiedCheckpoint *ethpb.Checkpoint, finalizedCheckpoint *ethpb.Checkpoint) {
	store := protoarray.New(justifiedCheckpoint.Epoch, finalizedCheckpoint.Epoch, bytesutil.ToBytes32(finalizedCheckpoint.Root))
	s.forkChoiceStore = store
}
