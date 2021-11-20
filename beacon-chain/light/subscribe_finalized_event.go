package light

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func (s *Service) subscribeFinalizedEvent(ctx context.Context) {
	stateChan := make(chan *feed.Event, 1)
	sub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChan)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-stateChan:
			if ev.Type == statefeed.FinalizedCheckpoint {
				blk, beaconState, err := s.parseFinalizedEvent(ctx, ev.Data)
				if err != nil {
					log.Error(err)
					continue
				}
				if err := s.onFinalized(ctx, blk, beaconState); err != nil {
					log.Error(err)
					continue
				}
			}
		}
	}
}

func (s *Service) parseFinalizedEvent(
	ctx context.Context, eventData interface{},
) (block.SignedBeaconBlock, state.BeaconState, error) {
	finalizedCheckpoint, ok := eventData.(*v1.EventFinalizedCheckpoint)
	if !ok {
		return nil, nil, errors.New("expected finalized checkpoint event")
	}
	checkpointRoot := bytesutil.ToBytes32(finalizedCheckpoint.Block)
	blk, err := s.cfg.Database.Block(ctx, checkpointRoot)
	if err != nil {
		return nil, nil, err
	}
	if blk == nil || blk.IsNil() {
		return nil, nil, err
	}
	st, err := s.cfg.StateGen.StateByRoot(ctx, checkpointRoot)
	if err != nil {
		return nil, nil, err
	}
	if st == nil || st.IsNil() {
		return nil, nil, err
	}
	return blk, st, nil
}

func (s *Service) onFinalized(
	ctx context.Context, signedBlock block.SignedBeaconBlock, postState state.BeaconStateAltair,
) error {
	log.Info("Getting beacon state")
	innerState, ok := postState.InnerStateUnsafe().(*ethpb.BeaconStateAltair)
	if !ok {
		return errors.New("expected an Altair beacon state")
	}
	blk := signedBlock.Block()
	log.Info("Getting beacon block header")
	header, err := block.BeaconBlockHeaderFromBlockInterface(blk)
	if err != nil {
		return err
	}
	log.Info("Getting state tree")
	tr, err := innerState.GetTree()
	if err != nil {
		return err
	}
	log.Info("Getting next sync committee proof")
	nextSyncCommitteeBranch, err := tr.Prove(NextSyncCommitteeIndex)
	if err != nil {
		return err
	}
	log.Info("Getting next sync committee")
	nextSyncCommittee, err := postState.NextSyncCommittee()
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	log.Info("Writing finalized by epoch")
	currentEpoch := slots.ToEpoch(blk.Slot())
	s.finalizedByEpoch[currentEpoch] = &ethpb.LightClientFinalizedCheckpoint{
		Header:                  header,
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch.Hashes,
	}
	return nil
}
