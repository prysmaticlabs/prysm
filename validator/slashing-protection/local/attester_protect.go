package local

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/params"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
	attestinghistory "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
	"github.com/sirupsen/logrus"
)

// IsSlashableAttestation determines if an incoming attestation is slashable
// according to local protection. Then, if the attestation successfully passes
// checks, we update our local attesting history accordingly.
func (s *Service) IsSlashableAttestation(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	pubKey [48]byte,
	signingRoot [32]byte,
) (bool, error) {
	if indexedAtt == nil || indexedAtt.Data == nil {
		return false, errors.New("received nil attestation")
	}
	lock := mputil.NewMultilock(string(pubKey[:]))
	lock.Lock()
	defer lock.Unlock()
	history, minAtt, err := s.validatorDB.AttestationHistoryForPubKey(ctx, pubKey)
	if err != nil {
		return false, fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	if history == nil {
		return false, nil
	}
	latestEpochWritten, err := attestinghistory.GetLatestEpochWritten(history)
	if err != nil {
		return false, errors.Wrapf(err, "could not get latest epoch written for pubkey %#x", pubKey)
	}
	// An attestation older than the weak subjectivity cant be detected with our protection db, we should just return false.
	if differenceOutsideWeakSubjectivityBounds(latestEpochWritten, indexedAtt.Data.Target.Epoch) {
		return false, nil
	}
	lowerThenMin := isLowerThenMin(&minAtt, indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch)
	doubleVote, err := isDoubleVote(history, indexedAtt.Data.Target.Epoch, signingRoot)
	if err != nil {
		return false, errors.Wrapf(err, "could not check if pubkey is attempting a double vote %#x", pubKey)
	}
	surroundVote, err := isSurroundVote(
		ctx,
		history,
		latestEpochWritten,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
	)
	if err != nil {
		return false, errors.Wrapf(err, "could not check if pubkey is attempting a surround vote %#x", pubKey)
	}
	// If an attestation is a double vote or a surround vote, it is slashable.
	if lowerThenMin || doubleVote || surroundVote {
		slashingprotection.LocalSlashableAttestationsTotal.Inc()
		return true, nil
	}
	// We update the attester history with new values.
	newAttesterHistory, err := attestinghistory.MarkAllAsAttestedSinceLatestWrittenEpoch(
		ctx,
		history,
		&attestinghistory.HistoricalAttestation{
			Source:      indexedAtt.Data.Source.Epoch,
			Target:      indexedAtt.Data.Target.Epoch,
			SigningRoot: signingRoot[:],
		},
	)
	if err != nil {
		return false, errors.Wrap(err, "could not update attesting history data")
	}

	if err := s.validatorDB.SaveAttestationHistoryForPubKey(ctx, pubKey, newAttesterHistory); err != nil {
		return false, err
	}
	return false, nil
}

// Checks that the difference between the latest epoch written and
// target epoch is greater than or equal to the weak subjectivity period.
func differenceOutsideWeakSubjectivityBounds(latestEpochWritten, targetEpoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	return latestEpochWritten >= wsPeriod && targetEpoch <= latestEpochWritten-wsPeriod
}

func isLowerThenMin(
	minAtt *attestinghistory.MinAttestation,
	sourceEpoch uint64,
	targetEpoch uint64,
) bool {
	if minAtt == nil {
		return false
	}
	// Check if source of target epoch of the attestation is lower then the minimum.
	if sourceEpoch < minAtt.Source || targetEpoch < minAtt.Target {
		return true
	}
	return false
}

func isDoubleVote(
	history attestinghistory.History,
	targetEpoch uint64,
	signingRoot [32]byte,
) (bool, error) {
	// Check if there has already been a vote for this target epoch.
	historicalAttestation, err := attestinghistory.HistoricalAttestationAtTargetEpoch(history, targetEpoch)
	if err != nil {
		return false, errors.Wrapf(err, "could not get data for target epoch: %d", targetEpoch)
	}
	isEmpty := attestinghistory.IsEmptyHistoricalAttestation(historicalAttestation)
	if !isEmpty && !bytes.Equal(signingRoot[:], historicalAttestation.SigningRoot) {
		log.WithFields(logrus.Fields{
			"signingRoot":                   fmt.Sprintf("%#x", signingRoot),
			"targetEpoch":                   targetEpoch,
			"previouslyAttestedSigningRoot": fmt.Sprintf("%#x", historicalAttestation.SigningRoot),
		}).Warn("Attempted to submit a double vote, but blocked by slashing protection")
		return true, nil
	}
	return false, nil
}

func isSurroundVote(
	ctx context.Context,
	history attestinghistory.History,
	latestEpochWritten,
	sourceEpoch,
	targetEpoch uint64,
) (bool, error) {
	for i := sourceEpoch; i <= targetEpoch; i++ {
		historicalAtt, err := checkHistoryAtTargetEpoch(ctx, history, latestEpochWritten, i)
		if err != nil {
			return false, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
		}
		if attestinghistory.IsEmptyHistoricalAttestation(historicalAtt) {
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
			return false, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
		}
		if attestinghistory.IsEmptyHistoricalAttestation(historicalAtt) {
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

// Returns the actual attesting history at a specified target epoch.
// The response is nil if there was no attesting history at that epoch.
func checkHistoryAtTargetEpoch(
	ctx context.Context,
	history attestinghistory.History,
	latestEpochWritten,
	targetEpoch uint64,
) (*attestinghistory.HistoricalAttestation, error) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	if differenceOutsideWeakSubjectivityBounds(latestEpochWritten, targetEpoch) {
		return nil, nil
	}
	// Ignore target epoch is > latest written.
	if targetEpoch > latestEpochWritten {
		return nil, nil
	}
	historicalAtt, err := attestinghistory.HistoricalAttestationAtTargetEpoch(history, targetEpoch%wsPeriod)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
	}
	return historicalAtt, nil
}

func surroundedByPrevAttestation(prevSource, prevTarget, newSource, newTarget uint64) bool {
	return prevSource < newSource && newTarget < prevTarget
}

func surroundingPrevAttestation(prevSource, prevTarget, newSource, newTarget uint64) bool {
	return newSource < prevSource && prevTarget < newTarget
}
