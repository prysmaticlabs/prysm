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

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Service represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type Service struct {
	ctx               context.Context
	cancel            context.CancelFunc
	beaconDB          db.Database
	depositCache      *depositcache.DepositCache
	chainStartFetcher powchain.ChainStartFetcher
	opsPoolService    operations.OperationFeeds
	forkChoiceStore   forkchoice.ForkChoicer
	genesisTime       time.Time
	p2p               p2p.Broadcaster
	maxRoutines       int64
	headSlot          uint64
	headBlock         *ethpb.BeaconBlock
	headState         *pb.BeaconState
	canonicalRoots    map[uint64][]byte
	headLock          sync.RWMutex
	stateNotifier     statefeed.Notifier
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf    int
	ChainStartFetcher powchain.ChainStartFetcher
	BeaconDB          db.Database
	DepositCache      *depositcache.DepositCache
	OpsPoolService    operations.OperationFeeds
	P2p               p2p.Broadcaster
	MaxRoutines       int64
	StateNotifier     statefeed.Notifier
}

// NewService instantiates a new block service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	store := forkchoice.NewForkChoiceService(ctx, cfg.BeaconDB)
	return &Service{
		ctx:               ctx,
		cancel:            cancel,
		beaconDB:          cfg.BeaconDB,
		depositCache:      cfg.DepositCache,
		chainStartFetcher: cfg.ChainStartFetcher,
		opsPoolService:    cfg.OpsPoolService,
		forkChoiceStore:   store,
		p2p:               cfg.P2p,
		canonicalRoots:    make(map[uint64][]byte),
		maxRoutines:       cfg.MaxRoutines,
		stateNotifier:     cfg.StateNotifier,
	}, nil
}

// Start a blockchain service's main event loop.
func (s *Service) Start() {
	ctx := context.TODO()
	beaconState, err := s.beaconDB.HeadState(ctx)
	if err != nil {
		log.Fatalf("Could not fetch beacon state: %v", err)
	}

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
		s.genesisTime = time.Unix(int64(beaconState.GenesisTime), 0)
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
		if err := s.forkChoiceStore.GenesisStore(ctx, justifiedCheckpoint, finalizedCheckpoint); err != nil {
			log.Fatalf("Could not start fork choice service: %v", err)
		}
		s.stateNotifier.StateFeed().Send(&statefeed.Event{
			Type: statefeed.StateInitialized,
			Data: &statefeed.StateInitializedData{
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
			stateChannel := make(chan *statefeed.Event, 1)
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
	initialDeposits := s.chainStartFetcher.ChainStartDeposits()
	if err := s.initializeBeaconChain(ctx, genesisTime, initialDeposits, s.chainStartFetcher.ChainStartEth1Data()); err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	s.stateNotifier.StateFeed().Send(&statefeed.Event{
		Type: statefeed.StateInitialized,
		Data: &statefeed.StateInitializedData{
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
	deposits []*ethpb.Deposit,
	eth1data *ethpb.Eth1Data) error {
	_, span := trace.StartSpan(context.Background(), "beacon-chain.Service.initializeBeaconChain")
	defer span.End()
	log.Info("Genesis time reached, starting the beacon chain")
	s.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())

	genesisState, err := state.GenesisBeaconState(deposits, unixTime, eth1data)
	if err != nil {
		return errors.Wrap(err, "could not initialize genesis state")
	}

	if err := s.saveGenesisData(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis data")
	}

	// Update committee shuffled indices for genesis epoch.
	if featureconfig.Get().EnableNewCache {
		if err := helpers.UpdateCommitteeCache(genesisState); err != nil {
			return err
		}
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
func (s *Service) saveHead(ctx context.Context, b *ethpb.BeaconBlock, r [32]byte) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	s.headSlot = b.Slot

	s.canonicalRoots[b.Slot] = r[:]

	if err := s.beaconDB.SaveHeadBlockRoot(ctx, r); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}
	s.headBlock = b

	headState, err := s.beaconDB.State(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	s.headState = headState

	log.WithFields(logrus.Fields{
		"slot":     b.Slot,
		"headRoot": fmt.Sprintf("%#x", r),
	}).Debug("Saved new head info")
	return nil
}

// This gets called when beacon chain is first initialized to save validator indices and pubkeys in db
func (s *Service) saveGenesisValidators(ctx context.Context, state *pb.BeaconState) error {
	for i, v := range state.Validators {
		if err := s.beaconDB.SaveValidatorIndex(ctx, bytesutil.ToBytes48(v.PublicKey), uint64(i)); err != nil {
			return errors.Wrapf(err, "could not save validator index: %d", i)
		}
	}
	return nil
}

// This gets called when beacon chain is first initialized to save genesis data (state, block, and more) in db
func (s *Service) saveGenesisData(ctx context.Context, genesisState *pb.BeaconState) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	stateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		return errors.Wrap(err, "could not tree hash genesis state")
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}

	if err := s.beaconDB.SaveBlock(ctx, genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could save genesis block root")
	}
	if err := s.beaconDB.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.saveGenesisValidators(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis validators")
	}

	genesisCheckpoint := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}
	if err := s.forkChoiceStore.GenesisStore(ctx, genesisCheckpoint, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "Could not start fork choice service: %v")
	}

	s.headBlock = genesisBlk
	s.headState = genesisState
	s.canonicalRoots[genesisState.Slot] = genesisBlkRoot[:]

	return nil
}

// This gets called to initialize chain info variables using the finalized checkpoint stored in DB
func (s *Service) initializeChainInfo(ctx context.Context) error {
	s.headLock.Lock()
	defer s.headLock.Unlock()

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

	s.headSlot = s.headBlock.Slot
	s.canonicalRoots[s.headSlot] = finalized.Root

	return nil
}
