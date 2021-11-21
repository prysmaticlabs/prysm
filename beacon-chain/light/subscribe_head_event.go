package light

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	"github.com/prysmaticlabs/prysm/network/forks"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func (s *Service) subscribeHeadEvent(ctx context.Context) {
	stateChan := make(chan *feed.Event, 1)
	sub := s.cfg.StateNotifier.StateFeed().Subscribe(stateChan)
	defer sub.Unsubscribe()
	for {
		select {
		case ev := <-stateChan:
			if ev.Type == statefeed.NewHead {
				head, beaconState, err := s.getChainHeadAndState(ctx)
				if err != nil {
					log.Error(err)
					continue
				}
				if err := s.onHead(ctx, head, beaconState); err != nil {
					log.Error(err)
					continue
				}
			}
		case <-sub.Err():
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) getChainHeadAndState(ctx context.Context) (block.SignedBeaconBlock, state.BeaconState, error) {
	head, err := s.cfg.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, nil, err
	}
	if head == nil || head.IsNil() {
		return nil, nil, errors.New("head block is nil")
	}
	st, err := s.cfg.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, nil, errors.New("head state is nil")
	}
	if st == nil || st.IsNil() {
		return nil, nil, err
	}
	return head, st, nil
}

func (s *Service) onHead(ctx context.Context, head block.SignedBeaconBlock, postState state.BeaconStateAltair) error {
	if _, ok := postState.InnerStateUnsafe().(*ethpb.BeaconStateAltair); !ok {
		return errors.New("expected an Altair beacon state")
	}
	blk := head.Block()
	tb, err := ssz.NewTreeBackedState(postState)
	if err != nil {
		return err
	}
	header, err := block.BeaconBlockHeaderFromBlockInterface(blk)
	if err != nil {
		return err
	}
	finalityBranch, _, err := tb.Proof(finalizedCheckpointStateIndex)
	if err != nil {
		return err
	}
	nextSyncCommitteeBranch, gIndex, err := tb.Proof(nextSyncCommitteeStateIndex)
	if err != nil {
		return err
	}
	stRoot, err := postState.HashTreeRoot(ctx)
	if err != nil {
		return err
	}
	blkRoot, err := blk.HashTreeRoot()
	if err != nil {
		return err
	}
	log.Infof("On head, generating sync committee proof for root %#x and index %d, block root %#x, header state root %#x", stRoot[:], gIndex, blkRoot, header.StateRoot)
	nextSyncCommittee, err := postState.NextSyncCommittee()
	if err != nil {
		return err
	}

	s.lock.Lock()
	s.prevHeadData[blkRoot] = &ethpb.SyncAttestedData{
		Header:                  header,
		FinalityCheckpoint:      postState.FinalizedCheckpoint(),
		FinalityBranch:          finalityBranch,
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
	}
	s.lock.Unlock()
	syncAttestedBlockRoot, err := helpers.BlockRootAtSlot(postState, postState.Slot()-1)
	if err != nil {
		return err
	}

	fork, err := forks.Fork(slots.ToEpoch(blk.Slot()))
	if err != nil {
		return err
	}
	syncAggregate, err := blk.Body().SyncAggregate()
	if err != nil {
		return err
	}
	sigData := &signatureData{
		slot:          blk.Slot(),
		forkVersion:   fork.CurrentVersion,
		syncAggregate: syncAggregate,
	}

	s.lock.Lock()
	syncAttestedData, ok := s.prevHeadData[bytesutil.ToBytes32(syncAttestedBlockRoot)]
	if !ok {
		s.lock.Unlock()
		log.Info("Got useless data, skipping")
		return nil // Useless data.
	}
	s.lock.Unlock()
	commmitteePeriodWithFinalized, err := s.persistBestFinalizedUpdate(ctx, syncAttestedData, sigData)
	if err != nil {
		return err
	}

	if err := s.persistBestNonFinalizedUpdate(ctx, syncAttestedData, sigData, commmitteePeriodWithFinalized); err != nil {
		return err
	}

	s.lock.Lock()
	if len(s.prevHeadData) > PrevDataMaxSize {
		for k := range s.prevHeadData {
			delete(s.prevHeadData, k)
			if len(s.prevHeadData) <= PrevDataMaxSize {
				break
			}
		}
	}
	s.lock.Unlock()
	return nil
}
