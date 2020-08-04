package client

import (
	"context"
	"errors"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var failedPreAttSignLocalErr = "attempted to make slashable attestation, rejected by local slashing protection"
var failedPreAttSignExternalErr = "attempted to make slashable attestation, rejected by external slasher service"
var failedPostAttSignExternalErr = "external slasher service detected a submitted slashable attestation"

func (v *validator) preAttSignValidations(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	if featureconfig.Get().LocalProtection {
		v.attesterHistoryByPubKeyLock.RLock()
		attesterHistory, ok := v.attesterHistoryByPubKey[pubKey]
		v.attesterHistoryByPubKeyLock.RUnlock()
		if !ok {
			return nil
		}
		if isNewAttSlashable(attesterHistory, indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPreAttSignLocalErr)
		}
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CheckAttestationSafety(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPreAttSignExternalErr)
		}
	}
	return nil
}

func (v *validator) postAttSignUpdate(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	if featureconfig.Get().LocalProtection {
		v.attesterHistoryByPubKeyLock.Lock()
		attesterHistory := v.attesterHistoryByPubKey[pubKey]
		attesterHistory = markAttestationForTargetEpoch(attesterHistory, indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch)
		v.attesterHistoryByPubKey[pubKey] = attesterHistory
		v.attesterHistoryByPubKeyLock.Unlock()
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CommitAttestation(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPostAttSignExternalErr)
		}
	}
	return nil
}

// isNewAttSlashable uses the attestation history to determine if an attestation of sourceEpoch
// and targetEpoch would be slashable. It can detect double, surrounding, and surrounded votes.
func isNewAttSlashable(history *slashpb.AttestationHistory, sourceEpoch uint64, targetEpoch uint64) bool {
	if history == nil {
		return false
	}
	farFuture := params.BeaconConfig().FarFutureEpoch
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	// Previously pruned, we should return false.
	if int(targetEpoch) <= int(history.LatestEpochWritten)-int(wsPeriod) {
		return false
	}

	// Check if there has already been a vote for this target epoch.
	if safeTargetToSource(history, targetEpoch) != farFuture {
		return true
	}

	// Check if the new attestation would be surrounding another attestation.
	for i := sourceEpoch; i <= targetEpoch; i++ {
		// Unattested for epochs are marked as FAR_FUTURE_EPOCH.
		if safeTargetToSource(history, i) == farFuture {
			continue
		}
		if history.TargetToSource[i%wsPeriod] > sourceEpoch {
			return true
		}
	}

	// Check if the new attestation is being surrounded.
	for i := targetEpoch; i <= history.LatestEpochWritten; i++ {
		if safeTargetToSource(history, i) < sourceEpoch {
			return true
		}
	}

	return false
}

// markAttestationForTargetEpoch returns the modified attestation history with the passed-in epochs marked
// as attested for. This is done to prevent the validator client from signing any slashable attestations.
func markAttestationForTargetEpoch(history *slashpb.AttestationHistory, sourceEpoch uint64, targetEpoch uint64) *slashpb.AttestationHistory {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	if targetEpoch > history.LatestEpochWritten {
		// If the target epoch to mark is ahead of latest written epoch, override the old targets and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := history.LatestEpochWritten + wsPeriod
		for i := history.LatestEpochWritten + 1; i < targetEpoch && i <= maxToWrite; i++ {
			history.TargetToSource[i%wsPeriod] = params.BeaconConfig().FarFutureEpoch
		}
		history.LatestEpochWritten = targetEpoch
	}
	history.TargetToSource[targetEpoch%wsPeriod] = sourceEpoch
	return history
}

// safeTargetToSource makes sure the epoch accessed is within bounds, and if it's not it at
// returns the "default" FAR_FUTURE_EPOCH value.
func safeTargetToSource(history *slashpb.AttestationHistory, targetEpoch uint64) uint64 {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	if targetEpoch > history.LatestEpochWritten || int(targetEpoch) < int(history.LatestEpochWritten)-int(wsPeriod) {
		return params.BeaconConfig().FarFutureEpoch
	}
	return history.TargetToSource[targetEpoch%wsPeriod]
}
