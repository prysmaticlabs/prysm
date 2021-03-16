package blockchain

import (
	"context"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func SaveGenesisData(ctx context.Context, genesisState iface.BeaconState, beaconDB db.HeadAccessDatabase) (*eth.SignedBeaconBlock, error) {
	stateRoot, err := genesisState.HashTreeRoot(ctx)
	if err != nil {
		return nil, err
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := genesisBlk.Block.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get genesis block root")
	}
	if err := beaconDB.SaveBlock(ctx, genesisBlk); err != nil {
		return nil, errors.Wrap(err, "could not save genesis block")
	}
	if err := beaconDB.SaveState(ctx, genesisState, genesisBlkRoot); err != nil {
		return nil, errors.Wrap(err, "could not save genesis state")
	}
	if err := beaconDB.SaveStateSummary(ctx, &pb.StateSummary{
		Slot: 0,
		Root: genesisBlkRoot[:],
	}); err != nil {
		return nil, err
	}

	if err := beaconDB.SaveHeadBlockRoot(ctx, genesisBlkRoot); err != nil {
		return nil, errors.Wrap(err, "could not save head block root")
	}
	if err := beaconDB.SaveGenesisBlockRoot(ctx, genesisBlkRoot); err != nil {
		return nil, errors.Wrap(err, "could not save genesis block root")
	}

	genesisCheckpoint := &eth.Checkpoint{Root: genesisBlkRoot[:]}
	if err := beaconDB.SaveJustifiedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return nil, errors.Wrap(err, "could not save justified checkpoint")
	}
	if err := beaconDB.SaveFinalizedCheckpoint(ctx, genesisCheckpoint); err != nil {
		return nil, errors.Wrap(err, "could not save finalized checkpoint")
	}


	return genesisBlk, nil
}
