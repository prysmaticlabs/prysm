package kv

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	statev2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
)

// SaveOrigin loads an ssz serialized Block & BeaconState from an io.Reader
// (ex: an open file) prepares the database so that the beacon node can begin
// syncing, using the provided values as their point of origin. This is an alternative
// to syncing from genesis, and should only be run on an empty database.
func (s *Store) SaveOrigin(ctx context.Context, stateReader, blockReader io.Reader) error {
	// unmarshal both block and state before trying to save anything
	// so that we fail early if there is any issue with the ssz data
	blk := &ethpb.SignedBeaconBlockAltair{}
	bb, err := ioutil.ReadAll(blockReader)
	if err != nil {
		return errors.Wrap(err, "error reading block given to SaveOrigin")
	}
	if err := blk.UnmarshalSSZ(bb); err != nil {
		return errors.Wrap(err, "could not unmarshal checkpoint block")
	}
	wblk, err := wrapper.WrappedAltairSignedBeaconBlock(blk)
	if err != nil {
		return errors.Wrap(err, "could not wrap checkpoint block")
	}
	bs, err := statev2.InitializeFromSSZReader(stateReader)
	if err != nil {
		return errors.Wrap(err, "could not initialize checkpoint state from reader")
	}

	// save block
	if err := s.SaveBlock(ctx, wblk); err != nil {
		return errors.Wrap(err, "could not save checkpoint block")
	}
	blockRoot, err := blk.Block.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute HashTreeRoot of checkpoint block")
	}

	// save state
	if err = s.SaveState(ctx, bs, blockRoot); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	if err = s.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: bs.Slot(),
		Root: blockRoot[:],
	}); err != nil {
		return errors.Wrap(err, "could not save state summary")
	}

	// save origin block root in special key, to be used when the canonical
	// origin (start of chain, ie alternative to genesis) block or state is needed
	if err = s.SaveOriginBlockRoot(ctx, blockRoot); err != nil {
		return errors.Wrap(err, "could not save origin block root")
	}

	// mark block as head of chain, so that processing will pick up from this point
	if err = s.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}

	// rebuild the checkpoint from the block
	// use it to mark the block as justified and finalized
	slotEpoch, err := blk.Block.Slot.SafeDivSlot(params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		return err
	}
	chkpt := &ethpb.Checkpoint{
		Epoch: types.Epoch(slotEpoch),
		Root:  blockRoot[:],
	}
	if err = s.SaveJustifiedCheckpoint(ctx, chkpt); err != nil {
		return errors.Wrap(err, "could not mark checkpoint sync block as justified")
	}
	if err = s.SaveFinalizedCheckpoint(ctx, chkpt); err != nil {
		return errors.Wrap(err, "could not mark checkpoint sync block as finalized")
	}

	return nil
}
