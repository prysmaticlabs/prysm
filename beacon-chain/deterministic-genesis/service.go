// Package interopcoldstart allows for spinning up a deterministic-genesis
// local chain without the need for eth1 deposits useful for
// local client development and interoperability testing.
package interopcoldstart

import (
	"context"
	"io/ioutil"
	"math/big"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/prysmaticlabs/prysm/runtime/interop"
	"github.com/prysmaticlabs/prysm/time/slots"
)

var _ runtime.Service = (*Service)(nil)
var _ depositcache.DepositFetcher = (*Service)(nil)
var _ powchain.ChainStartFetcher = (*Service)(nil)

// Service spins up an client interoperability service that handles responsibilities such
// as kickstarting a genesis state for the beacon node from cli flags or a genesis.ssz file.
type Service struct {
	cfg                *Config
	ctx                context.Context
	cancel             context.CancelFunc
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

// NewService is an interoperability testing service to inject a deterministically generated genesis state
// into the beacon chain database and running services at start up. This service should not be used in production
// as it does not have any value other than ease of use for testing purposes.
func NewService(ctx context.Context, cfg *Config) *Service {
	log.Warn("Saving generated genesis state in database for interop testing")
	ctx, cancel := context.WithCancel(ctx)

	s := &Service{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	if s.cfg.GenesisPath != "" {
		data, err := ioutil.ReadFile(s.cfg.GenesisPath)
		if err != nil {
			log.Fatalf("Could not read pre-loaded state: %v", err)
		}
		genesisState := &ethpb.BeaconState{}
		if err := genesisState.UnmarshalSSZ(data); err != nil {
			log.Fatalf("Could not unmarshal pre-loaded state: %v", err)
		}
		genesisTrie, err := v1.InitializeFromProto(genesisState)
		if err != nil {
			log.Fatalf("Could not get state trie: %v", err)
		}
		if err := s.saveGenesisState(ctx, genesisTrie); err != nil {
			log.Fatalf("Could not save interop genesis state %v", err)
		}
		return s
	}

	// Save genesis state in db
	genesisState, _, err := interop.GenerateGenesisState(ctx, s.cfg.GenesisTime, s.cfg.NumValidators)
	if err != nil {
		log.Fatalf("Could not generate interop genesis state: %v", err)
	}
	genesisTrie, err := v1.InitializeFromProto(genesisState)
	if err != nil {
		log.Fatalf("Could not get state trie: %v", err)
	}
	if s.cfg.GenesisTime == 0 {
		// Generated genesis time; fetch it
		s.cfg.GenesisTime = genesisTrie.GenesisTime()
	}
	gRoot, err := genesisTrie.HashTreeRoot(s.ctx)
	if err != nil {
		log.Fatalf("Could not hash tree root genesis state: %v", err)
	}
	go slots.CountdownToGenesis(ctx, time.Unix(int64(s.cfg.GenesisTime), 0), s.cfg.NumValidators, gRoot)

	if err := s.saveGenesisState(ctx, genesisTrie); err != nil {
		log.Fatalf("Could not save interop genesis state %v", err)
	}

	return s
}

// Start initializes the genesis state from configured flags.
func (_ *Service) Start() {
}

// Stop does nothing.
func (_ *Service) Stop() error {
	return nil
}

// Status always returns nil.
func (_ *Service) Status() error {
	return nil
}

// AllDeposits mocks out the deposit cache functionality for interop.
func (_ *Service) AllDeposits(_ context.Context, _ *big.Int) []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartEth1Data mocks out the powchain functionality for interop.
func (_ *Service) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

// PreGenesisState returns an empty beacon state.
func (_ *Service) PreGenesisState() state.BeaconState {
	s, err := v1.InitializeFromProto(&ethpb.BeaconState{})
	if err != nil {
		panic("could not initialize state")
	}
	return s
}

// ClearPreGenesisData --
func (_ *Service) ClearPreGenesisData() {
	// no-op
}

// DepositByPubkey mocks out the deposit cache functionality for interop.
func (_ *Service) DepositByPubkey(_ context.Context, _ []byte) (*ethpb.Deposit, *big.Int) {
	return &ethpb.Deposit{}, nil
}

// DepositsNumberAndRootAtHeight mocks out the deposit cache functionality for interop.
func (_ *Service) DepositsNumberAndRootAtHeight(_ context.Context, _ *big.Int) (uint64, [32]byte) {
	return 0, [32]byte{}
}

// FinalizedDeposits mocks out the deposit cache functionality for interop.
func (_ *Service) FinalizedDeposits(_ context.Context) *depositcache.FinalizedDeposits {
	return nil
}

// NonFinalizedDeposits mocks out the deposit cache functionality for interop.
func (_ *Service) NonFinalizedDeposits(_ context.Context, _ *big.Int) []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

func (s *Service) saveGenesisState(ctx context.Context, genesisState state.BeaconState) error {
	if err := s.cfg.BeaconDB.SaveGenesisData(ctx, genesisState); err != nil {
		return err
	}

	s.chainStartDeposits = make([]*ethpb.Deposit, genesisState.NumValidators())

	for i := types.ValidatorIndex(0); uint64(i) < uint64(genesisState.NumValidators()); i++ {
		pk := genesisState.PubkeyAtIndex(i)
		s.chainStartDeposits[i] = &ethpb.Deposit{
			Data: &ethpb.Deposit_Data{
				PublicKey: pk[:],
			},
		}
	}
	return nil
}
