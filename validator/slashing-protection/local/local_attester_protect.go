package local

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection"
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
	lock := mputil.NewMultilock(fmt.Sprintf("%x", pubKey))
	lock.Lock()
	defer lock.Unlock()
	attesterHistory, ok := s.attesterHistoryByPubKey[pubKey]
	if !ok {
		return false, fmt.Errorf("no attesting history found for pubkey %#x", pubKey)
	}
	if attesterHistory == nil {
		return false, fmt.Errorf("nil attester history found for public key %#x", pubKey)
	}
	latestEpochWritten, err := attesterHistory.GetLatestEpochWritten(ctx)
	if err != nil {
		return false, errors.Wrapf(err, "could not get latest epoch written for pubkey %#x", pubKey)
	}
	// An attestation older than the weak subjectivity is not slashable, we should just return false.
	if differenceOutsideWeakSubjectivityBounds(latestEpochWritten, indexedAtt.Data.Target.Epoch) {
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
		slashingprotection.LocalSlashableAttestationsTotal.Inc()
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

	log.Infof("Updating store for pubkey %#x", pubKey)
	// We update our in-memory map of attester history.
	s.attesterHistoryByPubKey[pubKey] = newAttesterHistory
	log.Infof("Updated store for pubkey %#x", pubKey)
	return false, nil
}

// Checks that the difference between the latest epoch written and
// target epoch is greater than or equal to the weak subjectivity period.
func differenceOutsideWeakSubjectivityBounds(latestEpochWritten, targetEpoch uint64) bool {
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
		historyAtTarget, err := checkHistoryAtTargetEpoch(ctx, history, latestEpochWritten, i)
		if err != nil {
			return false, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
		}
		if historyAtTarget == nil || historyAtTarget.IsEmpty() {
			continue
		}
		if historyAtTarget.Source > sourceEpoch {
			// Surrounding attestation caught.
			return true, nil
		}
	}

	// Check if the new attestation is being surrounded.
	for i := targetEpoch; i <= latestEpochWritten; i++ {
		historyAtTarget, err := checkHistoryAtTargetEpoch(ctx, history, latestEpochWritten, i)
		if err != nil {
			return false, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
		}
		if historyAtTarget == nil || historyAtTarget.IsEmpty() {
			continue
		}
		if historyAtTarget.Source < sourceEpoch {
			// Surrounded attestation caught.
			return true, nil
		}
	}
	return false, nil
}

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
	historyData, err := history.GetTargetData(ctx, targetEpoch%wsPeriod)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get target data for target epoch: %d", targetEpoch)
	}
	return historyData, nil
}
