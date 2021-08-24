package kv

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

// deprecatedHistoryData stores the needed data to confirm if an attestation is slashable
// or repeated.
type deprecatedHistoryData struct {
	Source      types.Epoch
	SigningRoot []byte
}

// deprecatedEncodedAttestingHistory encapsulated history data.
type deprecatedEncodedAttestingHistory []byte

func (dh deprecatedEncodedAttestingHistory) assertSize() error {
	if dh == nil || len(dh) < minimalSize {
		return fmt.Errorf("encapsulated data size: %d is smaller then minimal size: %d", len(dh), minimalSize)
	}
	if (len(dh)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(dh), historySize)
	}
	return nil
}

func (dhd *deprecatedHistoryData) isEmpty() bool {
	if dhd == (*deprecatedHistoryData)(nil) {
		return true
	}
	if dhd.Source == params.BeaconConfig().FarFutureEpoch {
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
func newDeprecatedAttestingHistory(target types.Epoch) deprecatedEncodedAttestingHistory {
	relativeTarget := target % params.BeaconConfig().WeakSubjectivityPeriod
	historyDataSize := (relativeTarget + 1) * historySize
	arraySize := latestEpochWrittenSize + historyDataSize
	en := make(deprecatedEncodedAttestingHistory, arraySize)
	enc := en
	ctx := context.Background()
	var err error
	for i := types.Epoch(0); i <= target%params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		enc, err = enc.setTargetData(ctx, i, emptyHistoryData())
		if err != nil {
			log.WithError(err).Error("Failed to set empty target data")
		}
	}
	return enc
}

func (dh deprecatedEncodedAttestingHistory) getLatestEpochWritten(ctx context.Context) (types.Epoch, error) {
	if err := dh.assertSize(); err != nil {
		return 0, err
	}
	return types.Epoch(bytesutil.FromBytes8(dh[:latestEpochWrittenSize])), nil
}

func (dh deprecatedEncodedAttestingHistory) setLatestEpochWritten(
	ctx context.Context,
	latestEpochWritten types.Epoch,
) (deprecatedEncodedAttestingHistory, error) {
	if err := dh.assertSize(); err != nil {
		return nil, err
	}
	copy(dh[:latestEpochWrittenSize], bytesutil.EpochToBytesLittleEndian(latestEpochWritten))
	return dh, nil
}

func (dh deprecatedEncodedAttestingHistory) getTargetData(ctx context.Context, target types.Epoch) (*deprecatedHistoryData, error) {
	if err := dh.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to read target epoch from.
	// Modulus of target epoch X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(dh)) < uint64(cursor+historySize) {
		return nil, nil
	}
	history := &deprecatedHistoryData{}
	history.Source = types.Epoch(bytesutil.FromBytes8(dh[cursor : cursor+sourceSize]))
	sr := make([]byte, 32)
	copy(sr, dh[cursor+sourceSize:cursor+historySize])
	history.SigningRoot = sr
	return history, nil
}

func (dh deprecatedEncodedAttestingHistory) setTargetData(
	ctx context.Context,
	target types.Epoch,
	historyData *deprecatedHistoryData,
) (deprecatedEncodedAttestingHistory, error) {
	if err := dh.assertSize(); err != nil {
		return nil, err
	}
	// Cursor for the location to write target epoch to.
	// Modulus of target epoch  X weak subjectivity period in order to have maximum size to the encapsulated data array.
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize

	if uint64(len(dh)) < uint64(cursor+historySize) {
		ext := make([]byte, uint64(cursor+historySize)-uint64(len(dh)))
		dh = append(dh, ext...)
	}
	copy(dh[cursor:cursor+sourceSize], bytesutil.EpochToBytesLittleEndian(historyData.Source))
	copy(dh[cursor+sourceSize:cursor+sourceSize+signingRootSize], historyData.SigningRoot)

	return dh, nil
}
