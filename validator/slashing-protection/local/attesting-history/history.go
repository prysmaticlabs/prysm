package attestinghistory

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
)

const (
	// The size of each data entry in bytes for the source epoch (8 bytes) and signing root (32 bytes).
	uint64Size             = 8
	latestEpochWrittenSize = uint64Size
	targetSize             = uint64Size
	sourceSize             = uint64Size
	signingRootSize        = 32
	historySize            = targetSize + sourceSize + signingRootSize
	minimalSize            = latestEpochWrittenSize
	// Key prefix to minimal attestation source epoch in attestation bucket.
	minimalAttestationSourceEpochKeyPrefix = "minimal-attestation-source-epoch"
	// Key prefix to minimal attestation target epoch in attestation bucket.
	minimalAttestationTargetEpochKeyPrefix = "minimal-attestation-target-epoch"
)

// Structure that represents minimal attestation source and target that are allowed to be signed.
type MinAttestation struct {
	Source uint64
	Target uint64
}

type HistoricalAttestation struct {
	Source      uint64
	Target      uint64
	SigningRoot []byte
}

// History is a type alias to efficiently store historical attestation
// information via methods defined in this package. Methods are pure
// functions which return copies of History.
type History []byte

// New attestation history sized to the target epoch specified modulo WEAK_SUBJECTIVITY_PERIOD.
func New(targetEpoch uint64) History {
	relativeTarget := targetEpoch % params.BeaconConfig().WeakSubjectivityPeriod
	historyDataSize := (relativeTarget + 1) * historySize
	arraySize := latestEpochWrittenSize + historyDataSize
	initialHist := make(History, arraySize)
	currentHist := initialHist
	for epoch := uint64(0); epoch <= targetEpoch%params.BeaconConfig().WeakSubjectivityPeriod; epoch++ {
		historicalAtt := &HistoricalAttestation{
			Target:      epoch,
			Source:      params.BeaconConfig().FarFutureEpoch,
			SigningRoot: make([]byte, 32),
		}
		newHist, err := MarkAsAttested(currentHist, historicalAtt)
		if err != nil {
			log.WithError(err).Error("Failed to set empty target data")
		}
		currentHist = newHist
	}
	return currentHist
}

func GetLatestEpochWritten(hist History) (uint64, error) {
	if err := assertSize(hist); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(hist[:latestEpochWrittenSize]), nil
}

func SetLatestEpochWritten(hist History, latestEpochWritten uint64) (History, error) {
	if err := assertSize(hist); err != nil {
		return nil, err
	}
	newHist := make([]byte, len(hist))
	copy(newHist[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return newHist, nil
}

func HistoricalAttestationAtTargetEpoch(hist History, target uint64) (*HistoricalAttestation, error) {
	if err := assertSize(hist); err != nil {
		return nil, err
	}
	// Cursor for the location to read target epoch from.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(hist)) < cursor+historySize {
		return nil, nil
	}
	histAttestation := &HistoricalAttestation{}
	histAttestation.Source = bytesutil.FromBytes8(hist[cursor : cursor+sourceSize])
	histAttestation.Target = target
	sr := make([]byte, 32)
	copy(sr, hist[cursor+sourceSize:cursor+historySize])
	histAttestation.SigningRoot = sr
	return histAttestation, nil
}

func MarkAsAttested(hist History, incomingAtt *HistoricalAttestation) (History, error) {
	if err := assertSize(hist); err != nil {
		return nil, err
	}
	newHist := hist
	// Cursor for the location to write target epoch into.
	// Modulus of target epoch by WEAK_SUBJECTIVITY_PERIOD.
	cursor := latestEpochWrittenSize + (incomingAtt.Target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize
	if uint64(len(newHist)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(newHist)))
		newHist = append(newHist, ext...)
	}
	copy(newHist[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(incomingAtt.Source))
	copy(newHist[cursor+sourceSize:cursor+sourceSize+signingRootSize], incomingAtt.SigningRoot)
	return newHist, nil
}

func IsEmptyHistoricalAttestation(histAtt *HistoricalAttestation) bool {
	if histAtt == nil {
		return true
	}
	return histAtt.Source == params.BeaconConfig().FarFutureEpoch
}

// MarkAllAsAttestedSinceLatestWrittenEpoch returns an attesting history with specified target+epoch pairs
// since the latest written epoch up to the incoming attestation's target epoch as attested for.
func MarkAllAsAttestedSinceLatestWrittenEpoch(
	ctx context.Context,
	hist History,
	incomingAtt *HistoricalAttestation,
) (History, error) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	latestEpochWritten, err := GetLatestEpochWritten(hist)
	if err != nil {
		return History{}, errors.Wrap(err, "could not get latest epoch written from history")
	}
	currentHD := hist
	if incomingAtt.Target > latestEpochWritten {
		// If the target epoch to mark is ahead of latest written epoch, override the old targets and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := latestEpochWritten + wsPeriod
		for i := latestEpochWritten + 1; i < incomingAtt.Target && i <= maxToWrite; i++ {
			newHD, err := MarkAsAttested(hist, &HistoricalAttestation{
				Source: params.BeaconConfig().FarFutureEpoch,
				Target: i % wsPeriod,
			})
			if err != nil {
				return History{}, errors.Wrap(err, "could not set target data")
			}
			currentHD = newHD
		}
		newHD, err := SetLatestEpochWritten(currentHD, incomingAtt.Target)
		if err != nil {
			return History{}, errors.Wrap(err, "could not set latest epoch written")
		}
		currentHD = newHD
	}
	newHD, err := MarkAsAttested(currentHD, &HistoricalAttestation{
		Target:      incomingAtt.Target % wsPeriod,
		Source:      incomingAtt.Source,
		SigningRoot: incomingAtt.SigningRoot,
	})
	if err != nil {
		return History{}, errors.Wrap(err, "could not set target data")
	}
	return newHD, nil
}

func assertSize(hist History) error {
	if hist == nil || len(hist) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(hist), minimalSize)
	}
	if (len(hist)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(hist), historySize)
	}
	return nil
}

// GetMinTargetKey given a public key returns the min source db key.
func GetMinSourceKey(pubKey [48]byte) []byte {
	return append([]byte(minimalAttestationSourceEpochKeyPrefix), pubKey[:]...)
}

// GetMinTargetKey given a public key returns the min target db key.
func GetMinTargetKey(pubKey [48]byte) []byte {
	return append([]byte(minimalAttestationTargetEpochKeyPrefix), pubKey[:]...)
}
