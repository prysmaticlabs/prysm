package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var failedAttLocalProtectionErr = "attempted to make slashable attestation, rejected by local slashing protection"
var failedPreAttSignExternalErr = "attempted to make slashable attestation, rejected by external slasher service"
var failedPostAttSignExternalErr = "external slasher service detected a submitted slashable attestation"

func (v *validator) preAttSignValidations(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	ctx, span := trace.StartSpan(ctx, "validator.preAttSignUpdate")
	defer span.End()

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	v.attesterHistoryByPubKeyLock.RLock()
	attesterHistory, ok := v.attesterHistoryByPubKey[pubKey]
	v.attesterHistoryByPubKeyLock.RUnlock()
	if !ok {
		AttestationMapMiss.Inc()
		attesterHistoryMap, err := v.db.AttestationHistoryForPubKeysV2(ctx, [][48]byte{pubKey})
		if err != nil {
			return errors.Wrap(err, "could not get attester history")
		}
		attesterHistory, ok = attesterHistoryMap[pubKey]
		if !ok {
			log.WithField("publicKey", fmtKey).Debug("Could not get local slashing protection data for validator in pre validation")
		}
	} else {
		AttestationMapHit.Inc()
	}
	_, sr, err := v.getDomainAndSigningRoot(ctx, indexedAtt.Data)
	if err != nil {
		log.WithError(err).Error("Could not get domain and signing root from attestation")
		return err
	}
	slashable, err := isNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		sr,
	)
	if err != nil {
		return errors.Wrap(err, "could not check if attestation is slashable")
	}
	if slashable {
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedAttLocalProtectionErr)
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

func (v *validator) postAttSignUpdate(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte, signingRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "validator.postAttSignUpdate")
	defer span.End()

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	v.attesterHistoryByPubKeyLock.Lock()
	defer v.attesterHistoryByPubKeyLock.Unlock()
	attesterHistory, ok := v.attesterHistoryByPubKey[pubKey]
	if !ok {
		AttestationMapMiss.Inc()
		attesterHistoryMap, err := v.db.AttestationHistoryForPubKeysV2(ctx, [][48]byte{pubKey})
		if err != nil {
			return errors.Wrap(err, "could not get attester history")
		}
		attesterHistory, ok = attesterHistoryMap[pubKey]
		if !ok {
			log.WithField("publicKey", fmtKey).Debug("Could not get local slashing protection data for validator in post validation")
		}
	} else {
		AttestationMapHit.Inc()
	}
	slashable, err := isNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	)
	if err != nil {
		return errors.Wrap(err, "could not check if attestation is slashable")
	}
	if slashable {
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedAttLocalProtectionErr)
	}
	newHistory, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(
		ctx,
		attesterHistory,
		indexedAtt.Data.Target.Epoch,
		&kv.HistoryData{
			Source:      indexedAtt.Data.Source.Epoch,
			SigningRoot: signingRoot[:],
		},
	)
	if err != nil {
		return errors.Wrapf(err, "could not mark epoch %d as attested", indexedAtt.Data.Target.Epoch)
	}
	v.attesterHistoryByPubKey[pubKey] = newHistory

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CommitAttestation(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPostAttSignExternalErr)
		}
	}

	// Save source and target epochs to satisfy EIP3076 requirements.
	// The DB methods below will replace the lowest epoch in DB if necessary.
	if err := v.db.SaveLowestSignedSourceEpoch(ctx, pubKey, indexedAtt.Data.Source.Epoch); err != nil {
		return err
	}
	return v.db.SaveLowestSignedTargetEpoch(ctx, pubKey, indexedAtt.Data.Target.Epoch)
}

