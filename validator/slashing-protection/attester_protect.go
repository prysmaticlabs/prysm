package slashingprotection

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

var (
	ErrSlashableAttestation       = errors.New("attempted an attestation rejected by local slashing protection")
	ErrRemoteSlashableAttestation = errors.New("attempted an attestation rejected by remote slashing protection")
)

func (s *Service) IsSlashableAttestation(
	ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte, domain *ethpb.DomainResponse,
) error {
	metricsKey := fmt.Sprintf("%#x", pubKey[:])
	signingRoot, err := helpers.ComputeSigningRoot(indexedAtt.Data, domain.SignatureDomain)
	if err != nil {
		return err
	}
	s.attestingHistoryByPubKeyLock.RLock()
	attesterHistory, ok := s.attesterHistoryByPubKey[pubKey]
	s.attestingHistoryByPubKeyLock.RUnlock()
	if !ok {
		return fmt.Errorf("could not get local slashing protection data for validator %#x", pubKey)
	}
	// Check if the attestation is slashable by local and remote slashing protection
	if s.remoteProtector != nil && s.remoteProtector.IsSlashableAttestation(ctx, indexedAtt) {
		remoteSlashableAttestationsTotal.WithLabelValues(metricsKey).Inc()
		return ErrRemoteSlashableAttestation
	}
	if isNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	) {
		localSlashableAttestationsTotal.WithLabelValues(metricsKey).Inc()
		return ErrSlashableAttestation
	}
	attesterHistory = markAttestationForTargetEpoch(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	)
	s.attestingHistoryByPubKeyLock.Lock()
	s.attesterHistoryByPubKey[pubKey] = attesterHistory
	s.attestingHistoryByPubKeyLock.Unlock()
	return nil
}

// isNewAttSlashable uses the attestation history to determine if an attestation of sourceEpoch
// and targetEpoch would be slashable. It can detect double, surrounding, and surrounded votes.
func isNewAttSlashable(ctx context.Context, history kv.EncHistoryData, sourceEpoch, targetEpoch uint64, signingRoot [32]byte) bool {
	if history == nil {
		return false
	}
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	// Previously pruned, we should return false.
	latestEpochWritten, err := history.GetLatestEpochWritten(ctx)
	if err != nil {
		log.WithError(err).Error("Could not get latest epoch written from encapsulated data")
		return false
	}

	if latestEpochWritten >= wsPeriod && targetEpoch <= latestEpochWritten-wsPeriod { //Underflow protected older then weak subjectivity check.
		return false
	}

	// Check if there has already been a vote for this target epoch.
	hd, err := history.GetTargetData(ctx, targetEpoch)
	if err != nil {
		log.WithError(err).Errorf("Could not get target data for target epoch: %d", targetEpoch)
		return false
	}
	if !hd.IsEmpty() && !bytes.Equal(signingRoot[:], hd.SigningRoot) {
		return true
	}

	// Check if the new attestation would be surrounding another attestation.
	for i := sourceEpoch; i <= targetEpoch; i++ {
		// Unattested for epochs are marked as (*kv.HistoryData)(nil).
		historyBoundary := safeTargetToSource(ctx, history, i)
		if historyBoundary.IsEmpty() {
			continue
		}
		if historyBoundary.Source > sourceEpoch {
			return true
		}
	}

	// Check if the new attestation is being surrounded.
	for i := targetEpoch; i <= latestEpochWritten; i++ {
		h := safeTargetToSource(ctx, history, i)
		if h.IsEmpty() {
			continue
		}
		if h.Source < sourceEpoch {
			return true
		}
	}

	return false
}

// markAttestationForTargetEpoch returns the modified attestation history with the passed-in epochs marked
// as attested for. This is done to prevent the validator client from signing any slashable attestations.
func markAttestationForTargetEpoch(ctx context.Context, history kv.EncHistoryData, sourceEpoch, targetEpoch uint64, signingRoot [32]byte) kv.EncHistoryData {
	if history == nil {
		return nil
	}
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	latestEpochWritten, err := history.GetLatestEpochWritten(ctx)
	if err != nil {
		log.WithError(err).Error("Could not get latest epoch written from encapsulated data")
		return nil
	}
	if targetEpoch > latestEpochWritten {
		// If the target epoch to mark is ahead of latest written epoch, override the old targets and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := latestEpochWritten + wsPeriod
		for i := latestEpochWritten + 1; i < targetEpoch && i <= maxToWrite; i++ {
			history, err = history.SetTargetData(ctx, i%wsPeriod, &kv.HistoryData{Source: params.BeaconConfig().FarFutureEpoch})
			if err != nil {
				log.WithError(err).Error("Could not set target to the encapsulated data")
				return nil
			}
		}
		history, err = history.SetLatestEpochWritten(ctx, targetEpoch)
		if err != nil {
			log.WithError(err).Error("Could not set latest epoch written to the encapsulated data")
			return nil
		}
	}
	history, err = history.SetTargetData(ctx, targetEpoch%wsPeriod, &kv.HistoryData{Source: sourceEpoch, SigningRoot: signingRoot[:]})
	if err != nil {
		log.WithError(err).Error("Could not set target to the encapsulated data")
		return nil
	}
	return history
}

// safeTargetToSource makes sure the epoch accessed is within bounds, and if it's not it at
// returns the "default" nil value.
func safeTargetToSource(ctx context.Context, history kv.EncHistoryData, targetEpoch uint64) *kv.HistoryData {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	latestEpochWritten, err := history.GetLatestEpochWritten(ctx)
	if err != nil {
		log.WithError(err).Error("Could not get latest epoch written from encapsulated data")
		return nil
	}
	if targetEpoch > latestEpochWritten {
		return nil
	}
	if latestEpochWritten >= wsPeriod && targetEpoch < latestEpochWritten-wsPeriod { //Underflow protected older then weak subjectivity check.
		return nil
	}
	hd, err := history.GetTargetData(ctx, targetEpoch%wsPeriod)
	if err != nil {
		log.WithError(err).Errorf("Could not get target data for target epoch: %d", targetEpoch)
		return nil
	}
	return hd
}
