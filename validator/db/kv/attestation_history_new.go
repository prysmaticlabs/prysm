package kv

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
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

// AttestationHistoryNew stores the historical attestation data needed
// for protection of validators.
type AttestationHistoryNew struct {
	TargetToSource     map[uint64]*HistoryData
	LatestEpochWritten uint64
}

// HistoryData stores the needed data to confirm if an attestation is slashable
// or repeated.
type HistoryData struct {
	Source      uint64
	SigningRoot []byte
}

// encapsulated history data
type encHistoryData []byte

func (hd encHistoryData) assertSize() error { // pointer receiver will also work
	if len(hd) == 0 || len(hd) < minimalSize || (len(hd)-minimalSize)%historySize != 0 {
		return fmt.Errorf("encapsulated data size: %d is not a multiple of entry size: %d", len(hd), historySize)
	}
	return nil
}

func newAttestationHistoryArray(target uint64) encHistoryData {
	enc := make(encHistoryData, latestEpochWrittenSize+(target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize+historySize)
	return enc
}

func getLatestEpochWritten(ctx context.Context, data encHistoryData) (uint64, error) {
	if err := data.assertSize(); err != nil {
		return 0, err
	}
	return bytesutil.FromBytes8(data[:latestEpochWrittenSize]), nil
}

func setLatestEpochWritten(ctx context.Context, data encHistoryData, latestEpochWritten uint64) ([]byte, error) {
	if err := data.assertSize(); err != nil {
		return nil, err
	}
	copy(data[:latestEpochWrittenSize], bytesutil.Uint64ToBytesLittleEndian(latestEpochWritten))
	return data, nil
}

func getTargetData(ctx context.Context, data encHistoryData, target uint64) (*HistoryData, error) {
	if err := data.assertSize(); err != nil {
		return nil, err
	}
	cursor := (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize + latestEpochWrittenSize
	if uint64(len(data)) < cursor+historySize {
		return nil, fmt.Errorf("encapsulated data size: %d is smaller then the requested target location: %d", len(data), cursor+historySize)
	}
	history := &HistoryData{}

	history.Source = bytesutil.FromBytes8(data[cursor : cursor+sourceSize])
	sr := make([]byte, 32)
	copy(data[cursor+sourceSize:cursor+historySize], sr)
	history.SigningRoot = sr
	return history, nil
}

func setTargetData(ctx context.Context, data encHistoryData, target uint64, historyData *HistoryData) ([]byte, error) {
	if err := data.assertSize(); err != nil {
		return nil, err
	}
	cursor := latestEpochWrittenSize + (target%params.BeaconConfig().WeakSubjectivityPeriod)*historySize
	if uint64(len(data)) < cursor+historySize {
		ext := make([]byte, cursor+historySize-uint64(len(data)))
		data = append(data, ext...)
	}
	copy(data[cursor:cursor+sourceSize], bytesutil.Uint64ToBytesLittleEndian(historyData.Source))
	copy(data[cursor+sourceSize:cursor+sourceSize+signingRootSize], historyData.SigningRoot)
	return data, nil
}

// AttestationHistoryNewForPubKeys accepts an array of validator public keys and returns a mapping of corresponding attestation history.
func (store *Store) AttestationHistoryNewForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]encHistoryData, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestationHistoryForPubKeys")
	defer span.End()

	if len(publicKeys) == 0 {
		return make(map[[48]byte]encHistoryData), nil
	}

	var err error
	attestationHistoryForVals := make(map[[48]byte]encHistoryData)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for _, key := range publicKeys {
			enc := bucket.Get(key[:])
			var attestationHistory []byte
			if len(enc) == 0 {
				attestationHistory = newAttestationHistoryArray(0)
			} else {
				attestationHistory = enc
				if err != nil {
					return err
				}
			}
			attestationHistoryForVals[key] = attestationHistory
		}
		return nil
	})
	return attestationHistoryForVals, err
}

// SaveAttestationHistoryNewForPubKeys saves the attestation histories for the requested validator public keys.
func (store *Store) SaveAttestationHistoryNewForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]encHistoryData) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveAttestationHistoryForPubKeys")
	defer span.End()

	err := store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(newHistoricAttestationsBucket)
		for pubKey, encodedHistory := range historyByPubKeys {
			if err := bucket.Put(pubKey[:], encodedHistory); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