// isNewAttSlashable uses the attestation history to determine if an attestation of sourceEpoch
// and targetEpoch would be slashable. It can detect double, surrounding, and surrounded votes.
func isNewAttSlashable(
	ctx context.Context,
	history kv.EncHistoryData,
	sourceEpoch,
	targetEpoch uint64,
	signingRoot [32]byte,
) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "isNewAttSlashable")
	defer span.End()

	if history == nil {
		return false, nil
	}
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	// Previously pruned, we should return false.
	latestEpochWritten, err := history.GetLatestEpochWritten(ctx)
	if err != nil {
		log.WithError(err).Error("Could not get latest epoch written from encapsulated data")
		return false, err
	}

	if latestEpochWritten >= wsPeriod && targetEpoch <= latestEpochWritten-wsPeriod { //Underflow protected older then weak subjectivity check.
		return false, nil
	}

	// Check if there has already been a vote for this target epoch.
	hd, err := history.GetTargetData(ctx, targetEpoch)
	if err != nil {
		return false, errors.Wrapf(err, "could not get target data for epoch: %d", targetEpoch)
	}
	if !hd.IsEmpty() && !bytes.Equal(signingRoot[:], hd.SigningRoot) {
		log.WithFields(logrus.Fields{
			"signingRoot":                   fmt.Sprintf("%#x", signingRoot),
			"targetEpoch":                   targetEpoch,
			"previouslyAttestedSigningRoot": fmt.Sprintf("%#x", hd.SigningRoot),
		}).Warn("Attempted to submit a double vote, but blocked by slashing protection")
		return true, nil
	}

	isSurround, err := isSurroundVote(ctx, history, latestEpochWritten, sourceEpoch, targetEpoch)
	if err != nil {
		return false, errors.Wrap(err, "could not check if attestation is surround vote")
	}
	return isSurround, nil
}

func isSurroundVote(
	ctx context.Context,
	history kv.EncHistoryData,
	latestEpochWritten,
	sourceEpoch,
	targetEpoch uint64,
) (bool, error) {
	for i := sourceEpoch; i <= targetEpoch; i++ {
		historicalAtt, err := checkHistoryAtTargetEpoch(ctx, history, latestEpochWritten, i)
		if err != nil {
			return false, errors.Wrapf(err, "could not check historical attestation at target epoch: %d", i)
		}
		if historicalAtt.IsEmpty() {
			continue
		}
		prevTarget := i
		prevSource := historicalAtt.Source
		if surroundingPrevAttestation(prevSource, prevTarget, sourceEpoch, targetEpoch) {
			// Surrounding attestation caught.
			log.WithFields(logrus.Fields{
				"targetEpoch":                   targetEpoch,
				"sourceEpoch":                   sourceEpoch,
				"previouslyAttestedTargetEpoch": prevTarget,
				"previouslyAttestedSourceEpoch": prevSource,
			}).Warn("Attempted to submit a surrounding attestation, but blocked by slashing protection")
			return true, nil
		}
	}

	// Check if the new attestation is being surrounded.
	for i := targetEpoch; i <= latestEpochWritten; i++ {
		historicalAtt, err := checkHistoryAtTargetEpoch(ctx, history, latestEpochWritten, i)
		if err != nil {
			return false, errors.Wrapf(err, "could not check historical attestation at target epoch: %d", i)
		}
		if historicalAtt.IsEmpty() {
			continue
		}
		prevTarget := i
		prevSource := historicalAtt.Source
		if surroundedByPrevAttestation(prevSource, prevTarget, sourceEpoch, targetEpoch) {
			// Surrounded attestation caught.
			log.WithFields(logrus.Fields{
				"targetEpoch":                   targetEpoch,
				"sourceEpoch":                   sourceEpoch,
				"previouslyAttestedTargetEpoch": prevTarget,
				"previouslyAttestedSourceEpoch": prevSource,
			}).Warn("Attempted to submit a surrounded attestation, but blocked by slashing protection")
			return true, nil
		}
	}
	return false, nil
}

func surroundedByPrevAttestation(prevSource, prevTarget, newSource, newTarget uint64) bool {
	return prevSource < newSource && newTarget < prevTarget
}

func surroundingPrevAttestation(prevSource, prevTarget, newSource, newTarget uint64) bool {
	return newSource < prevSource && prevTarget < newTarget
}

// Checks that the difference between the latest epoch written and
// target epoch is greater than or equal to the weak subjectivity period.
func differenceOutsideWeakSubjectivityBounds(latestEpochWritten, targetEpoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	return latestEpochWritten >= wsPeriod && targetEpoch <= latestEpochWritten-wsPeriod
}

// safeTargetToSource makes sure the epoch accessed is within bounds, and if it's not it at
// returns the "default" nil value.
// Returns the actual attesting history at a specified target epoch.
// The response is nil if there was no attesting history at that epoch.
func checkHistoryAtTargetEpoch(
	ctx context.Context,
	history kv.EncHistoryData,
	latestEpochWritten,
	targetEpoch uint64,
) (*kv.HistoryData, error) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	if differenceOutsideWeakSubjectivityBounds(latestEpochWritten, targetEpoch) {
		return nil, nil
	}
	// Ignore target epoch is > latest written.
	if targetEpoch > latestEpochWritten {
		return nil, nil
	}
	historicalAtt, err := history.GetTargetData(ctx, targetEpoch%wsPeriod)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
	}
	return historicalAtt, nil
}
