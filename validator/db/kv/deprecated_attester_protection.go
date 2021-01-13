package kv

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const (
	// The size of each data entry in bytes for the Source epoch (8 bytes) and signing root (32 bytes).
	uint64Size             = 8
	latestEpochWrittenSize = uint64Size
	targetSize             = uint64Size
	sourceSize             = uint64Size
	signingRootSize        = 32
	historySize            = targetSize + sourceSize + signingRootSize
	minimalSize            = latestEpochWrittenSize
)

// deprecatedHistoryData stores the needed data to confirm if an attestation is slashable
// or repeated.
type deprecatedHistoryData struct {
	Source      uint64
	SigningRoot []byte
}

// deprecatedEncodedAttestingHistory encapsulated history data.
type deprecatedEncodedAttestingHistory []byte

func (hd deprecatedEncodedAttestingHistory) assertSize() error {
	if hd == nil || len(hd) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(hd), minimalSize)
	}
	if (len(hd)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(hd), historySize)
	}
	return nil
}

func (h *deprecatedHistoryData) isEmpty() bool {
	if h == (*deprecatedHistoryData)(nil) {
		return true
	}
	if h.Source == params.BeaconConfig().FarFutureEpoch {
		return true
	}
	return false
}

func emptyHistoryData() *deprecatedHistoryData {
	h := &deprecatedHistoryData{Source: params.BeaconConfig().FarFutureEpoch, SigningRoot: bytesutil.PadTo([]byte{}, 32)}
	return h
}

// newDeprecatedAttestingHistory creates a new encapsulated attestation history byte array
// sized by the latest epoch written.
func newDeprecatedAttestingHistory(target uint64) deprecatedEncodedAttestingHistory {
	relativeTarget := target % params.BeaconConfig().WeakSubjectivityPeriod
	historyDataSize := (relativeTarget + 1) * historySize
	arraySize := latestEpochWrittenSize + historyDataSize
	en := make(deprecatedEncodedAttestingHistory, arraySize)
	enc := en
	ctx := context.Background()
	var err error
	for i := uint64(0); i <= target%params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		enc, err = enc.setTargetData(ctx, i, emptyHistoryData())
		if err != nil {
			log.WithError(err).Error("Failed to set empty Target data")
		}
	}
	return enc
}

func (hd deprecatedEncodedAttestingHistory) getLatestEpochWritten(ctx context.Context) (uint64, error) {
	if err := hd.assertSize(); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(hd[:latestEpochWrittenSize]), nil
}

func (hd deprecatedEncodedAttestingHistory) setLatestEpochWritten(ctx context.Context, latestEpochWritten uint64) (deprecatedEncodedAttestingHistory, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	copy(hd[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return hd, nil
}

func (hd deprecatedEncodedAttestingHistory) getTargetData(ctx context.Context, target uint64) (*deprecatedHistoryData, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to read Target epoch from.
	// Modulus of Target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(hd)) < cursor+historySize {
		return nil, nil
	}
	history := &deprecatedHistoryData{}
	history.Source = bytesutil.FromBytes8(hd[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(sr, hd[cursor+sourceSize:cursor+historySize])
	history.SigningRoot = sr
	return history, nil
}

func (hd deprecatedEncodedAttestingHistory) setTargetData(ctx context.Context, target uint64, historyData *deprecatedHistoryData) (deprecatedEncodedAttestingHistory, error) {
	if err := hd.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to write Target epoch to.
	cursorToWrite := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize
	if uint64(len(hd)) < cursorToWrite+historySize {
		start := uint64(len(hd))
		ext := make([]byte, cursorToWrite+historySize-uint64(len(hd)))
		hd = append(hd, ext...)
		// We need to mark the epochs in between the latest written one and the newest one
		// we are writing as not attested by setting the Source epoch to FAR_FUTURE_EPOCH.
		for i := start; i < uint64(len(hd)); i += historySize {
			copy(
				hd[i:i+sourceSize],
				bytesutil.Uint64ToBytesLittleEndian(params.BeaconConfig().FarFutureEpoch),
			)
			copy(hd[i+sourceSize:i+sourceSize+signingRootSize], make([]byte, 32))
		}
	}
	copy(hd[cursorToWrite:cursorToWrite+sourceSize], bytesutil.Uint64ToBytesLittleEndian(historyData.Source))
	copy(hd[cursorToWrite+sourceSize:cursorToWrite+sourceSize+signingRootSize], historyData.SigningRoot)
	return hd, nil
}
