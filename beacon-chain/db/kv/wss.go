package kv

import (
	"context"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	statev2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"io"
	"io/ioutil"
)

const SLOTS_PER_EPOCH = 32

// SaveStateToHead sets the current head state.
func (s *Store) SaveStateToHead(ctx context.Context, bs state.BeaconState) error {

	return nil
}

// SaveInitialCheckpointState loads an ssz serialized BeaconState from an io.Reader
// (ex: an open file) and sets the given state to the head of the chain.
func (s *Store) SaveCheckpointInitialState(ctx context.Context, stateReader io.Reader, blockReader io.Reader) error {
	// save block to database
	blk := &ethpb.SignedBeaconBlockAltair{}
	bb, err := ioutil.ReadAll(blockReader)
	if err != nil {
		return err
	}
	if err := blk.UnmarshalSSZ(bb); err != nil {
		return errors.Wrap(err, "could not unmarshal checkpoint block")
	}
	wblk, err := wrapper.WrappedAltairSignedBeaconBlock(blk)
	if err != nil {
		return errors.Wrap(err, "could not wrap checkpoint block")
	}
	if err := s.SaveBlock(ctx, wblk); err != nil {
		return errors.Wrap(err, "could not save checkpoint block")
	}
	blockRoot, err := blk.Block.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute HashTreeRoot of checkpoint block")
	}

	bs, err := statev2.InitializeFromSSZReader(stateReader)
	if err != nil {
		return errors.Wrap(err, "could not initialize checkpoint state from reader")
	}

	if err = s.SaveState(ctx, bs, blockRoot); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	if err = s.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: bs.Slot(),
		Root: blockRoot[:],
	}); err != nil {
		return err
	}
	if err = s.SaveHeadBlockRoot(ctx, blockRoot); err != nil {
		return errors.Wrap(err, "could not save head block root")
	}
	if err = s.SaveCheckpointInitialBlockRoot(ctx, blockRoot); err != nil {
		return err
	}

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
