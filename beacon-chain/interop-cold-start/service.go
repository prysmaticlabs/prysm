package interop_cold_start

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/interop"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

)

var _ = shared.Service(&Service{})

type Service struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	genesisTime   uint64
	numValidators uint64
	beaconDB             db.Database
}

type Config struct {
	GenesisTime   uint64
	NumValidators uint64
	BeaconDB             db.Database
}

// NewColdStartService is an interoperability testing service to inject a deterministically generated genesis state
// into the beacon chain database and running services at start up. This service should not be used in production
// as it does not have any value other than ease of use for testing purposes.
func NewColdStartService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx: ctx,
		cancel: cancel,
		genesisTime: cfg.GenesisTime,
		numValidators: cfg.NumValidators,
		beaconDB:         cfg.BeaconDB,
	}
}

// Start initializes the genesis state from configured flags.
func (s *Service) Start() {
	ctx := context.TODO()
	log.Warn("Injecting generated genesis state for interop testing.")

	// Save genesis state in db
	genesisState, err := interop.GenerateGenesisState(s.genesisTime, s.numValidators)
	if err != nil {
		log.Fatalf("Could not generate interop genesis state: %v", err)
	}
	if err := s.saveGenesisState(ctx, genesisState); err != nil {
		log.Fatalf("Could not save interop genesis state %v", err)
	}
}

// Stop does nothing.
func (s *Service) Stop() error {
	return nil
}

// Status always returns nil.
func (s *Service) Status() error {
	return nil
}

func (s *Service) saveGenesisState(ctx context.Context, genesisState *pb.BeaconState) error {
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

	return nil
}