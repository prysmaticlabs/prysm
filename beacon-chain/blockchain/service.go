// Package blockchain defines the life-cycle and status of the beacon chain
// as well as the Ethereum Serenity beacon chain fork-choice rule based on
// Casper Proof of Stake finality.
package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/fork_choice"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "blockchain")

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
	beaconDB             *db.BeaconDB
	web3Service          *powchain.Web3Service
	opsPoolService       operations.OperationFeeds
	forkChoiceStore      *forkchoice.Store
	chainStartChan       chan time.Time
	canonicalBlockFeed   *event.Feed
	genesisTime          time.Time
	stateInitializedFeed *event.Feed
	p2p                  p2p.Broadcaster
	maxRoutines          int64
	headSlot         uint64
	canonicalRoots       map[uint64][]byte
}

// Config options for the service.
type Config struct {
	BeaconBlockBuf int
	Web3Service    *powchain.Web3Service
	BeaconDB       *db.BeaconDB
	OpsPoolService operations.OperationFeeds
	P2p            p2p.Broadcaster
	MaxRoutines    int64
}

// NewChainService instantiates a new service instance that will
// be registered into a running beacon node.
func NewChainService(ctx context.Context, cfg *Config) (*ChainService, error) {
	store := forkchoice.NewForkChoiceService(ctx, cfg.BeaconDB)
	ctx, cancel := context.WithCancel(ctx)
	return &ChainService{
		ctx:                  ctx,
		cancel:               cancel,
		beaconDB:             cfg.BeaconDB,
		web3Service:          cfg.Web3Service,
		opsPoolService:       cfg.OpsPoolService,
		forkChoiceStore:      store,
		canonicalBlockFeed:   new(event.Feed),
		chainStartChan:       make(chan time.Time),
		stateInitializedFeed: new(event.Feed),
		p2p:                  cfg.P2p,
		maxRoutines:          cfg.MaxRoutines,
		canonicalRoots:       make(map[uint64][]byte),
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
			c.processChainStartTime(genesisTime, subChainStart)
			return
		}()
	}
}

// processChainStartTime initializes a series of deposits from the ChainStart deposits in the eth1
// deposit contract, initializes the beacon chain's state, and kicks off the beacon chain.
func (c *ChainService) processChainStartTime(genesisTime time.Time, chainStartSub event.Subscription) {
	initialDeposits := c.web3Service.ChainStartDeposits()
	if err := c.initializeBeaconChain(genesisTime, initialDeposits, c.web3Service.ChainStartETH1Data()); err != nil {
		log.Fatalf("Could not initialize beacon chain: %v", err)
	}
	c.stateInitializedFeed.Send(genesisTime)
	chainStartSub.Unsubscribe()
}

// initializes the state and genesis block of the beacon chain to persistent storage
// based on a genesis timestamp value obtained from the ChainStart event emitted
// by the ETH1.0 Deposit Contract and the POWChain service of the node.
func (c *ChainService) initializeBeaconChain(genesisTime time.Time, deposits []*ethpb.Deposit, eth1data *ethpb.Eth1Data) error {
	_, span := trace.StartSpan(context.Background(), "beacon-chain.ChainService.initializeBeaconChain")
	defer span.End()
	log.Info("ChainStart time reached, starting the beacon chain!")
	c.genesisTime = genesisTime
	unixTime := uint64(genesisTime.Unix())
	if err := c.beaconDB.InitializeState(c.ctx, unixTime, deposits, eth1data); err != nil {
		return errors.Wrap(err, "could not initialize beacon state to disk")
	}
	beaconState, err := c.beaconDB.HeadState(c.ctx)
	if err != nil {
		return errors.Wrap(err, "could not attempt fetch beacon state")
	}

	stateRoot, err := ssz.HashTreeRoot(beaconState)
	if err != nil {
		return errors.Wrap(err, "could not hash beacon state")
	}

	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := c.beaconDB.SaveBlock(genBlock); err != nil {
		return errors.Wrap(err, "could not save genesis block to disk")
	}
	//if err := c.beaconDB.UpdateChainHead(ctx, genBlock, beaconState); err != nil {
	//	return errors.Wrap(err, "could not set chain head")
	//}
	if err := c.forkChoiceStore.GensisStore(beaconState); err != nil {
		return errors.Wrap(err, "could not start gensis store for fork choice")
	}

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

// CanonicalBlockFeed returns a channel that is written to
// whenever a new block is determined to be canonical in the chain.
func (c *ChainService) CanonicalBlockFeed() *event.Feed {
	return c.canonicalBlockFeed
}

// StateInitializedFeed returns a feed that is written to
// when the beacon state is first initialized.
func (c *ChainService) StateInitializedFeed() *event.Feed {
	return c.stateInitializedFeed
}

// FinalizedBlock returns the latest finalized block tracked in fork choice service.
func (c *ChainService) FinalizedBlock() (*ethpb.BeaconBlock, error) {
	checkpt := c.forkChoiceStore.FinalizedCheckpt()
	finalizedBlk, err := c.beaconDB.Block(bytesutil.ToBytes32(checkpt.Root))
	if err != nil {
		return nil, err
	}
	if finalizedBlk == nil {
		return nil, fmt.Errorf("finalized block %#x does not exist in db", hex.EncodeToString(checkpt.Root))
	}
	return finalizedBlk, nil
}

// FinalizedState returns the latest finalized state tracked in fork choice service.
func (c *ChainService) FinalizedState(ctx context.Context) (*pb.BeaconState, error) {
	checkpt := c.forkChoiceStore.FinalizedCheckpt()
	finalizedState, err := c.beaconDB.ForkChoiceState(ctx, checkpt.Root)
	if err != nil {
		return nil, err
	}
	if finalizedState == nil {
		return nil, fmt.Errorf("finalized state %#x does not exist in db", hex.EncodeToString(checkpt.Root))
	}
	return finalizedState, nil
}

// FinalizedCheckpt returns the latest finalized checkpoint tracked in fork choice service.
func (c *ChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return c.forkChoiceStore.FinalizedCheckpt()
}

// JustifiedCheckpt returns the latest justified checkpoint tracked in fork choice service.
func (c *ChainService) JustifiedCheckpt() *ethpb.Checkpoint {
	return c.forkChoiceStore.JustifiedCheckpt()
}

// HeadSlot returns the slot of the head of the chain.
func (c *ChainService) HeadSlot() uint64 {
	return c.headSlot
}

// HeadRoot returns the root of the head of the chain.
func (c *ChainService) HeadRoot() []byte {
	return c.canonicalRoots[c.headSlot]
}

// HeadBlock returns the block of the head of the chain.
func (c *ChainService) HeadBlock() (*ethpb.BeaconBlock, error) {
	r := bytesutil.ToBytes32(c.canonicalRoots[c.headSlot])
	return c.beaconDB.Block(r)
}

// CanonicalRoot returns the canonical root of a given slot.
func (c *ChainService) CanonicalRoot(slot uint64) []byte {
	return c.canonicalRoots[slot]
}
