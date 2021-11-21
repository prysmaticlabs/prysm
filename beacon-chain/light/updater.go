package light

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
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

func (s *Service) persistBestFinalizedUpdate(ctx context.Context, syncAttestedData *ethpb.SyncAttestedData, sigData *signatureData) (uint64, error) {
	finalizedEpoch := syncAttestedData.FinalityCheckpoint.Epoch

	s.lock.RLock()
	finalizedData := s.finalizedByEpoch[finalizedEpoch]
	s.lock.RUnlock()

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

	s.lock.RLock()
	prevBestUpdate := s.bestUpdateByPeriod[committeePeriod]
	s.lock.RUnlock()

	if prevBestUpdate == nil || isBetterUpdate(prevBestUpdate, newUpdate) {
		s.lock.Lock()
		s.bestUpdateByPeriod[committeePeriod] = newUpdate
		s.lock.Unlock()
	}

	s.lock.RLock()
	prevLatestUpdate := s.latestFinalizedUpdate
	s.lock.RUnlock()

	if prevLatestUpdate == nil || isLatestBestFinalizedUpdate(prevLatestUpdate, newUpdate) {
		s.lock.Lock()
		s.latestFinalizedUpdate = newUpdate
		s.lock.Unlock()
		log.Info("Putting latest best finalized update")
		rt, err := newUpdate.NextSyncCommittee.HashTreeRoot()
		if err != nil {
			return 0, err
		}
		log.Infof("Header state root %#x, state hash tree root %#x", newUpdate.Header.StateRoot, newUpdate.Header.StateRoot)
		log.Infof("Generating proof against root %#x with gindex %d and leaf root %#x", newUpdate.Header.StateRoot, 55, rt)
		log.Info("-----")
		log.Infof("Proof with length %d", len(newUpdate.NextSyncCommitteeBranch))
		for _, elem := range newUpdate.NextSyncCommitteeBranch {
			log.Infof("%#x", bytesutil.Trunc(elem))
		}
		log.Info("-----")
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
		s.lock.RLock()
		prevBestUpdate := s.bestUpdateByPeriod[committeePeriod]
		s.lock.RUnlock()
		if prevBestUpdate == nil || isBetterUpdate(prevBestUpdate, newUpdate) {
			s.lock.Lock()
			s.bestUpdateByPeriod[committeePeriod] = newUpdate
			s.lock.Unlock()
		}
	}

	// Store the latest update here overall. Not checking it's the best
	s.lock.RLock()
	prevLatestUpdate := s.latestNonFinalizedUpdate
	s.lock.RUnlock()

	if prevLatestUpdate == nil || isLatestBestNonFinalizedUpdate(prevLatestUpdate, newUpdate) {
		// TODO: Don't store nextCommittee, that can be fetched through getBestUpdates()
		s.lock.Lock()
		s.latestNonFinalizedUpdate = newUpdate
		s.lock.Unlock()
	}
	return nil
}
