package slashingprotection

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
	signingRoot [32]byte,
) (bool, error) {
	if indexedAtt == nil || indexedAtt.Data == nil {
		return false, errors.New("received nil attestation")
	}
	s.attestingHistoryByPubKeyLock.Lock()
	defer s.attestingHistoryByPubKeyLock.Unlock()
	attesterHistory, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return false, fmt.Errorf("could not get local slashing protection data for validator %#x", pubKey)
	}
	if attesterHistory == nil {
		return false, fmt.Errorf("nil attester history found for public key %#x", pubKey)
	}
	latestEpochWritten, err := attesterHistory.GetLatestEpochWritten(ctx)
	if err != nil {
		return false, errors.Wrapf(err, "could not get latest epoch written for pubkey %#x", pubKey)
	}
	// An attestation older than the weak subjectivity is not slashable, we should just return false.
	if isOlderThanWeakSubjectivity(ctx, latestEpochWritten, indexedAtt.Data.Target.Epoch) {
		return false, nil
	}
	doubleVote, err := isDoubleVote(ctx, attesterHistory, indexedAtt.Data.Target.Epoch, signingRoot)
	if err != nil {
		return false, errors.Wrapf(err, "could not check if pubkey is attempting a double vote %#x", pubKey)
	}
	surroundVote, err := isSurroundVote(
		ctx,
		attesterHistory,
		indexedAtt.Data.Target.Epoch,
		indexedAtt.Data.Target.Epoch,
		indexedAtt.Data.Source.Epoch,
	)
	if err != nil {
		return false, errors.Wrapf(err, "could not check if pubkey is attempting a surround vote %#x", pubKey)
	}
	// If an attestation is a double vote or a surround vote, it is slashable.
	if doubleVote || surroundVote {
		localSlashableAttestationsTotal.Inc()
		return true, nil
	}
	// We update the attester history with new values.
	newAttesterHistory, err := attesterHistory.UpdateHistoryForAttestation(
		ctx,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	)
	if err != nil {
		return false, errors.Wrap(err, "could not update attesting history data")
	}

	// We update our in-memory map of attester history.
	s.attesterHistoryByPubKey[pubKey] = newAttesterHistory

	// We check the attestation with respect to remote slashing protection if enabled.
	if s.remoteProtector != nil {
		slashable, err := s.remoteProtector.IsSlashableAttestation(ctx, indexedAtt, pubKey, signingRoot)
		if err != nil {
			return false, err
		}
		remoteSlashableAttestationsTotal.Inc()
		return slashable, nil
	}
	return false, nil
}

func isOlderThanWeakSubjectivity(ctx context.Context, latestEpochWritten, targetEpoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	return latestEpochWritten >= wsPeriod && targetEpoch <= latestEpochWritten-wsPeriod
}

func isDoubleVote(ctx context.Context, history kv.EncHistoryData, targetEpoch uint64, signingRoot [32]byte) (bool, error) {
	// Check if there has already been a vote for this target epoch.
	hd, err := history.GetTargetData(ctx, targetEpoch)
	if err != nil {
		return false, errors.Wrapf(err, "could not get data for target epoch: %d", targetEpoch)
	}
	return !hd.IsEmpty() && !bytes.Equal(signingRoot[:], hd.SigningRoot), nil
}

func isSurroundVote(
	ctx context.Context,
	history kv.EncHistoryData,
	latestEpochWritten,
	sourceEpoch,
	targetEpoch uint64,
) (bool, error) {
	// Check if the new attestation would be surrounding another attestation.
	for i := sourceEpoch; i <= targetEpoch; i++ {
		// Unattested for epochs are marked as (*kv.HistoryData)(nil).
		historyBoundary := safeTargetToSource(ctx, history, i)
		if historyBoundary.IsEmpty() {
			continue
		}
		if historyBoundary.Source > sourceEpoch {
			return true, nil
		}
	}

	// Check if the new attestation is being surrounded.
	for i := targetEpoch; i <= latestEpochWritten; i++ {
		h := safeTargetToSource(ctx, history, i)
		if h.IsEmpty() {
			continue
		}
		if h.Source < sourceEpoch {
			return true, nil
		}
	}
	return false, nil
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
