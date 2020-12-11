package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
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
)

// HistoryData stores the needed data to confirm if an attestation is slashable
// or repeated.
type HistoryData struct {
	Source      uint64
	SigningRoot []byte
}

// EncHistoryData encapsulated history data.
type EncHistoryData []byte

func (hd EncHistoryData) assertSize() error {
	if hd == nil || len(hd) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(hd), minimalSize)
	}
	if (len(hd)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(hd), historySize)
	}
	return nil
}

func (h *HistoryData) IsEmpty() bool {
	if h == (*HistoryData)(nil) {
		return true
	}
	if h.Source == params.BeaconConfig().FarFutureEpoch {
		return true
	}
	return false
}

func emptyHistoryData() *HistoryData {
	h := &HistoryData{Source: params.BeaconConfig().FarFutureEpoch, SigningRoot: bytesutil.PadTo([]byte{}, 32)}
	return h
}

// NewAttestationHistoryArray creates a new encapsulated attestation history byte array
// sized by the latest epoch written.
func NewAttestationHistoryArray(target uint64) EncHistoryData {
	relativeTarget := target % params.BeaconConfig().WeakSubjectivityPeriod
	historyDataSize := (relativeTarget + 1) * historySize
	arraySize := latestEpochWrittenSize + historyDataSize
	en := make(EncHistoryData, arraySize)
	enc := en
	ctx := context.Background()
	var err error
	for i := uint64(0); i <= target%params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		enc, err = enc.SetTargetData(ctx, i, emptyHistoryData())
		if err != nil {
			log.WithError(err).Error("Failed to set empty target data")
		}
	}
	return enc
}

func (hd EncHistoryData) GetLatestEpochWritten(ctx context.Context) (uint64, error) {
	if err := hd.assertSize(); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(hd[:latestEpochWrittenSize]), nil
}

func (hd EncHistoryData) SetLatestEpochWritten(ctx context.Context, latestEpochWritten uint64) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	copy(hd[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return hd, nil
}

func (hd EncHistoryData) GetTargetData(ctx context.Context, target uint64) (*HistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to read target epoch from.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(hd)) < cursor+historySize {
		return nil, nil
	}
	history := &HistoryData{}
	history.Source = bytesutil.FromBytes8(hd[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(sr, hd[cursor+sourceSize:cursor+historySize])
	history.SigningRoot = sr
	return history, nil
}

func (hd EncHistoryData) SetTargetData(ctx context.Context, target uint64, historyData *HistoryData) (EncHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to write target epoch to.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize

	if uint64(len(hd)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(hd)))
		hd = append(hd, ext...)
	}
	copy(hd[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(historyData.Source))
	copy(hd[cursor+sourceSize:cursor+sourceSize+signingRootSize], historyData.SigningRoot)

	return hd, nil
}

// MarkAllAsAttestedSinceLatestWrittenEpoch returns an attesting history with specified target+epoch pairs
// since the latest written epoch up to the incoming attestation's target epoch as attested for.
func MarkAllAsAttestedSinceLatestWrittenEpoch(
	ctx context.Context,
	hist EncHistoryData,
	incomingTarget uint64,
	incomingAtt *HistoryData,
) (EncHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "kv.MarkAllAttestedSinceLastWrittenEpoch")
	defer span.End()

	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	latestEpochWritten, err := hist.GetLatestEpochWritten(ctx)
	if err != nil {
		return EncHistoryData{}, errors.Wrap(err, "could not get latest epoch written from history")
	}
	currentHD := hist
	if incomingTarget > latestEpochWritten {
		// If the target epoch to mark is ahead of latest written epoch, override the old targets and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := latestEpochWritten + wsPeriod
		for i := latestEpochWritten + 1; i < incomingTarget && i <= maxToWrite; i++ {
			newHD, err := hist.SetTargetData(ctx, i%wsPeriod, &HistoryData{
				Source: params.BeaconConfig().FarFutureEpoch,
			})
			if err != nil {
				return EncHistoryData{}, errors.Wrap(err, "could not set target data")
			}
			currentHD = newHD
		}
		newHD, err := currentHD.SetLatestEpochWritten(ctx, incomingTarget)
		if err != nil {
			return EncHistoryData{}, errors.Wrap(err, "could not set latest epoch written")
		}
		currentHD = newHD
	}
	newHD, err := currentHD.SetTargetData(ctx, incomingTarget%wsPeriod, &HistoryData{
		Source:      incomingAtt.Source,
		SigningRoot: incomingAtt.SigningRoot,
	})
	if err != nil {
		return EncHistoryData{}, errors.Wrap(err, "could not set target data")
	}
	return newHD, nil
}
