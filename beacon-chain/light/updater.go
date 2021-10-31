package light

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network/forks"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

// Precomputed values for generalized indices.
const (
	FinalizedRootIndex     = 105
	NextSyncCommitteeIndex = 55
	PrevDataMaxSize        = 64
)

var log = logrus.WithField("prefix", "light")

type signatureData struct {
	slot          types.Slot
	forkVersion   []byte
	syncAggregate *ethpb.SyncAggregate
}

func (s *Service) onHead(ctx context.Context, postState state.BeaconStateAltair, head block.BeaconBlock) error {
	log.Info("Head updated, persisting best updates")
	innerState, ok := postState.InnerStateUnsafe().(*ethpb.BeaconStateAltair)
	if !ok {
		return errors.New("not altair")
	}
	tr, err := innerState.GetTree()
	if err != nil {
		return err
	}
	header, err := block.BeaconBlockHeaderFromBlockInterface(head)
	if err != nil {
		return err
	}
	finalityBranch, err := tr.Prove(FinalizedRootIndex)
	if err != nil {
		return err
	}
	nextSyncCommitteeBranch, err := tr.Prove(NextSyncCommitteeIndex)
	if err != nil {
		return err
	}
	blkRoot, err := head.HashTreeRoot()
	if err != nil {
		return err
	}
	s.lock.Lock()
	s.prevHeadData[blkRoot] = &ethpb.SyncAttestedData{
		Header:                  header,
		FinalityCheckpoint:      innerState.FinalizedCheckpoint,
		FinalityBranch:          finalityBranch.Hashes,
		NextSyncCommittee:       innerState.NextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch.Hashes,
	}
	s.lock.Unlock()
	syncAttestedBlockRoot, err := helpers.BlockRootAtSlot(postState, innerState.Slot-1)
	if err != nil {
		return err
	}

	fork, err := forks.Fork(slots.ToEpoch(head.Slot()))
	if err != nil {
		return err
	}
	syncAggregate, err := head.Body().SyncAggregate()
	if err != nil {
		return err
	}
	sigData := &signatureData{
		slot:          head.Slot(),
		forkVersion:   fork.CurrentVersion,
		syncAggregate: syncAggregate,
	}
	// Recover attested data from prevData cache. If not found, this SyncAggregate is useless
	s.lock.Lock()
	syncAttestedData, ok := s.prevHeadData[bytesutil.ToBytes32(syncAttestedBlockRoot)]
	if !ok {
		s.lock.Unlock()
		return errors.New("useless")
	}
	s.lock.Unlock()
	commmitteePeriodWithFinalized, err := s.persistBestFinalizedUpdate(ctx, syncAttestedData, sigData)
	if err != nil {
		return err
	}

	// Then, store the best non finalized update per period
	if err := s.persistBestNonFinalizedUpdate(ctx, syncAttestedData, sigData, commmitteePeriodWithFinalized); err != nil {
		return err
	}
	// Prune old prevHeadData
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

func (s *Service) onFinalized(ctx context.Context, postState state.BeaconStateAltair, head block.BeaconBlock) error {
	log.Info("State finalized, persisting light client finalized checkpoint")
	innerState, ok := postState.InnerStateUnsafe().(*ethpb.BeaconStateAltair)
	if !ok {
		return errors.New("not altair")
	}
	header, err := block.BeaconBlockHeaderFromBlockInterface(head)
	if err != nil {
		return err
	}
	tr, err := innerState.GetTree()
	if err != nil {
		return err
	}
	nextSyncCommitteeBranch, err := tr.Prove(NextSyncCommitteeIndex)
	if err != nil {
		return err
	}
	nextSyncCommittee, err := postState.NextSyncCommittee()
	if err != nil {
		return err
	}
	return s.Database.SaveLightClientFinalizedCheckpoint(ctx, 0, &ethpb.LightClientFinalizedCheckpoint{
		Header:                  header,
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch.Hashes,
	})
}

func (s *Service) persistBestFinalizedUpdate(ctx context.Context, syncAttestedData *ethpb.SyncAttestedData, sigData *signatureData) (uint64, error) {
	finalizedEpoch := syncAttestedData.FinalityCheckpoint.Epoch
	_ = finalizedEpoch
	finalizedData, err := s.Database.LightClientFinalizedCheckpoint(ctx, finalizedEpoch)
	if err != nil {
		return 0, err
	}
	if finalizedData == nil {
		return 0, nil
	}
	committeePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(syncAttestedData.Header.Slot))
	signaturePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(sigData.slot))
	if committeePeriod != signaturePeriod {
		return 0, nil
	}
	newUpdate := &ethpb.LightClientUpdate{
		Header:                  finalizedData.Header,
		NextSyncCommittee:       finalizedData.NextSyncCommittee,
		NextSyncCommitteeBranch: finalizedData.NextSyncCommitteeBranch,
		FinalityHeader:          syncAttestedData.Header,
		FinalityBranch:          syncAttestedData.FinalityBranch,
		SyncCommitteeBits:       sigData.syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature:  sigData.syncAggregate.SyncCommitteeSignature,
		ForkVersion:             sigData.forkVersion,
	}
	prevBestUpdate, err := s.Database.LightClientBestUpdateForPeriod(ctx, committeePeriod)
	if err != nil {
		return 0, err
	}
	if prevBestUpdate == nil || isBetterUpdate(prevBestUpdate, newUpdate) {
		if err := s.Database.SaveLightClientBestUpdateForPeriod(ctx, committeePeriod, newUpdate); err != nil {
			return 0, err
		}
	}
	prevLatestUpdate, err := s.Database.LightClientLatestFinalizedUpdate(ctx)
	if err != nil {
		return 0, err
	}
	if prevLatestUpdate == nil || isLatestBestFinalizedUpdate(prevLatestUpdate, newUpdate) {
		if err := s.Database.SaveLightClientLatestFinalizedUpdate(ctx, newUpdate); err != nil {
			return 0, err
		}
	}
	return committeePeriod, nil
}

