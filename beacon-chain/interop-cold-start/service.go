// Package interopcoldstart allows for spinning up a deterministic
// local chain without the need for eth1 deposits useful for
// local client development and interoperability testing.
package interopcoldstart

import (
	"context"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
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
	beaconDB           db.HeadAccessDatabase
	powchain           powchain.Service
	depositCache       *depositcache.DepositCache
	genesisPath        string
	chainStartDeposits []*ethpb.Deposit
}

// Config options for the interop service.
type Config struct {
	GenesisTime   uint64
	NumValidators uint64
	BeaconDB      db.HeadAccessDatabase
	DepositCache  *depositcache.DepositCache
	GenesisPath   string
}

// NewColdStartService is an interoperability testing service to inject a deterministically generated genesis state
// into the beacon chain database and running services at start up. This service should not be used in production
// as it does not have any value other than ease of use for testing purposes.
func NewColdStartService(ctx context.Context, cfg *Config) *Service {
	log.Warn("Saving generated genesis state in database for interop testing")
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
		if err := genesisState.UnmarshalSSZ(data); err != nil {
			log.Fatalf("Could not unmarshal pre-loaded state: %v", err)
		}
		genesisTrie, err := stateTrie.InitializeFromProto(genesisState)
		if err != nil {
			log.Fatalf("Could not get state trie: %v", err)
		}
		if err := s.saveGenesisState(ctx, genesisTrie); err != nil {
			log.Fatalf("Could not save interop genesis state %v", err)
		}
		return s
	}

	// Save genesis state in db
	genesisState, _, err := interop.GenerateGenesisState(s.genesisTime, s.numValidators)
	if err != nil {
		log.Fatalf("Could not generate interop genesis state: %v", err)
	}
	genesisTrie, err := stateTrie.InitializeFromProto(genesisState)
	if err != nil {
		log.Fatalf("Could not get state trie: %v", err)
	}
	if s.genesisTime == 0 {
		// Generated genesis time; fetch it
		s.genesisTime = genesisTrie.GenesisTime()
	}
	go slotutil.CountdownToGenesis(ctx, time.Unix(int64(s.genesisTime), 0), s.numValidators)

	if err := s.saveGenesisState(ctx, genesisTrie); err != nil {
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
func (s *Service) AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
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

// PreGenesisState returns an empty beacon state.
func (s *Service) PreGenesisState() *stateTrie.BeaconState {
	return &stateTrie.BeaconState{}
}

// ClearPreGenesisData --
func (s *Service) ClearPreGenesisData() {
	//no-op
}

// DepositByPubkey mocks out the deposit cache functionality for interop.
func (s *Service) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	return &ethpb.Deposit{}, big.NewInt(1)
}

// DepositsNumberAndRootAtHeight mocks out the deposit cache functionality for interop.
func (s *Service) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	return 0, [32]byte{}
}

// FinalizedDeposits mocks out the deposit cache functionality for interop.
func (s *Service) FinalizedDeposits(ctx context.Context) *depositcache.FinalizedDeposits {
	return nil
}

// NonFinalizedDeposits mocks out the deposit cache functionality for interop.
func (s *Service) NonFinalizedDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

func (s *Service) saveGenesisState(ctx context.Context, genesisState *stateTrie.BeaconState) error {
	s.chainStartDeposits = make([]*ethpb.Deposit, genesisState.NumValidators())
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}

	if err := s.beaconDB.SaveBlock(ctx, genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 0,
		Root: genesisBlkRoot[:],
	}); err != nil {
		return err
	}
	if err := s.beaconDB.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis block root")
	}
	if err := s.beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	genesisCheckpoint := &ethpb.Checkpoint{Root: genesisBlkRoot[:]}
	if err := s.beaconDB.SaveJustifiedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could not save justified checkpoint")
	}
	if err := s.beaconDB.SaveFinalizedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}

	pubKeys := make([][48]byte, 0, genesisState.NumValidators())
	indices := make([]uint64, 0, genesisState.NumValidators())
	for i := uint64(0); i < uint64(genesisState.NumValidators()); i++ {
		pk := genesisState.PubkeyAtIndex(i)
		pubKeys = append(pubKeys, pk)
		indices = append(indices, i)
		s.chainStartDeposits[i] = &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey: pk[:],
			},
		}
	}
	return nil
}
