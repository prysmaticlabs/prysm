// Package blockchain defines the life-cycle and status of the beacon chain
// as well as the Ethereum Serenity beacon chain fork-choice rule based on
// Casper Proof of Stake finality.
package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/forkchoice"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ChainFeeds interface defines the methods of the ChainService which provide
// information feeds.
type ChainFeeds interface {
	StateInitializedFeed() *event.Feed
}

// ChainService represents a service that handles the internal
// logic of managing the full PoS beacon chain.
type ChainService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	beaconDB             db.Database
	depositCache         *depositcache.DepositCache
	web3Service          *powchain.Web3Service
	opsPoolService       operations.OperationFeeds
	forkChoiceStore      forkchoice.ForkChoicer
	chainStartChan       chan time.Time
	genesisTime          time.Time
	stateInitializedFeed *event.Feed
	p2p                  p2p.Broadcaster
	maxRoutines          int64
	headSlot             uint64
	headBlock            *ethpb.BeaconBlock
	headState            *pb.BeaconState
	canonicalRoots       map[uint64][]byte
	canonicalRootsLock   sync.RWMutex
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf int
	Web3Service    *powchain.Web3Service
	BeaconDB       db.Database
	DepositCache   *depositcache.DepositCache
	OpsPoolService operations.OperationFeeds
	P2p            p2p.Broadcaster
	MaxRoutines    int64
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	ctx, cancel := context.WithCancel(ctx)
	store := forkchoice.NewForkChoiceService(ctx, cfg.BeaconDB)
	return &ChainService{
		ctx:                  ctx,
		cancel:               cancel,
		beaconDB:             cfg.BeaconDB,
		depositCache:         cfg.DepositCache,
		web3Service:          cfg.Web3Service,
		opsPoolService:       cfg.OpsPoolService,
		forkChoiceStore:      store,
		chainStartChan:       make(chan time.Time),
		stateInitializedFeed: new(event.Feed),
		p2p:                  cfg.P2p,
		canonicalRoots:       make(map[uint64][]byte),
		maxRoutines:          cfg.MaxRoutines,
	}, nil
}

// Start a blockchain service's main event loop.
func (c *ChainService) Start() {
	beaconState, err := c.beaconDB.HeadState(c.ctx)
	if err != nil {
		log.Fatalf("Could not fetch beacon state: %v", err)
	}
	// If the chain has already been initialized, simply start the block processing routine.
	if beaconState != nil {
		log.Info("Beacon chain data already exists, starting service")
		c.genesisTime = time.Unix(int64(beaconState.GenesisTime), 0)
	} else {
		log.Info("Waiting for ChainStart log from the Validator Deposit Contract to start the beacon chain...")
		if c.web3Service == nil {
			log.Fatal("Not configured web3Service for POW chain")
			return // return need for TestStartUninitializedChainWithoutConfigPOWChain.
		}
		subChainStart := c.web3Service.ChainStartFeed().Subscribe(c.chainStartChan)
		go func() {
			genesisTime := <-c.chainStartChan
			c.processChainStartTime(c.ctx, genesisTime, subChainStart)
			return
		}()
	}
}

// processChainStartTime initializes a series of deposits from the ChainStart deposits in the eth1
// deposit contract, initializes the beacon chain's state, and kicks off the beacon chain.
func (c *ChainService) processChainStartTime(ctx context.Context, genesisTime time.Time, chainStartSub event.Subscription) {
	initialDeposits := c.web3Service.ChainStartDeposits()
	if err := c.initializeBeaconChain(ctx, genesisTime, initialDeposits, c.web3Service.ChainStartETH1Data()); err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	c.stateInitializedFeed.Send(genesisTime)
	chainStartSub.Unsubscribe()
}

// initializes the state and genesis block of the beacon chain to persistent storage
// based on a genesis timestamp value obtained from the ChainStart event emitted
// by the ETH1.0 Deposit Contract and the POWChain service of the node.
func (c *ChainService) initializeBeaconChain(
	ctx context.Context,
	genesisTime time.Time,
	deposits []*ethpb.Deposit,
	eth1data *ethpb.Eth1Data) error {
	_, span := trace.StartSpan(context.Background(), "beacon-chain.ChainService.initializeBeaconChain")
	defer span.End()
	log.Info("ChainStart time reached, starting the beacon chain!")
	c.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())

	genesisState, err := state.GenesisBeaconState(deposits, unixTime, eth1data)
	if err != nil {
		return errors.Wrap(err, "could not initialize genesis state")
	}

	stateRoot, err := ssz.HashTreeRoot(genesisState)
	if err != nil {
		return errors.Wrap(err, "could not tree hash genesis state")
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])

	if err := c.saveGenesisValidators(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not save genesis validators")
	}

	if err := c.forkChoiceStore.GenesisStore(ctx, genesisState); err != nil {
		return errors.Wrap(err, "could not start genesis store for fork choice")
	}

	c.headBlock = genesisBlk
	c.headState = genesisState
	c.canonicalRoots[genesisState.Slot] = c.FinalizedCheckpt().Root
	c.canonicalRoots[genesisState.Slot] = c.FinalizedCheckpt().Root

	return nil
}

// Stop the blockchain service's main event loop and associated goroutines.
func (c *ChainService) Stop() error {
	defer c.cancel()

	log.Info("Stopping service")
	return nil
}

// Status always returns nil.
// TODO(1202): Add service health checks.
func (c *ChainService) Status() error {
	if runtime.NumGoroutine() > int(c.maxRoutines) {
		return fmt.Errorf("too many goroutines %d", runtime.NumGoroutine())
	}
	return nil
}

// StateInitializedFeed returns a feed that is written to
// when the beacon state is first initialized.
func (c *ChainService) StateInitializedFeed() *event.Feed {
	return c.stateInitializedFeed
}

// This gets called to update canonical root mapping.
func (c *ChainService) saveHead(ctx context.Context, b *ethpb.BeaconBlock, r [32]byte) error {

	c.headSlot = b.Slot

	c.canonicalRootsLock.Lock()
	c.canonicalRoots[b.Slot] = r[:]
	defer c.canonicalRootsLock.Unlock()

	if err := c.beaconDB.SaveHeadBlockRoot(ctx, r); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}
	c.headBlock = b

	s, err := c.beaconDB.State(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	c.headState = s

	log.WithFields(logrus.Fields{
		"slots": b.Slot,
		"root":  hex.EncodeToString(r[:]),
	}).Debug("Saved head info")
	return nil
}

// This gets called when beacon chain is first initialized to save validator indices and pubkeys in db
func (c *ChainService) saveGenesisValidators(ctx context.Context, s *pb.BeaconState) error {
	for i, v := range s.Validators {
		if err := c.beaconDB.SaveValidatorIndex(ctx, bytesutil.ToBytes48(v.PublicKey), uint64(i)); err != nil {
			return errors.Wrapf(err, "could not save validator index: %d", i)
		}
	}
	return nil
}
