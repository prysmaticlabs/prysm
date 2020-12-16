package attestinghistory

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "hist")

// IsNewAttSlashable uses the attestation history to determine if an attestation of sourceEpoch
// and targetEpoch would be slashable. It can detect double, surrounding, and surrounded votes.
func IsNewAttSlashable(
	ctx context.Context,
	history kv.EncHistoryData,
	sourceEpoch,
	targetEpoch uint64,
	signingRoot [32]byte,
) (bool, error) {
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
	if !hd.IsEmpty() {
		signingRootIsDifferent := bytes.Equal(hd.SigningRoot, params.BeaconConfig().ZeroHash[:]) ||
			!bytes.Equal(hd.SigningRoot, signingRoot[:])
		if signingRootIsDifferent {
			log.WithFields(logrus.Fields{
				"signingRoot":                   fmt.Sprintf("%#x", signingRoot),
				"targetEpoch":                   targetEpoch,
				"previouslyAttestedSigningRoot": fmt.Sprintf("%#x", hd.SigningRoot),
			}).Warn("Attempted to submit a double vote, but blocked by slashing protection")
			return true, nil
		}
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
