package kv

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/sniff"
)

// SaveOrigin loads an ssz serialized Block & BeaconState from an io.Reader
// (ex: an open file) prepares the database so that the beacon node can begin
// syncing, using the provided values as their point of origin. This is an alternative
// to syncing from genesis, and should only be run on an empty database.
func (s *Store) SaveOrigin(ctx context.Context, stateReader, blockReader io.Reader) error {
	// unmarshal both block and state before trying to save anything
	// so that we fail early if there is any issue with the ssz data
	sb, err := ioutil.ReadAll(stateReader)
	if err != nil {
		return errors.Wrap(err, "could not read origin state bytes from reader")
	}
	cf, err := sniff.ConfigForkForState(sb)
	if err != nil {
		return errors.Wrap(err, "could not sniff config+fork for origin state bytes")
	}
	state, err := sniff.BeaconStateForConfigFork(sb, cf)
	if err != nil {
		return errors.Wrap(err, "failed to initialize origin state w/ bytes + config+fork")
	}

	bb, err := ioutil.ReadAll(stateReader)
	if err != nil {
		return errors.Wrap(err, "could not read origin block bytes from reader")
	}
	wblk, err := sniff.BlockForConfigFork(bb, cf)
	if err != nil {
		return errors.Wrap(err, "failed to initialize origin block w/ bytes + config+fork")
	}
	blk := wblk.Block()

	// save block
	blockRoot, err := blk.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute HashTreeRoot of checkpoint block")
	}
	log.Infof("saving checkpoint block to db, w/ root=%#x", blockRoot)
	if err := s.SaveBlock(ctx, wblk); err != nil {
		return errors.Wrap(err, "could not save checkpoint block")
	}

	// save state
	log.Infof("calling SaveState w/ blockRoot=%x", blockRoot)
	if err = s.SaveState(ctx, state, blockRoot); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	if err = s.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: state.Slot(),
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
	slotEpoch, err := blk.Slot().SafeDivSlot(params.BeaconConfig().SlotsPerEpoch)
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
