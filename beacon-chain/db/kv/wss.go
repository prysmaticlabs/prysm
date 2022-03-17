package kv

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/detect"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// SaveOrigin loads an ssz serialized Block & BeaconState from an io.Reader
// (ex: an open file) prepares the database so that the beacon node can begin
// syncing, using the provided values as their point of origin. This is an alternative
// to syncing from genesis, and should only be run on an empty database.
func (s *Store) SaveOrigin(ctx context.Context, stateReader, blockReader io.Reader) error {
	sb, err := ioutil.ReadAll(stateReader)
	if err != nil {
		return errors.Wrap(err, "failed to read origin state bytes")
	}
	bb, err := ioutil.ReadAll(blockReader)
	if err != nil {
		return errors.Wrap(err, "error reading block given to SaveOrigin")
	}

	cf, err := detect.ByState(sb)
	if err != nil {
		return errors.Wrap(err, "failed to detect config and fork for origin state")
	}
	bs, err := cf.UnmarshalBeaconState(sb)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal origin state")
	}
	wblk, err := cf.UnmarshalBeaconBlock(bb)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal origin SignedBeaconBlock")
	}

	blockRoot, err := wblk.Block().HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute HashTreeRoot of checkpoint block")
	}
	// save block
	if err := s.SaveBlock(ctx, wblk); err != nil {
		return errors.Wrap(err, "could not save checkpoint block")
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
	slotEpoch, err := wblk.Block().Slot().SafeDivSlot(params.BeaconConfig().SlotsPerEpoch)
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
