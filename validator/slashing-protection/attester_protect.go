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

// IsSlashableAttestation determines if an incoming attestation is slashable
// according to local protection and remote protection (if enabled). Then, if the attestation
// successfully passes checks, we update our local attesting history accordingly.
func (s *Service) IsSlashableAttestation(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	pubKey [48]byte,
	domain *ethpb.DomainResponse,
) (bool, error) {
	metricsKey := fmt.Sprintf("%#x", pubKey[:])
	signingRoot, err := helpers.ComputeSigningRoot(indexedAtt.Data, domain.SignatureDomain)
	if err != nil {
		return false, err
	}
	s.attestingHistoryByPubKeyLock.Lock()
	defer s.attestingHistoryByPubKeyLock.Unlock()
	attesterHistory, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return false, fmt.Errorf("could not get local slashing protection data for validator %#x", pubKey)
	}
	// Check if the attestation is slashable by local and remote slashing protection
	if s.remoteProtector != nil {
		slashable, err := s.remoteProtector.IsSlashableAttestation(ctx, indexedAtt, pubKey, domain)
		if err != nil {
			return false, err
		}
		remoteSlashableAttestationsTotal.WithLabelValues(metricsKey).Inc()
		return slashable, nil
	}
	if isNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	) {
		localSlashableAttestationsTotal.WithLabelValues(metricsKey).Inc()
		return true, nil
	}
	// We update the attester history with new values.
	attesterHistory, err = attesterHistory.UpdateHistoryForAttestation(
		ctx,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	)
	if err != nil {
		return false, errors.Wrap(err, "could not update attesting history data")
	}
	s.attesterHistoryByPubKey[pubKey] = attesterHistory
	return false, nil
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
