package light

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

func (s *Service) subscribeEvents(ctx context.Context) {
	c := make(chan *feed.Event, 1)
	sub := s.cfg.StateNotifier.StateFeed().Subscribe(c)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-c:
			if ev.Type == statefeed.BlockProcessed {
				d, ok := ev.Data.(*statefeed.BlockProcessedData)
				if !ok {
					continue
				}
				if err := s.newBlock(ctx, d.BlockRoot, d.SignedBlock, d.PostState); err != nil {
					log.WithError(err).Error("Could not process new block")
					continue
				}
			} else if ev.Type == statefeed.FinalizedCheckpoint {
				_, ok := ev.Data.(*ethpbv1.EventFinalizedCheckpoint)
				if !ok {
					continue
				}
			}
		case err := <-sub.Err():
			log.WithError(err).Error("Could not subscribe to state notifier")
			return
		case <-ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		}
	}
}

func (s *Service) newBlock(ctx context.Context, r [32]byte, blk block.SignedBeaconBlock, st state.BeaconState) error {
	if st.Version() == version.Phase0 {
		return nil
	}
	h, err := blk.Header()
	if err != nil {
		return err
	}
	com, err := st.NextSyncCommittee()
	if err != nil {
		return err
	}
	b, err := st.NextSyncCommitteeProof()
	if err != nil {
		return err
	}
	update := &ethpb.LightClientUpdate{
		AttestedHeader:          h.Header,
		NextSyncCommittee:       com,
		NextSyncCommitteeBranch: b,
		ForkVersion:             st.Fork().CurrentVersion,
	}
	if err := s.saveUpdate(r[:], update); err != nil {
		return err
	}

	parentRoot := blk.Block().ParentRoot()
	update, err = s.getUpdate(parentRoot)
	if err != nil {
		return err
	}
	if update == nil {
		return nil
	}
	agg, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		return err
	}
	update.SyncAggregate = agg
	return s.cfg.BeaconDB.SaveLightClientUpdate(ctx, update)
}

func (s *Service) newFinalized(ctx context.Context) {

}