func (s *Service) persistBestNonFinalizedUpdate(ctx context.Context, syncAttestedData *ethpb.SyncAttestedData, sigData *signatureData, period uint64) error {
	committeePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(syncAttestedData.Header.Slot))
	signaturePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(sigData.slot))
	if committeePeriod != signaturePeriod {
		return nil
	}

	newUpdate := &ethpb.LightClientUpdate{
		Header:                  syncAttestedData.Header,
		NextSyncCommittee:       syncAttestedData.NextSyncCommittee,
		NextSyncCommitteeBranch: syncAttestedData.NextSyncCommitteeBranch,
		FinalityHeader:          nil,
		FinalityBranch:          nil,
		SyncCommitteeBits:       sigData.syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature:  sigData.syncAggregate.SyncCommitteeSignature,
		ForkVersion:             sigData.forkVersion,
	}

	// Optimization: If there's already a finalized update for this committee period, no need to
	// create a non-finalized update>
	if committeePeriod != period {
		prevBestUpdate, err := s.Database.LightClientBestUpdateForPeriod(ctx, committeePeriod)
		if err != nil {
			return err
		}
		if prevBestUpdate == nil || isBetterUpdate(prevBestUpdate, newUpdate) {
			if err := s.Database.SaveLightClientBestUpdateForPeriod(ctx, committeePeriod, newUpdate); err != nil {
				return err
			}
		}
	}

	// Store the latest update here overall. Not checking it's the best
	prevLatestUpdate, err := s.Database.LightClientLatestNonFinalizedUpdate(ctx)
	if err != nil {
		return err
	}
	if prevLatestUpdate == nil || isLatestBestNonFinalizedUpdate(prevLatestUpdate, newUpdate) {
		// TODO: Don't store nextCommittee, that can be fetched through getBestUpdates()
		if err := s.Database.SaveLightClientLatestNonFinalizedUpdate(ctx, newUpdate); err != nil {
			return err
		}
	}
	return nil
}
