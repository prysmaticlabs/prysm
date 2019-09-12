package interopcoldstart

import (
	"context"
	"io/ioutil"
	"math/big"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/interop"
)

var _ = shared.Service(&Service{})
var _ = depositcache.DepositFetcher(&Service{})
var _ = powchain.ChainStartFetcher(&Service{})

// Service spins up an client interoperability service that handles responsibilities such
// as kickstarting a genesis state for the beacon node from cli flags or a genesis.ssz file.
type Service struct {
	ctx                context.Context
	cancel             context.CancelFunc
	genesisTime        uint64
	numValidators      uint64
	beaconDB           db.Database
	powchain           powchain.Service
	depositCache       *depositcache.DepositCache
	genesisPath        string
	chainStartDeposits []*ethpb.Deposit
}

// Config options for the interop service.
type Config struct {
	GenesisTime   uint64
	NumValidators uint64
	BeaconDB      db.Database
	DepositCache  *depositcache.DepositCache
	GenesisPath   string
}

// NewColdStartService is an interoperability testing service to inject a deterministically generated genesis state
// into the beacon chain database and running services at start up. This service should not be used in production
// as it does not have any value other than ease of use for testing purposes.
func NewColdStartService(ctx context.Context, cfg *Config) *Service {
	log.Warn("Saving generated genesis state in database for interop testing.")
	ctx, cancel := context.WithCancel(ctx)

	s := &Service{
		ctx:           ctx,
		cancel:        cancel,
		genesisTime:   cfg.GenesisTime,
		numValidators: cfg.NumValidators,
		beaconDB:      cfg.BeaconDB,
		depositCache:  cfg.DepositCache,
		genesisPath:   cfg.GenesisPath,
	}

	if s.genesisPath != "" {
		data, err := ioutil.ReadFile(s.genesisPath)
		if err != nil {
			log.Fatalf("Could not read pre-loaded state: %v", err)
		}
		genesisState := &pb.BeaconState{}
		if err := ssz.Unmarshal(data, genesisState); err != nil {
			log.Fatalf("Could not unmarshal pre-loaded state: %v", err)
		}
		if err := s.saveGenesisState(ctx, genesisState); err != nil {
			log.Fatalf("Could not save interop genesis state %v", err)
		}
		return s
	}

	// Save genesis state in db
	genesisState, _, err := interop.GenerateGenesisState(s.genesisTime, s.numValidators)
	if err != nil {
		log.Fatalf("Could not generate interop genesis state: %v", err)
	}
	if err := s.saveGenesisState(ctx, genesisState); err != nil {
		log.Fatalf("Could not save interop genesis state %v", err)
	}

	return s
}

// Start initializes the genesis state from configured flags.
func (s *Service) Start() {
}

// Stop does nothing.
func (s *Service) Stop() error {
	return nil
}

// Status always returns nil.
func (s *Service) Status() error {
	return nil
}

// AllDeposits mocks out the deposit cache functionality for interop.
func (s *Service) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartDeposits mocks out the powchain functionality for interop.
func (s *Service) ChainStartDeposits() []*ethpb.Deposit {
	return s.chainStartDeposits
}

// ChainStartEth1Data mocks out the powchain functionality for interop.
func (s *Service) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

// ChainStartFeed mocks out the powchain functionality for interop.
func (s *Service) ChainStartFeed() *event.Feed {
	return new(event.Feed)
}

// DepositByPubkey mocks out the deposit cache functionality for interop.
func (s *Service) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	return &ethpb.Deposit{}, big.NewInt(1)
}

// DepositsNumberAndRootAtHeight mocks out the deposit cache functionality for interop.
func (s *Service) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	return 0, [32]byte{}
}

func (s *Service) saveGenesisState(ctx context.Context, genesisState *pb.BeaconState) error {
	s.chainStartDeposits = make([]*ethpb.Deposit, len(genesisState.Validators))
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
	genesisCheckpoint := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}
	if err := s.beaconDB.SaveJustifiedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could save justified checkpoint")
	}
	if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could save finalized checkpoint")
	}

	for i, v := range genesisState.Validators {
		if err := s.beaconDB.SaveValidatorIndex(ctx, bytesutil.ToBytes48(v.PublicKey), uint64(i)); err != nil {
			return errors.Wrapf(err, "could not save validator index: %d", i)
		}
		s.chainStartDeposits[i] = &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey: v.PublicKey,
			},
		}
	}
	return nil
}
