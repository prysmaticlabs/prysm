package kv

import (
	"context"
	"io/ioutil"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	dbIface "github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// SaveGenesisData bootstraps the beaconDB with a given genesis state.
func (s *Store) SaveGenesisData(ctx context.Context, genesisState iface.BeaconState) error {
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}
	if err := s.SaveBlock(ctx, genesisBlk); err != nil {
		return errors.Wrap(err, "could not save genesis block")
	}
	if err := s.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis state")
	}
	if err := s.SaveStateSummary(ctx, &pbp2p.StateSummary{
		Slot: 0,
		Root: genesisBlkRoot[:],
	}); err != nil {
		return err
	}

	if err := s.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err := s.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return errors.Wrap(err, "could not save genesis block root")
	}

	genesisCheckpoint := &eth.Checkpoint{Root: genesisBlkRoot[:]}
	if err := s.SaveJustifiedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could not save justified checkpoint")
	}
	if err := s.SaveFinalizedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return errors.Wrap(err, "could not save finalized checkpoint")
	}

	return nil
}

// LoadGenesisFromFile loads a genesis state from a given file path, if no genesis exists already.
func (s *Store) LoadGenesisFromFile(ctx context.Context, filePath string) error {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	st := &pbp2p.BeaconState{}
	if err := st.UnmarshalSSZ(b); err != nil {
		return err
	}
	gs, err := state.InitializeFromProtoUnsafe(st)
	if err != nil {
		return err
	}
	existing, err := s.GenesisState(ctx)
	if err != nil {
		return err
	}
	// If some different genesis state existed already, return an error. The same genesis state is
	// considered a no-op.
	if existing != nil {
		a, err := existing.HashTreeRoot(ctx)
		if err != nil {
			return err
		}
		b, err := gs.HashTreeRoot(ctx)
		if err != nil {
			return err
		}
		if a == b {
			return nil
		}
		return dbIface.ErrExistingGenesisState
	}

	return s.SaveGenesisData(ctx, gs)
}

// EnsureEmbeddedGenesis checks that a genesis block has been generated when an embedded genesis
// state is used. If a genesis block does not exist, but a genesis does, then we should call
// SaveGenesisData on the existing genesis state.
func (s *Store) EnsureEmbeddedGenesis(ctx context.Context) error {
	gb, err := s.GenesisBlock(ctx)
	if err != nil {
		return err
	}
	if gb != nil {
		return nil
	}
	gs, err := s.GenesisState(ctx)
	if err != nil {
		return err
	}
	if gs != nil {
		return s.SaveGenesisData(ctx, gs)
	}
	return nil
}
